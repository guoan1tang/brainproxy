package analyzer

import (
	"regexp"
	"sort"
	"strings"

	"github.com/brainproxy/brainproxy/internal/models"
)

// AnalyzeResponse extracts tool calls and metadata from an API response.
func AnalyzeResponse(resp *models.Response, durationMs int64) *models.Analysis {
	analysis := &models.Analysis{
		Model:                    resp.Model,
		InputTokens:              resp.Usage.InputTokens,
		OutputTokens:             resp.Usage.OutputTokens,
		CacheCreationInputTokens: resp.Usage.CacheCreationInputTokens,
		CacheReadInputTokens:     resp.Usage.CacheReadInputTokens,
		StopReason:               resp.StopReason,
		DurationMs:               durationMs,
	}

	for _, block := range resp.Content {
		if block.Type == "tool_use" {
			analysis.ToolCalls = append(analysis.ToolCalls, models.ToolCallInfo{
				ID:    block.ID,
				Name:  block.Name,
				Input: block.Input,
			})
		}
	}

	if analysis.ToolCalls == nil {
		analysis.ToolCalls = []models.ToolCallInfo{}
	}

	return analysis
}

// AnalyzeRequest extracts MCP server names and skill names from the request.
// MCP servers are derived from tool names matching mcp__SERVER__action.
// Skills are detected by the marker "Base directory for this skill:" in user messages.
func AnalyzeRequest(req *models.Request) (mcpServers []string, skillsUsed []string) {
	// Extract unique MCP server names from tool definitions.
	serverSet := make(map[string]bool)
	for _, tool := range req.Tools {
		if strings.HasPrefix(tool.Name, "mcp__") {
			parts := strings.SplitN(tool.Name, "__", 3)
			if len(parts) >= 2 {
				serverSet[parts[1]] = true
			}
		}
	}
	for name := range serverSet {
		mcpServers = append(mcpServers, name)
	}
	sort.Strings(mcpServers)

	// Extract skill names from the LAST user message only.
	// Skills are loaded by Claude Code into the most recent user message
	// when invoked. Scanning all historical messages would cause false positives
	// since earlier skill invocations remain in conversation history.
	skillSet := make(map[string]bool)
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			scanForSkills(req.Messages[i].Content, skillSet)
			break
		}
	}

	for name := range skillSet {
		skillsUsed = append(skillsUsed, name)
	}
	sort.Strings(skillsUsed)
	return
}

// scanForSkills inspects content (typed as any after JSON unmarshal) for skill markers.
func scanForSkills(content any, skillSet map[string]bool) {
	switch c := content.(type) {
	case string:
		extractSkillsFromText(c, skillSet)
	case []any:
		for _, item := range c {
			if block, ok := item.(map[string]any); ok {
				if text, ok := block["text"].(string); ok {
					extractSkillsFromText(text, skillSet)
				}
			}
		}
	}
}

// skillPathRe matches a valid skill directory path:
//
//	/.claude/skills/SKILL_NAME
//
// where SKILL_NAME contains only word chars and hyphens.
var skillPathRe = regexp.MustCompile(`\.claude/skills/([\w][\w-]*)`)

// extractSkillsFromText scans text for the skill marker pattern and
// adds any discovered skill names to skillSet.
// Only matches paths containing ".claude/skills/" to avoid false positives
// from assistant messages that discuss the skill system.
func extractSkillsFromText(text string, skillSet map[string]bool) {
	const marker = "Base directory for this skill:"
	remaining := text
	for {
		idx := strings.Index(remaining, marker)
		if idx == -1 {
			return
		}
		rest := remaining[idx+len(marker):]
		// Only look at the current line
		if nlIdx := strings.Index(rest, "\n"); nlIdx != -1 {
			rest = rest[:nlIdx]
		}
		rest = strings.TrimSpace(rest)

		// Validate: path must contain ".claude/skills/"
		if matches := skillPathRe.FindStringSubmatch(rest); len(matches) > 1 {
			skillSet[matches[1]] = true
		}

		// Advance past this occurrence to find more skills.
		remaining = remaining[idx+len(marker):]
	}
}
