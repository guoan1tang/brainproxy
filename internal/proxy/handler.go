package proxy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/brainproxy/brainproxy/internal/analyzer"
	"github.com/brainproxy/brainproxy/internal/models"
	"github.com/brainproxy/brainproxy/internal/store"
	"github.com/google/uuid"
)

// Handler proxies Anthropic API requests, intercepting /v1/messages to record
// request/response pairs and push real-time events over WebSocket.
type Handler struct {
	baseURL string
	apiKey  string
	store   store.Store
	events  chan<- models.WSEvent
	client  *http.Client
}

// NewHandler creates a new proxy Handler.
//   - baseURL: upstream Anthropic API base URL (e.g. https://api.anthropic.com)
//   - apiKey: Anthropic API key injected into forwarded requests
//   - s: store for persisting request events
//   - events: channel for pushing WebSocket events to the hub
func NewHandler(baseURL, apiKey string, s store.Store, events chan<- models.WSEvent) *Handler {
	return &Handler{
		baseURL: baseURL,
		apiKey:  apiKey,
		store:   s,
		events:  events,
		client:  &http.Client{},
	}
}

// ServeHTTP dispatches incoming requests. POST /v1/messages is intercepted and
// recorded; all other paths return 404.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("proxy: %s %s (content-type: %s)", r.Method, r.URL.Path, r.Header.Get("Content-Type"))
	if r.URL.Path == "/v1/messages" && r.Method == "POST" {
		h.handleMessages(w, r)
		return
	}
	http.NotFound(w, r)
}

// handleMessages is the core proxy logic: read request, forward upstream,
// analyse the response, store the event, and push WS notifications.
func (h *Handler) handleMessages(w http.ResponseWriter, r *http.Request) {
	// 1. Read the full request body.
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	r.Body.Close()

	// 2. Parse as models.Request.
	var req models.Request
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		http.Error(w, "invalid request JSON", http.StatusBadRequest)
		return
	}
	req.RawJSON = json.RawMessage(bodyBytes)

	// 3. Generate ID, create event, store it.
	eventID := uuid.New().String()
	event := &models.RequestEvent{
		ID:        eventID,
		Timestamp: time.Now(),
		Request:   req,
	}
	h.store.Add(event)

	// 3b. Extract MCP servers and skills from the request (preliminary analysis).
	mcpServers, skillsUsed := analyzer.AnalyzeRequest(&req)
	event.Analysis = &models.Analysis{
		McpServers: mcpServers,
		SkillsUsed: skillsUsed,
	}
	h.store.Update(eventID, func(e *models.RequestEvent) {
		e.Analysis = event.Analysis
	})

	// 4. Push request.new event.
	h.events <- models.WSEvent{Type: "request.new", Data: event}

	// 5. If streaming, delegate to handleStreaming (Task 5).
	if req.Stream {
		h.handleStreaming(w, r, bodyBytes, event)
		return
	}

	// 6. Forward the request to the upstream API.
	start := time.Now()

	upstreamResp, err := h.forwardRequest(r, bodyBytes)
	if err != nil {
		log.Printf("proxy error: %v", err)
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	defer upstreamResp.Body.Close()

	durationMs := time.Since(start).Milliseconds()

	// 7. Read the full response body.
	respBytes, err := io.ReadAll(upstreamResp.Body)
	if err != nil {
		http.Error(w, "failed to read response", http.StatusBadGateway)
		return
	}

	// 8. Parse response, run analyzer, update stored event.
	var resp models.Response
	if err := json.Unmarshal(respBytes, &resp); err == nil {
		resp.RawJSON = json.RawMessage(respBytes)
		event.Response = &resp
		respAnalysis := analyzer.AnalyzeResponse(&resp, durationMs)
		// Merge response analysis into the preliminary request analysis.
		if event.Analysis == nil {
			event.Analysis = &models.Analysis{}
		}
		event.Analysis.ToolCalls = respAnalysis.ToolCalls
		event.Analysis.Model = respAnalysis.Model
		event.Analysis.InputTokens = respAnalysis.InputTokens
		event.Analysis.OutputTokens = respAnalysis.OutputTokens
		event.Analysis.CacheCreationInputTokens = respAnalysis.CacheCreationInputTokens
		event.Analysis.CacheReadInputTokens = respAnalysis.CacheReadInputTokens
		event.Analysis.StopReason = respAnalysis.StopReason
		event.Analysis.DurationMs = respAnalysis.DurationMs
		// McpServers and SkillsUsed are preserved from request analysis.
		h.store.Update(eventID, func(e *models.RequestEvent) {
			e.Response = event.Response
			e.Analysis = event.Analysis
		})
	}

	// 9. Push request.complete event.
	h.events <- models.WSEvent{Type: "request.complete", Data: event}

	// 10. Return the response to the client.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(upstreamResp.StatusCode)
	w.Write(respBytes)
}

// forwardRequest builds and sends the upstream HTTP request.
// It forwards all headers from the original client request, then
// overrides the API key from config (so the proxy can inject the real key).
func (h *Handler) forwardRequest(originalReq *http.Request, body []byte) (*http.Response, error) {
	target, _ := url.Parse(h.baseURL)
	reqURL := target.String() + "/v1/messages"

	req, err := http.NewRequestWithContext(originalReq.Context(), "POST", reqURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	// Forward all original headers from the client
	for key, vals := range originalReq.Header {
		for _, val := range vals {
			req.Header.Set(key, val)
		}
	}

	// Override API key from config (the real upstream key, not the client's)
	req.Header.Set("x-api-key", h.apiKey)

	return h.client.Do(req)
}

// handleStreaming handles streaming (SSE) requests. It forwards the request
// upstream, streams events back to the client in real time, and accumulates
// the full response for analysis and storage.
func (h *Handler) handleStreaming(w http.ResponseWriter, r *http.Request, bodyBytes []byte, event *models.RequestEvent) {
	start := time.Now()

	upstreamResp, err := h.forwardRequest(r, bodyBytes)
	if err != nil {
		log.Printf("proxy streaming error: %v", err)
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	defer upstreamResp.Body.Close()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(upstreamResp.StatusCode)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	log.Printf("streaming: upstream status=%d content-type=%s", upstreamResp.StatusCode, upstreamResp.Header.Get("Content-Type"))

	acc := NewStreamAccumulator()
	scanner := bufio.NewScanner(upstreamResp.Body)
	// Increase buffer to 10MB — Claude Code responses with large system
	// prompts and tool results easily exceed the default 64KB limit.
	scanner.Buffer(make([]byte, 0, 10*1024*1024), 10*1024*1024)
	var lastEventType string
	var lineCount int

	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

		// Debug: log first 10 lines to understand SSE format
		if lineCount <= 10 {
			log.Printf("streaming: line %d: %s", lineCount, line)
		}

		fmt.Fprintf(w, "%s\n", line)
		flusher.Flush()

		if strings.HasPrefix(line, "event:") {
			lastEventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			acc.ProcessEvent(lastEventType, data)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("streaming: scanner error after %d lines: %v", lineCount, err)
	}
	log.Printf("streaming: finished reading %d lines from upstream", lineCount)

	durationMs := time.Since(start).Milliseconds()
	resp := acc.BuildResponse()
	log.Printf("streaming: accumulated response — id=%s model=%s content_blocks=%d tool_calls=%d",
		resp.ID, resp.Model, len(resp.Content), len(resp.Content))
	event.Response = resp
	respAnalysis := analyzer.AnalyzeResponse(resp, durationMs)
	// Merge response analysis into the preliminary request analysis.
	if event.Analysis == nil {
		event.Analysis = &models.Analysis{}
	}
	event.Analysis.ToolCalls = respAnalysis.ToolCalls
	event.Analysis.Model = respAnalysis.Model
	event.Analysis.InputTokens = respAnalysis.InputTokens
	event.Analysis.OutputTokens = respAnalysis.OutputTokens
	event.Analysis.CacheCreationInputTokens = respAnalysis.CacheCreationInputTokens
	event.Analysis.CacheReadInputTokens = respAnalysis.CacheReadInputTokens
	event.Analysis.StopReason = respAnalysis.StopReason
	event.Analysis.DurationMs = respAnalysis.DurationMs
	// McpServers and SkillsUsed are preserved from request analysis.
	h.store.Update(event.ID, func(e *models.RequestEvent) {
		e.Response = event.Response
		e.Analysis = event.Analysis
	})
	h.events <- models.WSEvent{Type: "request.complete", Data: event}
}
