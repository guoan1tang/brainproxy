# BrainProxy 🔍

Claude Code 的 LLM 请求代理 — 实时可视化展示请求、工具调用和 Token 用量，浏览器打开即用。零配置，`brainproxy claude` 一键启动。

## 功能

- 🧠 **实时可视化** — 大脑中心图 + 功能节点环绕动画
- 🔧 **Tool Call 追踪** — 检测并高亮所有工具调用
- 🔌 **MCP Server 检测** — 自动识别 MCP 服务器及其工具
- 📦 **Skill 识别** — 检测已加载和已使用的技能
- 💰 **Token 统计** — 输入/输出 Token、Cache 命中率
- 📝 **请求日志** — 完整请求/响应自动保存为 JSON，支持 jq 分析
- 🚀 **零配置** — 不改 `~/.claude/settings.json`，一条命令启动

## 快速开始

### 1. 下载

从 [GitHub Releases](../../releases) 下载对应平台的二进制文件：

| 平台 | 文件 |
|------|------|
| macOS Apple Silicon | `brainproxy-darwin-arm64.tar.gz` |
| macOS Intel | `brainproxy-darwin-amd64.tar.gz` |
| Linux x86_64 | `brainproxy-linux-amd64.tar.gz` |
| Linux ARM64 | `brainproxy-linux-arm64.tar.gz` |
| Windows | `brainproxy-windows-amd64.zip` |

解压后放入 PATH 目录即可使用。

### 2. 初始化配置

```bash
brainproxy setup \
  --api-key sk-your-api-key \
  --base-url https://dashscope.aliyuncs.com/apps/anthropic
```

这会生成 `config.yaml` 并创建 `logs/` 目录。

### 3. 启动

```bash
# 一键启动代理 + Claude Code（推荐）
brainproxy claude

# 或者只启动代理，手动配置 Claude Code
brainproxy
```

启动后打开 http://localhost:8080 查看 Web UI。

## 命令参考

| 命令 | 说明 |
|------|------|
| `brainproxy` | 启动代理服务器（手动模式） |
| `brainproxy claude [args...]` | 自动启动代理 + Claude Code，透传所有参数 |
| `brainproxy setup [flags]` | 生成 config.yaml |
| `brainproxy version` | 打印版本信息 |
| `brainproxy help` | 显示帮助 |

### setup 参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `--api-key` | 上游 LLM API Key（必填） | — |
| `--base-url` | 上游 API 地址（必填） | — |
| `--port` | 代理监听端口 | `8080` |
| `--config` | 配置文件路径 | `config.yaml` |
| `--log-dir` | 日志保存目录 | `logs` |

## 配置

### config.yaml

```yaml
api_key: "sk-your-api-key"           # 上游 API Key
base_url: "https://your-api.com"     # 上游 API 地址
port: 8080                           # 代理端口
buffer_size: 500                     # 内存缓冲区大小
log_dir: "logs"                      # 请求日志目录
```

### 环境变量

所有配置项均可通过环境变量覆盖：

```bash
BRAINPROXY_API_KEY=sk-xxx
BRAINPROXY_BASE_URL=https://your-api.com
BRAINPROXY_PORT=8080
BRAINPROXY_LOG_DIR=logs
```

## 请求日志

每个完成的请求自动保存为 JSON 文件到 `logs/` 目录：

```
logs/
├── 20260617-143052_a1b2c3d4.json
├── 20260617-143058_e5f6g7h8.json
└── ...
```

用 `jq` 分析：

```bash
# 查看所有工具调用
jq '.analysis.tool_calls[] | .name' logs/*.json

# Token 用量统计
jq '.analysis | {model, input_tokens, output_tokens}' logs/*.json

# 查看完整请求
jq '.request' logs/20260617-143052_a1b2c3d4.json

# 查看完整响应
jq '.response' logs/20260617-143052_a1b2c3d4.json
```

## 架构

```
Claude Code                    BrainProxy (:8080)                    LLM API
    │                                │                                  │
    │  POST /v1/messages             │                                  │
    │ ─────────────────────────────► │                                  │
    │                                │  记录 + 分析                      │
    │                                │ ────────────────────────────────►│
    │                                │                                  │
    │                                │  响应（流式/非流式）               │
    │                                │◄────────────────────────────────│
    │  原样返回                       │                                  │
    │◄──────────────────────────────│  记录 + 分析 + 保存日志            │
    │                                │                                  │
    │                                │──► Web UI (实时推送)              │
    │                                │──► logs/*.json (持久化)           │
```

## 从源码构建

需要 Go 1.22+ 和 Node.js 20+。

```bash
git clone https://github.com/guoan1tang/brainproxy.git
cd brainproxy
make build        # 编译（含前端构建）
make build-all    # 交叉编译所有平台
make test         # 运行测试
```

## 停用

- `brainproxy claude` 模式：退出 Claude Code 后代理自动关闭，无需任何操作
- 手动模式：`Ctrl+C` 停止代理
- 如需恢复直连：将 `~/.claude/settings.json` 中的 `ANTHROPIC_BASE_URL` 改回原始地址

## License

MIT
