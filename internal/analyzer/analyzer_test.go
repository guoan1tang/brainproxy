package analyzer

import (
	"sort"
	"testing"

	"github.com/brainproxy/brainproxy/internal/models"
)

func TestAnalyzeRequest_MCPServers(t *testing.T) {
	req := &models.Request{
		Tools: []models.ToolDef{
			{Name: "Bash"},
			{Name: "mcp__yunxiao__create_branch"},
			{Name: "mcp__yunxiao__list_pipelines"},
			{Name: "mcp__feishu__send_message"},
		},
	}
	servers, _ := AnalyzeRequest(req)
	if len(servers) != 2 {
		t.Errorf("expected 2 MCP servers, got %d: %v", len(servers), servers)
	}
	sort.Strings(servers)
	if servers[0] != "feishu" || servers[1] != "yunxiao" {
		t.Errorf("expected [feishu, yunxiao], got %v", servers)
	}
}

func TestAnalyzeRequest_NoMCPServers(t *testing.T) {
	req := &models.Request{
		Tools: []models.ToolDef{
			{Name: "Bash"},
			{Name: "Read"},
			{Name: "Write"},
		},
	}
	servers, _ := AnalyzeRequest(req)
	if len(servers) != 0 {
		t.Errorf("expected 0 MCP servers, got %d: %v", len(servers), servers)
	}
}

func TestAnalyzeRequest_Skills(t *testing.T) {
	req := &models.Request{
		Messages: []models.Message{
			{Role: "user", Content: "Base directory for this skill: /Users/test/.claude/skills/brainstorming\n\n# Brainstorming..."},
		},
	}
	_, skills := AnalyzeRequest(req)
	if len(skills) != 1 || skills[0] != "brainstorming" {
		t.Errorf("expected [brainstorming], got %v", skills)
	}
}

func TestAnalyzeRequest_MultipleSkills(t *testing.T) {
	req := &models.Request{
		Messages: []models.Message{
			{Role: "user", Content: "Base directory for this skill: /Users/test/.claude/skills/brainstorming\n\nSome text"},
			{Role: "assistant", Content: "Base directory for this skill: /Users/test/.claude/skills/code-review\n\nMore text"},
			{Role: "user", Content: "Base directory for this skill: /Users/test/.claude/skills/writing-plans\n\nPlans text"},
		},
	}
	_, skills := AnalyzeRequest(req)
	// Only the last user message is scanned
	if len(skills) != 1 || skills[0] != "writing-plans" {
		t.Errorf("expected [writing-plans] (last user msg only), got %v", skills)
	}
}

func TestAnalyzeRequest_SkillsInContentBlocks(t *testing.T) {
	// When JSON is unmarshaled into any, arrays become []any and objects become map[string]any.
	req := &models.Request{
		Messages: []models.Message{
			{
				Role: "user",
				Content: []any{
					map[string]any{
						"type": "text",
						"text": "Base directory for this skill: /Users/test/.claude/skills/pdf\n\nPDF skill content",
					},
				},
			},
		},
	}
	_, skills := AnalyzeRequest(req)
	if len(skills) != 1 || skills[0] != "pdf" {
		t.Errorf("expected [pdf], got %v", skills)
	}
}

func TestAnalyzeRequest_SkillsInSystemPromptArray(t *testing.T) {
	// System prompt is NOT scanned for skills (only user messages).
	req := &models.Request{
		System: []any{
			map[string]any{
				"type": "text",
				"text": "Base directory for this skill: /Users/test/.claude/skills/init",
			},
		},
	}
	_, skills := AnalyzeRequest(req)
	if len(skills) != 0 {
		t.Errorf("expected 0 skills (system prompt not scanned), got %v", skills)
	}
}

func TestAnalyzeRequest_DuplicateSkills(t *testing.T) {
	req := &models.Request{
		Messages: []models.Message{
			{Role: "user", Content: "Base directory for this skill: /Users/test/.claude/skills/brainstorming\n\nFirst mention"},
			{Role: "assistant", Content: "Base directory for this skill: /Users/test/.claude/skills/brainstorming\n\nSecond mention"},
		},
	}
	_, skills := AnalyzeRequest(req)
	if len(skills) != 1 {
		t.Errorf("expected 1 skill (deduped), got %d: %v", len(skills), skills)
	}
}

func TestAnalyzeRequest_AssistantMessagesIgnored(t *testing.T) {
	// Assistant messages discussing skills should NOT be detected.
	req := &models.Request{
		Messages: []models.Message{
			{Role: "user", Content: "Base directory for this skill: /Users/test/.claude/skills/brainstorming\n\nSkill content"},
			{Role: "assistant", Content: "The marker 'Base directory for this skill: /path/skills/NAME' appears in messages"},
			{Role: "assistant", Content: "Base directory for this skill: /some/other/path\n\nNot a real skill"},
		},
	}
	_, skills := AnalyzeRequest(req)
	if len(skills) != 1 || skills[0] != "brainstorming" {
		t.Errorf("expected only [brainstorming], got %v", skills)
	}
}

func TestAnalyzeRequest_EmptyRequest(t *testing.T) {
	req := &models.Request{}
	servers, skills := AnalyzeRequest(req)
	if len(servers) != 0 {
		t.Errorf("expected 0 servers, got %d", len(servers))
	}
	if len(skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(skills))
	}
}

func TestAnalyzeRequest_MCPToolWithTwoUnderscores(t *testing.T) {
	// Tool name has exactly the mcp__SERVER__action pattern.
	req := &models.Request{
		Tools: []models.ToolDef{
			{Name: "mcp__github__create_pr"},
		},
	}
	servers, _ := AnalyzeRequest(req)
	if len(servers) != 1 || servers[0] != "github" {
		t.Errorf("expected [github], got %v", servers)
	}
}

func TestAnalyzeResponse_ToolCalls(t *testing.T) {
	resp := &models.Response{
		Model:      "claude-sonnet-4-20250514",
		StopReason: "tool_use",
		Usage: models.Usage{
			InputTokens:  100,
			OutputTokens: 50,
		},
		Content: []models.ContentBlock{
			{Type: "text", Text: "Let me help with that."},
			{Type: "tool_use", ID: "toolu_123", Name: "Bash", Input: []byte(`{"command":"ls"}`)},
		},
	}
	analysis := AnalyzeResponse(resp, 2000)
	if analysis.Model != "claude-sonnet-4-20250514" {
		t.Errorf("expected model claude-sonnet-4-20250514, got %s", analysis.Model)
	}
	if len(analysis.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(analysis.ToolCalls))
	}
	if analysis.ToolCalls[0].Name != "Bash" {
		t.Errorf("expected tool name Bash, got %s", analysis.ToolCalls[0].Name)
	}
	if analysis.DurationMs != 2000 {
		t.Errorf("expected duration 2000ms, got %d", analysis.DurationMs)
	}
}

func TestAnalyzeResponse_NoToolCalls(t *testing.T) {
	resp := &models.Response{
		Model:      "claude-sonnet-4-20250514",
		StopReason: "end_turn",
		Usage:      models.Usage{InputTokens: 10, OutputTokens: 5},
		Content:    []models.ContentBlock{{Type: "text", Text: "Hello!"}},
	}
	analysis := AnalyzeResponse(resp, 500)
	if len(analysis.ToolCalls) != 0 {
		t.Errorf("expected 0 tool calls, got %d", len(analysis.ToolCalls))
	}
}
