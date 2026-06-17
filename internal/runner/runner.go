package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/brainproxy/brainproxy/internal/config"
	"github.com/brainproxy/brainproxy/internal/logger"
	"github.com/brainproxy/brainproxy/internal/models"
	"github.com/brainproxy/brainproxy/internal/proxy"
	"github.com/brainproxy/brainproxy/internal/store"
	"github.com/brainproxy/brainproxy/internal/ws"
)

// RunClaude starts the proxy server in background, then launches claude
// as a child process with ANTHROPIC_BASE_URL pointing to the proxy.
// When claude exits, the proxy shuts down.
func RunClaude(configPath string, claudeArgs []string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.APIKey == "" {
		return fmt.Errorf("api_key is required (run 'brainproxy setup' first)")
	}

	// --- Start proxy server in background ---
	addr := fmt.Sprintf(":%d", cfg.Port)
	srv, err := startProxyServer(cfg, addr)
	if err != nil {
		return err
	}

	// Wait for the proxy to be ready
	if err := waitForPort(cfg.Port, 5*time.Second); err != nil {
		srv.Shutdown(context.Background())
		return fmt.Errorf("proxy failed to start: %w", err)
	}

	proxyURL := fmt.Sprintf("http://localhost:%d", cfg.Port)
	log.Printf("BrainProxy running on %s → %s", proxyURL, cfg.BaseURL)

	// --- Launch claude ---
	exitCode := launchClaude(cfg, proxyURL, claudeArgs)

	// --- Shutdown proxy ---
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	log.Println("Proxy shut down.")

	os.Exit(exitCode)
	return nil // unreachable
}

// startProxyServer builds and starts the HTTP proxy server in a goroutine.
func startProxyServer(cfg *config.Config, addr string) (*http.Server, error) {
	memStore := store.NewMemoryStore(cfg.BufferSize)
	hub := ws.NewHub()
	go hub.Run()

	reqLogger, err := logger.New(cfg.LogDir, cfg.MaxLogFiles)
	if err != nil {
		return nil, fmt.Errorf("init logger: %w", err)
	}

	events := make(chan models.WSEvent, 256)
	go func() {
		for event := range events {
			if event.Type == "request.complete" {
				if re, ok := event.Data.(*models.RequestEvent); ok {
					reqLogger.Save(re)
				}
			}
			data, err := json.Marshal(event)
			if err != nil {
				log.Printf("event marshal error: %v", err)
				continue
			}
			hub.Broadcast(string(data))
		}
	}()

	proxyHandler := proxy.NewHandler(cfg.BaseURL, cfg.APIKey, memStore, events)

	mux := http.NewServeMux()
	mux.Handle("/v1/", proxyHandler)
	mux.HandleFunc("/ws", hub.HandleWS)
	mux.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(memStore.List())
	})
	mux.HandleFunc("/api/events/", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[len("/api/events/"):]
		event := memStore.Get(id)
		if event == nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(event)
	})

	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("proxy server error: %v", err)
		}
	}()

	return srv, nil
}

// launchClaude starts the claude CLI with proxy environment variables.
// Returns the exit code of the claude process.
func launchClaude(cfg *config.Config, proxyURL string, args []string) int {
	// Find claude binary
	claudeBin, err := exec.LookPath("claude")
	if err != nil {
		log.Fatal("claude not found in PATH. Please install Claude Code first.")
	}

	cmd := exec.Command(claudeBin, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Build environment: inherit all current env vars, then override proxy-related ones
	cmd.Env = os.Environ()
	cmd.Env = setEnv(cmd.Env, "ANTHROPIC_BASE_URL", proxyURL)
	cmd.Env = setEnv(cmd.Env, "ANTHROPIC_AUTH_TOKEN", cfg.APIKey)

	log.Printf("Launching Claude Code with proxy (Web UI: %s)", proxyURL)

	// Forward signals to claude
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for sig := range sigCh {
			if cmd.Process != nil {
				cmd.Process.Signal(sig)
			}
		}
	}()

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		log.Printf("claude error: %v", err)
		return 1
	}
	return 0
}

// waitForPort polls until a TCP connection to the port succeeds.
func waitForPort(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for port %d", port)
}

// setEnv sets or replaces an environment variable in a slice of "KEY=VALUE" strings.
func setEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, e := range env {
		if len(e) > len(prefix) && e[:len(prefix)] == prefix {
			env[i] = key + "=" + value
			return env
		}
	}
	return append(env, key+"="+value)
}
