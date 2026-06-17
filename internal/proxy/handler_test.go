package proxy

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/brainproxy/brainproxy/internal/models"
	"github.com/brainproxy/brainproxy/internal/store"
)

func TestProxyHandler_ForwardsAndRecords(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Errorf("expected path /v1/messages, got %s", r.URL.Path)
		}
		if r.Header.Get("x-api-key") == "" {
			t.Error("expected x-api-key header to be forwarded")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"id": "msg_test",
			"type": "message",
			"role": "assistant",
			"model": "claude-sonnet-4-20250514",
			"content": [{"type": "text", "text": "Hi"}],
			"stop_reason": "end_turn",
			"usage": {"input_tokens": 10, "output_tokens": 5}
		}`))
	}))
	defer upstream.Close()

	memStore := store.NewMemoryStore(100)
	events := make(chan models.WSEvent, 10)
	handler := NewHandler(upstream.URL, "test-key", memStore, events)

	reqBody := `{"model":"claude-sonnet-4-20250514","messages":[{"role":"user","content":"Hello"}],"max_tokens":100}`
	req := httptest.NewRequest("POST", "/v1/messages", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp models.Response
	body, _ := io.ReadAll(rec.Body)
	json.Unmarshal(body, &resp)
	if resp.ID != "msg_test" {
		t.Errorf("expected response ID msg_test, got %s", resp.ID)
	}

	storedEvents := memStore.List()
	if len(storedEvents) != 1 {
		t.Fatalf("expected 1 stored event, got %d", len(storedEvents))
	}
	if storedEvents[0].Analysis == nil {
		t.Fatal("expected analysis to be populated")
	}
	if storedEvents[0].Analysis.Model != "claude-sonnet-4-20250514" {
		t.Errorf("expected model claude-sonnet-4-20250514, got %s", storedEvents[0].Analysis.Model)
	}

	if len(events) < 2 {
		t.Fatalf("expected at least 2 WS events, got %d", len(events))
	}
	e1 := <-events
	if e1.Type != "request.new" {
		t.Errorf("expected first event type 'request.new', got '%s'", e1.Type)
	}
	e2 := <-events
	if e2.Type != "request.complete" {
		t.Errorf("expected second event type 'request.complete', got '%s'", e2.Type)
	}
}
