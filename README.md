# kui

Go agent runtime with profile system, hot-switching TUI, and multi-provider support. Pi-inspired, built with Bubble Tea.

## Quick start

```bash
# Set your API key
export OPENAI_API_KEY="sk-..."

# Run a prompt (one-shot)
kui "explain this codebase"

# Start the interactive TUI
kui tui
```

## Features

| Feature | Description |
|---------|-------------|
| **Profile system** | Per-profile model, provider, tools, skills, MCP servers, system prompt |
| **Hot switching** | TAB-driven profile switching mid-session |
| **Multi-provider** | OpenAI, OpenCode Zen (free models available) |
| **Thinking levels** | `--thinking off\|low\|medium\|high` for reasoning models |
| **Dynamic extensions** | Runtime-loaded subprocess extensions via JSON-RPC 2.0 |
| **MCP support** | Model Context Protocol servers with tool bridging |
| **Skills** | Local + remote skill index with frontmatter metadata |

## Usage

```
kui [--] PROMPT...           Run agent loop, print answer to stdout
kui tui                      Start interactive TUI
kui profile list             List profiles (active marked)
kui profile switch <name>    Activate profile for session
kui profile model <name> <model>   Set per-profile model
kui profile thinking <name> <level>  Set per-profile thinking level
```

## Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--model` | `-m` | Override resolved model |
| `--provider` | `-p` | Select provider (`openai`, `opencode`) |
| `--thinking` | | Reasoning effort: `off`, `low`, `medium`, `high` |
| `--tools` | | Comma-separated tool names to include |
| `--exclude-tools` | | Comma-separated tool names to exclude |
| `--no-tools` | | Disable all tools |
| `--no-extensions` | `-ne` | Skip extension loading |
| `--no-skills` | `-ns` | Skip skill index building |
| `--verbose` | | Debug output to stderr |
| `--mode` | | Output format: `text` (default) or `json` |
| `--approve` | `-a` | Bypass permission checks |

## Environment

| Variable | Required | Description |
|----------|----------|-------------|
| `OPENAI_API_KEY` | Yes | API key for chat-completions endpoint |
| `OPENAI_BASE_URL` | No | Defaults to `https://api.openai.com/v1` |
| `OPENAI_MODEL` | No | Defaults to `gpt-4o-mini` |
| `KUI_PROVIDER` | No | Default provider (`openai`, `opencode`) |
| `OPENCODE_API_KEY` | No | API key for OpenCode Zen |
| `KUI_HOME` | No | Config directory override |

## Providers

### OpenAI (default)

Uses the OpenAI chat-completions API. Set `OPENAI_API_KEY` and optionally `OPENAI_BASE_URL` for compatible endpoints.

### OpenCode Zen

Uses OpenCode's OpenAI-compatible endpoint with free/cheap models:

```bash
export OPENCODE_API_KEY="your-key"
kui --provider opencode "hello"
```

Available models: `big-pickle` (free, 200K context), `mimo-v2.5` ($0.14/1M tokens, 1M context).

## Configuration

kui reads configuration from:

1. **Global**: `~/.config/kui/` (or `$KUI_HOME/kui/`)
2. **Project**: `.kui/` in current directory
3. **Profile**: per-profile settings in `profile.yaml`

### Profile example

```yaml
# ~/.config/kui/profiles/my-profile/profile.yaml
model: gpt-4o
provider: openai
thinking: medium
tools:
  include: [bash, read, write]
  exclude: [webfetch]
system_prompt: |
  You are a senior Go developer.
```

## Dynamic extensions

kui supports runtime-loaded extensions via MCP-style subprocess protocol:

```yaml
# ~/.config/kui/extensions.yaml
extensions:
  paths:
    - ~/.config/kui/extensions/my-ext
```

Each extension is a executable that speaks JSON-RPC 2.0 over stdio (`kui-ext/1` protocol). Extensions register tools that become available in the agent loop.

## Building

```bash
go build -o kui ./cmd/kui/
go test ./...
```

## Architecture

```
cmd/kui/           CLI entry point, flag parsing, profile commands
internal/
  core/            Provider, Tool, Extension interfaces
  agent/           Agent loop (provider → tools → provider cycle)
  runtime/         Build/Reload/Close lifecycle
  adapters/
    providers/     OpenAI, OpenCode adapters + registry
    profile/       Profile loader (3-layer resolution)
    extensions/    Compiled-in extension registry
    tools/         Built-in tools (bash, read, write, glob, grep, web)
    skills/        Skill index (local + remote)
    mcp/           MCP server manager
    store/         Session store
  extensions/
    dynamic/       Runtime extension loader (subprocess)
    example/       Example compiled-in extension
  mcp/             MCP protocol client
  tui/             Bubble Tea TUI with profile switcher
```

## License

MIT
