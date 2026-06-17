package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/brainproxy/brainproxy/internal/config"
	"github.com/brainproxy/brainproxy/internal/logger"
	"github.com/brainproxy/brainproxy/internal/models"
	"github.com/brainproxy/brainproxy/internal/proxy"
	"github.com/brainproxy/brainproxy/internal/runner"
	"github.com/brainproxy/brainproxy/internal/setup"
	"github.com/brainproxy/brainproxy/internal/store"
	"github.com/brainproxy/brainproxy/internal/ws"
	"github.com/brainproxy/brainproxy/web"
)

// version is set at build time via ldflags.
var version = "dev"

func main() {
	if len(os.Args) < 2 {
		// No subcommand: start proxy server (default behavior)
		startServer()
		return
	}

	switch os.Args[1] {
	case "claude":
		runClaude(os.Args[2:])
	case "setup":
		runSetup(os.Args[2:])
	case "version":
		fmt.Printf("brainproxy %s\n", version)
	case "start":
		startServer()
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`brainproxy - LLM proxy with real-time visualization

Usage:
  brainproxy                    Start proxy server (default)
  brainproxy claude [args...]   Launch Claude Code with proxy auto-configured
  brainproxy setup [flags]      Generate config.yaml
  brainproxy version            Print version
  brainproxy help               Show this help

Setup flags:
  --api-key   API key for upstream LLM provider (required)
  --base-url  Upstream API base URL (required)
  --port      Proxy port (default: 8080)

Examples:
  brainproxy setup --api-key sk-xxx --base-url https://dashscope.aliyuncs.com/apps/anthropic
  brainproxy claude
  brainproxy claude --model qwen3.7-max`)
}

// runSetup handles the "setup" subcommand.
func runSetup(args []string) {
	fs := flag.NewFlagSet("setup", flag.ExitOnError)
	apiKey := fs.String("api-key", "", "API key for upstream LLM provider (required)")
	baseURL := fs.String("base-url", "", "Upstream API base URL (required)")
	port := fs.Int("port", 8080, "Proxy listen port")
	configPath := fs.String("config", "config.yaml", "Config file path")
	logDir := fs.String("log-dir", "logs", "Request log directory")
	fs.Parse(args)

	opts := setup.Options{
		APIKey:  *apiKey,
		BaseURL: *baseURL,
		Port:    *port,
		LogDir:  *logDir,
	}
	if err := setup.Run(*configPath, opts); err != nil {
		fmt.Fprintf(os.Stderr, "setup error: %v\n", err)
		os.Exit(1)
	}
}

// runClaude handles the "claude" subcommand.
func runClaude(args []string) {
	configPath := "config.yaml"
	// Check if --config is in the args
	for i, arg := range args {
		if arg == "--config" && i+1 < len(args) {
			configPath = args[i+1]
			// Remove --config and its value from args passed to claude
			args = append(args[:i], args[i+2:]...)
			break
		}
	}
	if err := runner.RunClaude(configPath, args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// startServer runs the proxy server (original behavior).
func startServer() {
	configPath := "config.yaml"
	// Allow --config flag
	for i, arg := range os.Args {
		if arg == "--config" && i+1 < len(os.Args) {
			configPath = os.Args[i+1]
			break
		}
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if cfg.APIKey == "" {
		log.Fatal("api_key is required (set in config.yaml or run 'brainproxy setup')")
	}

	// Initialize components
	memStore := store.NewMemoryStore(cfg.BufferSize)
	hub := ws.NewHub()
	go hub.Run()

	// Initialize request logger
	reqLogger, err := logger.New(cfg.LogDir, cfg.MaxLogFiles)
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	if cfg.LogDir != "" {
		log.Printf("Request logs will be saved to %s/", cfg.LogDir)
	}

	// Event channel: bridge from proxy to WebSocket hub
	events := make(chan models.WSEvent, 256)
	go func() {
		for event := range events {
			// Save completed requests to disk
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

	// Set up routes
	proxyHandler := proxy.NewHandler(cfg.BaseURL, cfg.APIKey, memStore, events)

	mux := http.NewServeMux()

	// API proxy
	mux.Handle("/v1/", proxyHandler)

	// WebSocket endpoint
	mux.HandleFunc("/ws", hub.HandleWS)

	// REST API for fetching stored events
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

	// Serve embedded frontend
	distFS, err := fs.Sub(web.DistFS, "dist")
	if err != nil {
		log.Fatalf("failed to load embedded frontend: %v", err)
	}
	mux.Handle("/", http.FileServer(http.FS(distFS)))

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("BrainProxy v%s starting on http://localhost%s", version, addr)
	log.Printf("Proxy forwarding to %s", cfg.BaseURL)
	log.Printf("WebSocket available at ws://localhost%s/ws", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
