# Flo AI SDLC

AI-powered spec-driven, test-driven development for individuals, teams & parallel agents.

[![CI](https://github.com/richgo/flo/actions/workflows/ci.yml/badge.svg)](https://github.com/richgo/flo/actions/workflows/ci.yml)
[![Tests](https://img.shields.io/badge/tests-140%20passing-brightgreen)]()
[![Go](https://img.shields.io/badge/go-1.24-blue)]()

## Overview

EAS orchestrates AI agents for structured development workflows:
- **Spec-Driven**: Start with SPEC.md, break into tasks
- **Test-Driven**: Agents must pass tests before completing tasks
- **Multi-Backend**: Claude Code or GitHub Copilot SDK
- **Git-Native**: All state stored in `.flo/` directory

## Installation

```bash
go install github.com/richgo/flo/cmd/eas@latest
```

## Quick Start

```bash
# Initialize a feature workspace
`flo init user-auth --backend claude

# Edit the specification
vim .flo/SPEC.md

# Create tasks
`flo task create "Implement OAuth" --repo android
`flo task create "Add token storage" --repo android --deps t-001
`flo task create "iOS OAuth" --repo ios

# Check status
`flo status

# Start agent work on a task
`flo work t-001
```

## Commands

| Command | Description |
|---------|-------------|
| `flo init <feature>` | Initialize workspace |
| `flo task list` | List all tasks |
| `flo task create <title>` | Create a task |
| `flo task get <id>` | Get task details |
| `flo status` | Show workspace status |
| `flo work <task-id>` | Run agent on task |
| `flo spec validate [path]` | Validate SPEC.md format |
| `flo config show` | Show configuration and secrets (masked) |
| `eas mcp serve` | Start MCP server |

## Architecture

```
.flo/
├── config.yaml       # Feature configuration
├── SPEC.md           # Feature specification
├── tasks/
│   └── manifest.json # Task DAG
└── mcp.json          # Auto-generated MCP config
```

### Backends

- **Claude**: Uses Claude Code CLI via MCP
- **Copilot**: Uses GitHub Copilot SDK (Go)

Both backends share the same tool definitions and TDD enforcement.

## Tools (MCP)

EAS exposes these tools to agents:

| Tool | Description |
|------|-------------|
| `eas_task_list` | List tasks with filters |
| `eas_task_get` | Get task details |
| `eas_task_claim` | Claim a task |
| `eas_task_complete` | Complete task (runs tests) |
| `eas_run_tests` | Run tests for task |
| `eas_spec_read` | Read SPEC.md |

## Development

### Environment Variables

Flo supports the following environment variables for configuration:

| Variable | Description | Required |
|----------|-------------|----------|
| `CLAUDE_API_KEY` | API key for Claude backend | Yes (if using Claude) |
| `COPILOT_TOKEN` | GitHub Copilot token | Yes (if using Copilot) |
| `FLO_BACKEND` | Default backend (claude/copilot) | No (defaults to claude) |
| `FLO_MODEL` | Default model to use | No |

You can set these variables in:
- System environment variables
- `.env` file in project root
- `.flo/.env` file in the workspace

Example `.env` file:
```bash
CLAUDE_API_KEY=sk-ant-api-xxxxx
COPILOT_TOKEN=ghp_xxxxx
FLO_BACKEND=claude
```

View current configuration with: `flo config show`

### Building from Source

```bash
# Using Make (recommended)
make build      # Build binary to bin/flo
make test       # Run tests
make lint       # Run linter
make all        # Run lint, test, and build
make install    # Install to GOPATH/bin
make clean      # Remove build artifacts

# Or using Go directly
go build -o bin/flo ./cmd/flo
go test ./...
```

### Testing Locally

```bash
# Build and test
make all

# Test the binary
./bin/flo init test-feature
./bin/flo task create "Test task"
./bin/flo status
```

## Documentation

- [Architecture](ARCHITECTURE.md)
- [CLI Approaches](CLI-APPROACHES.md)
- [Dual Backend Design](DUAL-BACKEND.md)
- [Copilot SDK Deep Dive](COPILOT-SDK.md)
- [Research Notes](RESEARCH.md)

## License

MIT
