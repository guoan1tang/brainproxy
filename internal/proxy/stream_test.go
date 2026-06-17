package proxy

import "testing"

func TestParseSSELine(t *testing.T) {
	tests := []struct {
		line      string
		eventType string
		data      string
	}{
		// Anthropic format (with space)
		{"event: content_block_start", "content_block_start", ""},
		{"data: {\"type\":\"content_block_start\"}", "", "{\"type\":\"content_block_start\"}"},
		// DashScope format (no space)
		{"event:message_start", "message_start", ""},
		{"data:{\"type\":\"message_start\"}", "", "{\"type\":\"message_start\"}"},
		// Edge cases
		{"", "", ""},
		{"event: ping", "ping", ""},
		{"event:ping", "ping", ""},
	}
	for _, tt := range tests {
		eventType, data := parseSSELine(tt.line)
		if eventType != tt.eventType {
			t.Errorf("line %q: expected event type %q, got %q", tt.line, tt.eventType, eventType)
		}
		if data != tt.data {
			t.Errorf("line %q: expected data %q, got %q", tt.line, tt.data, data)
		}
	}
}

func TestAccumulateStreamingResponse(t *testing.T) {
	acc := NewStreamAccumulator()
	acc.ProcessEvent("message_start", `{"type":"message_start","message":{"id":"msg_1","model":"claude-sonnet-4-20250514","role":"assistant","type":"message","usage":{"input_tokens":25}}}`)
	acc.ProcessEvent("content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_1","name":"search","input":{}}}`)
	acc.ProcessEvent("content_block_delta", `{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"query\":"}}`)
	acc.ProcessEvent("content_block_delta", `{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"\"hello\"}"}}`)
	acc.ProcessEvent("content_block_stop", `{"type":"content_block_stop","index":0}`)
	acc.ProcessEvent("message_delta", `{"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":30}}`)
	acc.ProcessEvent("message_stop", `{"type":"message_stop"}`)

	resp := acc.BuildResponse()
	if resp.ID != "msg_1" {
		t.Errorf("expected ID msg_1, got %s", resp.ID)
	}
	if resp.Model != "claude-sonnet-4-20250514" {
		t.Errorf("expected model claude-sonnet-4-20250514, got %s", resp.Model)
	}
	if resp.StopReason != "tool_use" {
		t.Errorf("expected stop_reason tool_use, got %s", resp.StopReason)
	}
	if len(resp.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(resp.Content))
	}
	if resp.Content[0].Type != "tool_use" {
		t.Errorf("expected content type tool_use, got %s", resp.Content[0].Type)
	}
	if resp.Content[0].Name != "search" {
		t.Errorf("expected tool name search, got %s", resp.Content[0].Name)
	}
}
