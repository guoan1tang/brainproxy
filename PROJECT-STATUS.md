# BrainProxy 项目状态总结

## 项目概述

BrainProxy 是一个本地 Go 代理，拦截 Claude Code 发往 LLM API 的请求，实时可视化展示在 Web UI 上。

## 架构

```
Claude Code → localhost:8080 → DashScope (qwen3.7-max)
                 ↓
          React Web UI (内嵌在 Go binary 中)
          - 大脑中心 + 功能节点环绕
          - 请求列表 + 详情面板
          - WebSocket 实时推送
```

## 项目位置

- **代码**: `/Users/duliday/Documents/workspace/skill-testing/brainproxy/`
- **设计文档**: `docs/superpowers/specs/2026-06-15-brainproxy-design.md`
- **实现计划**: `docs/superpowers/plans/2026-06-15-brainproxy.md`
- **Git**: 已初始化，所有改动已提交

## 技术栈

- **后端**: Go 1.22+, gorilla/websocket, 内存 Ring Buffer 存储
- **前端**: React 18 + TypeScript + Vite + framer-motion
- **部署**: 单 binary（Go embed 内嵌前端）

## 已实现功能

### 后端 (Go)
- [x] 反向代理拦截 `/v1/messages`（支持 streaming SSE + 非 streaming）
- [x] DashScope SSE 格式兼容（冒号后无空格 `event:xxx` vs Anthropic 的 `event: xxx`）
- [x] 流式响应累积（StreamAccumulator）
- [x] Tool Call 检测（从 response content 中提取 tool_use 块）
- [x] MCP Server 检测（从 tool 定义中提取 `mcp__SERVER__action`）
- [x] Skill 检测（从最后一条 user 消息中提取 `Base directory for this skill:` 标记）
- [x] Cache token 提取（cache_creation_input_tokens, cache_read_input_tokens）
- [x] WebSocket Hub 实时推送事件
- [x] 所有请求路径日志（debug 用）

### 前端 (React)
- [x] 大脑中心可视化（脉冲动画）
- [x] 功能节点环绕（Tool/MCP/Skill 三类，不同颜色和图标）
- [x] 激活节点发光 + 连线粒子动画
- [x] 请求列表（显示 tokens、cache 命中百分比、tool 数量）
- [x] 请求详情面板（Input/Output 双栏）
- [x] Available Skills 列表（合并所有来源，有描述的排前面）
- [x] Skills Loaded 列表（检测已加载的技能）
- [x] System Reminders 提取（从 messages 中提取 `<system-reminder>` 内容）
- [x] Tool Sidebar（所有工具分组列表，激活的高亮）
- [x] 节点点击展示 tool call 详情（输入参数 JSON）
- [x] 同名工具合并显示（如 TaskCreate ×5）
- [x] Cache token 显示（绿色 cache read，橙色 cache create）
- [x] 点击请求时大脑只显示该请求的数据

## 关键配置

### config.yaml
```yaml
api_key: "sk-your-api-key"
base_url: "https://dashscope.aliyuncs.com/apps/anthropic"
port: 8080
buffer_size: 500
```

### Claude Code settings (~/.claude/settings.json)
```json
{
  "ANTHROPIC_BASE_URL": "http://localhost:8080"  // 指向 BrainProxy
}
```
要停用 BrainProxy，改回：`"ANTHROPIC_BASE_URL": "https://dashscope.aliyuncs.com/apps/anthropic"`

## 启动方式

```bash
cd /Users/duliday/Documents/workspace/skill-testing/brainproxy
./brainproxy
# 浏览器打开 http://localhost:8080
```

## Claude Code 技能系统发现

### 当前版本的行为（已验证）
1. **技能名称列表**注入在 `messages[1]`（role=system）中，格式：`The following skills are available for use with the Skill tool:`
2. **大部分技能只有名字**，没有描述（如 `- brainstorming`）
3. **少数技能有描述**，出现在后续的 system 消息中（如 msg[139] 的 `duliday-prd-producer`）
4. **技能完整内容**只在被调用后作为 user message 加载（`Base directory for this skill:` 标记）
5. **没有找到 `<available_skills>` XML 块**——可能新版本才有

### 官方声称的行为（待验证）
- 技能信息应以 `<available_skills>` XML 格式注入 system prompt
- 包含 name + description
- 升级 Claude Code 后可能出现

### 两种技能路由设计对比
| 维度 | Claude Code 当前模式 | claude.ai 模式 |
|------|---------------------|---------------|
| 路由 | 只有名字 | 名字 + 描述 |
| 加载 | 懒加载（Skill tool） | 预加载系统提示 |
| 激活 | "1%可能就调用" | 按触发条件匹配 |
| Token | 省上下文 | 占系统提示但更精准 |

## 待实现（Phase 2）

- [ ] Skill 文件管理系统（加载 `~/.claude/skills/*.md`）
- [ ] 更精准的 Skill 匹配（基于描述语义匹配）
- [ ] SQLite 持久化存储
- [ ] 技能使用统计和分析

## Git 提交历史（最近）

```
4233551 feat: merge skill descriptions from all sources into Available Skills list
74f4fda debug: log all incoming request paths
df14485 feat: extract and display system-reminder blocks from messages
dd4337c feat: extract and display skills loaded from messages in Input panel
864b849 fix: merge duplicate tools in Activated bar, only show called MCP servers
dd66493 fix: only show MCP node in graph when its tools are actually called
74e9109 feat: display cache hit tokens (read/create) in request detail and list
cf1d7f0 feat: merge duplicate tool nodes with count, click node to show tool call details
4b41b0d feat: show all request tools as nodes, light up called ones; compact mode
a635c20 fix: only detect skills from last user message to avoid history false positives
b7e9620 feat: activated tools in graph, all tools in sidebar list
51f307d feat: detect MCP servers and Skills, display as categorized nodes
69f2e2c fix: support DashScope SSE format (no space after colon)
4623013 fix: null safety for old events with missing analysis fields
```
