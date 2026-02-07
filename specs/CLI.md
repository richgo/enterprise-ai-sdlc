# CLI Component Specification

## Overview

The `eas` CLI is the main entry point for engineers. It provides commands for initializing features, managing tasks, and running agents.

## Commands

### flo init
Initialize a new feature workspace.

```bash
`flo init <feature-name> [--backend claude|copilot]
```

Creates:
- `.flo/config.yaml`
- `.flo/SPEC.md` (template)
- `.flo/tasks/manifest.json`

### flo task
Task management subcommands.

```bash
`flo task list [--status pending|in_progress|complete] [--repo <name>]
`flo task create <title> [--repo <name>] [--deps <id,...>] [--priority <n>]
`flo task get <id>
`flo task claim <id>
`flo task complete <id>
```

### flo status
Show feature status overview.

```bash
`flo status
```

Displays:
- Feature name and backend
- Task counts by status
- Ready tasks (can be started)
- Recent activity

### flo work
Start agent work on a task.

```bash
`flo work <task-id> [--backend claude|copilot]
```

### eas mcp
MCP server for Claude integration.

```bash
`flo mcp serve [--port <n>]
```

## Acceptance Criteria

### flo init
- [ ] Creates .flo directory structure
- [ ] Writes default config.yaml
- [ ] Creates empty SPEC.md template
- [ ] Initializes empty task manifest
- [ ] Returns error if already initialized

### flo task list
- [ ] Lists all tasks by default
- [ ] Filters by status
- [ ] Filters by repo
- [ ] Shows task ID, title, status, deps

### flo task create
- [ ] Creates task with unique ID
- [ ] Validates deps exist
- [ ] Saves to manifest
- [ ] Returns new task ID

### flo status
- [ ] Shows feature name
- [ ] Shows task counts
- [ ] Shows ready tasks
- [ ] Works without tasks
