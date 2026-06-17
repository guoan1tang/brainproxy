package proxy

import (
	"encoding/json"
	"strings"

	"github.com/brainproxy/brainproxy/internal/models"
)

// parseSSELine parses a single SSE line and returns (eventType, data).
// Handles both "event: foo" (Anthropic) and "event:foo" (DashScope) formats.
//   - "event: foo" or "event:foo" → ("foo", "")
//   - "data: bar" or "data:bar"   → ("", "bar")
//   - anything else → ("", "")
func parseSSELine(line string) (eventType, data string) {
	if strings.HasPrefix(line, "event:") {
		val := strings.TrimPrefix(line, "event:")
		val = strings.TrimPrefix(val, " ") // optional space
		return val, ""
	}
	if strings.HasPrefix(line, "data:") {
		val := strings.TrimPrefix(line, "data:")
		val = strings.TrimPrefix(val, " ") // optional space
		return "", val
	}
	return "", ""
}

// streamContentBlock tracks an in-progress content block during SSE accumulation.
type streamContentBlock struct {
	Type        string // "text" or "tool_use"
	ID          string // tool_use block ID
	Name        string // tool_use block name
	TextParts   []string
	JSONParts   []string
}

// StreamAccumulator collects SSE streaming events and builds a complete Response.
type StreamAccumulator struct {
	id          string
	model       string
	role        string
	respType    string
	stopReason  string
	inputTok    int
	outputTok   int
	cacheCreate int
	cacheRead   int
	blocks      []*streamContentBlock
}

// NewStreamAccumulator creates a new StreamAccumulator.
func NewStreamAccumulator() *StreamAccumulator {
	return &StreamAccumulator{}
}

// messageStartPayload mirrors the message_start event JSON structure.
type messageStartPayload struct {
	Message struct {
		ID    string `json:"id"`
		Model string `json:"model"`
		Role  string `json:"role"`
		Type  string `json:"type"`
		Usage struct {
			InputTokens              int `json:"input_tokens"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
			CacheReadInputTokens     int `json:"cache_read_input_tokens"`
		} `json:"usage"`
	} `json:"message"`
}

// contentBlockStartPayload mirrors the content_block_start event JSON structure.
type contentBlockStartPayload struct {
	Index        int `json:"index"`
	ContentBlock struct {
		Type   string `json:"type"`
		ID     string `json:"id"`
		Name   string `json:"name"`
	} `json:"content_block"`
}

// contentBlockDeltaPayload mirrors the content_block_delta event JSON structure.
type contentBlockDeltaPayload struct {
	Index int `json:"index"`
	Delta struct {
		Type        string `json:"type"`
		Text        string `json:"text"`
		PartialJSON string `json:"partial_json"`
	} `json:"delta"`
}

// messageDeltaPayload mirrors the message_delta event JSON structure.
type messageDeltaPayload struct {
	Delta struct {
		StopReason string `json:"stop_reason"`
	} `json:"delta"`
	Usage struct {
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// ProcessEvent handles a single SSE event, accumulating data into the response.
func (a *StreamAccumulator) ProcessEvent(eventType, data string) {
	switch eventType {
	case "message_start":
		var p messageStartPayload
		if err := json.Unmarshal([]byte(data), &p); err != nil {
			return
		}
		a.id = p.Message.ID
		a.model = p.Message.Model
		a.role = p.Message.Role
		a.respType = p.Message.Type
		a.inputTok = p.Message.Usage.InputTokens
		a.cacheCreate = p.Message.Usage.CacheCreationInputTokens
		a.cacheRead = p.Message.Usage.CacheReadInputTokens

	case "content_block_start":
		var p contentBlockStartPayload
		if err := json.Unmarshal([]byte(data), &p); err != nil {
			return
		}
		block := &streamContentBlock{
			Type: p.ContentBlock.Type,
			ID:   p.ContentBlock.ID,
			Name: p.ContentBlock.Name,
		}
		// Grow the blocks slice to accommodate the index.
		for len(a.blocks) <= p.Index {
			a.blocks = append(a.blocks, nil)
		}
		a.blocks[p.Index] = block

	case "content_block_delta":
		var p contentBlockDeltaPayload
		if err := json.Unmarshal([]byte(data), &p); err != nil {
			return
		}
		if p.Index < 0 || p.Index >= len(a.blocks) || a.blocks[p.Index] == nil {
			return
		}
		block := a.blocks[p.Index]
		switch p.Delta.Type {
		case "text_delta":
			block.TextParts = append(block.TextParts, p.Delta.Text)
		case "input_json_delta":
			block.JSONParts = append(block.JSONParts, p.Delta.PartialJSON)
		}

	case "message_delta":
		var p messageDeltaPayload
		if err := json.Unmarshal([]byte(data), &p); err != nil {
			return
		}
		a.stopReason = p.Delta.StopReason
		a.outputTok = p.Usage.OutputTokens
	}
}

// BuildResponse finalises accumulated data and returns a complete *models.Response.
func (a *StreamAccumulator) BuildResponse() *models.Response {
	content := make([]models.ContentBlock, 0, len(a.blocks))
	for _, block := range a.blocks {
		if block == nil {
			continue
		}
		cb := models.ContentBlock{
			Type: block.Type,
		}
		switch block.Type {
		case "text":
			cb.Text = strings.Join(block.TextParts, "")
		case "tool_use":
			cb.ID = block.ID
			cb.Name = block.Name
			mergedJSON := strings.Join(block.JSONParts, "")
			if mergedJSON != "" {
				cb.Input = json.RawMessage(mergedJSON)
			} else {
				cb.Input = json.RawMessage("{}")
			}
		}
		content = append(content, cb)
	}

	resp := &models.Response{
		ID:         a.id,
		Type:       a.respType,
		Role:       a.role,
		Model:      a.model,
		StopReason: a.stopReason,
		Content:    content,
		Usage: models.Usage{
			InputTokens:              a.inputTok,
			OutputTokens:             a.outputTok,
			CacheCreationInputTokens: a.cacheCreate,
			CacheReadInputTokens:     a.cacheRead,
		},
	}

	// Set RawJSON from the built response.
	if raw, err := json.Marshal(resp); err == nil {
		resp.RawJSON = json.RawMessage(raw)
	}

	return resp
}
