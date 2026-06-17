package models

import (
	"encoding/json"
	"time"
)

// RequestEvent represents a single intercepted API request/response pair.
type RequestEvent struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Request   Request   `json:"request"`
	Response  *Response `json:"response,omitempty"`
	Analysis  *Analysis `json:"analysis,omitempty"`
}

// Request captures the incoming API request body.
type Request struct {
	Model     string          `json:"model"`
	System    any             `json:"system,omitempty"`
	Messages  []Message       `json:"messages"`
	Tools     []ToolDef       `json:"tools,omitempty"`
	MaxTokens int             `json:"max_tokens,omitempty"`
	Stream    bool            `json:"stream,omitempty"`
	RawJSON   json.RawMessage `json:"raw_json"`
}

// Message is a single message in the conversation.
type Message struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

// ContentBlock is a typed content block (text, tool_use, tool_result, etc).
type ContentBlock struct {
	Type    string          `json:"type"`
	Text    string          `json:"text,omitempty"`
	ID      string          `json:"id,omitempty"`
	Name    string          `json:"name,omitempty"`
	Input   json.RawMessage `json:"input,omitempty"`
	Content json.RawMessage `json:"content,omitempty"`
}

// ToolDef is a tool definition from the request.
type ToolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema,omitempty"`
}

// Response captures the API response.
type Response struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Role       string          `json:"role"`
	Content    []ContentBlock  `json:"content"`
	Model      string          `json:"model"`
	StopReason string          `json:"stop_reason"`
	Usage      Usage           `json:"usage"`
	RawJSON    json.RawMessage `json:"raw_json"`
}

// Usage holds token usage info.
type Usage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

// Analysis holds the proxy's analysis of a request/response pair.
type Analysis struct {
	ToolCalls                []ToolCallInfo `json:"tool_calls"`
	Model                    string         `json:"model"`
	InputTokens              int            `json:"input_tokens"`
	OutputTokens             int            `json:"output_tokens"`
	CacheCreationInputTokens int            `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int            `json:"cache_read_input_tokens,omitempty"`
	StopReason               string         `json:"stop_reason"`
	DurationMs               int64          `json:"duration_ms"`
	McpServers               []string       `json:"mcp_servers,omitempty"`
	SkillsUsed               []string       `json:"skills_used,omitempty"`
}

// ToolCallInfo describes a single tool call detected in the response.
type ToolCallInfo struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// WSEvent is a WebSocket event pushed to the frontend.
type WSEvent struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}
