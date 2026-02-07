# Flo v2: Developer Flow & BYO AI

## Vision

Flo keeps developers in the zone. Whether you're hands-on-keyboard or orchestrating agents, Flo maintains your flow state by handling the context switching between AI providers seamlessly.

## Core Principles

1. **Flow First** - Minimize interruptions, maximize productivity
2. **Human or Agent** - Work ON tasks or IN the loop orchestrating them
3. **BYO AI** - Aggregate all AI CLIs, switch providers/models freely
4. **Quota Aware** - Auto-failover when limits hit, efficient quota usage

## Supported Backends

| Backend | CLI | Status |
|---------|-----|--------|
| Claude Code | `claude` | ✅ Implemented |
| GitHub Copilot | `copilot` | ✅ Implemented |
| OpenAI Codex | `codex` | 🔄 Planned |
| Google Gemini | `gemini` | 🔄 Planned |

## Task-Level Model Control

Tasks specify their model/provider in frontmatter:

```markdown
---
id: t-001
model: claude/sonnet
fallback: copilot/gpt-4
type: refactor
---
# Task Title
```

## Task Types & Default Models

| Type | Description | Default Model | Reasoning |
|------|-------------|---------------|-----------|
| design | Architecture, specs | claude/opus | Deep thinking |
| build | New features | claude/sonnet | Balance |
| refactor | Code cleanup | copilot/gpt-4 | Fast iterations |
| test | Write tests | claude/sonnet | Accuracy |
| fix | Bug fixes | copilot/gpt-4 | Quick turnaround |
| docs | Documentation | claude/haiku | Simple output |
| review | Code review | claude/sonnet | Nuanced feedback |

## Config Schema

```yaml
# .flo/config.yaml
feature: my-feature

defaults:
  model: claude/sonnet
  fallback: copilot/gpt-4
  
taskTypes:
  design:
    model: claude/opus
    thinking: extended
  build:
    model: claude/sonnet
  refactor:
    model: copilot/gpt-4
  test:
    model: claude/sonnet
  fix:
    model: copilot/gpt-4
  docs:
    model: claude/haiku
  review:
    model: claude/sonnet

backends:
  claude:
    quotaCheck: true
    premiumLimit: 50  # per hour
  copilot:
    quotaCheck: true
  codex:
    enabled: false
  gemini:
    enabled: false

quota:
  autoFailover: true
  retryAfter: 60  # seconds
```

## Quota Management

1. Track usage per backend (premium requests, tokens)
2. Detect quota exhaustion (429 responses, rate limits)
3. Auto-failover to fallback model
4. Resume primary when quota resets

## CLI Changes

```bash
# Task creation with type
flo task create "Refactor auth module" --type refactor

# Override model for a task
flo task create "Design new API" --type design --model gemini/pro

# Work respects task model
flo work t-001  # Uses model from task.md

# Check quota status
flo quota status

# Force specific backend
flo work t-001 --backend codex
```

## Success Criteria

1. Support 4+ backends (claude, copilot, codex, gemini)
2. Task.md contains model/provider specification
3. Auto-failover on quota exhaustion works
4. Config defines defaults by task type
5. `flo quota status` shows usage across backends
