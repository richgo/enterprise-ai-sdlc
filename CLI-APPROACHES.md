# CLI Approaches: Extension vs New

Evaluating whether to extend an existing CLI (Copilot CLI, Claude Code) or build a new `eas` CLI.

**Goal:** Minimize engineer cognitive load while providing enterprise SDLC capabilities.

---

## The Three Options

| Approach | Example | Cognitive Load | Control | Build Effort |
|----------|---------|----------------|---------|--------------|
| **A. Extend Existing** | Claude Code + MCP | Low | Medium | Low |
| **B. New CLI** | `eas` standalone | High | Full | High |
| **C. Thin Wrapper** | `eas` → delegates | Medium | Full | Medium |

---

## Option A: Extend Existing CLI (Conductor Pattern)

### How Conductor Works (Anthropic Internal)
Conductor doesn't replace Claude Code—it **augments** it:
- Custom MCP servers provide enterprise tools
- `context/` directories inject project-specific context
- Claude Code remains the agent runtime

### Implementation with Claude Code

```bash
# Engineer's workflow - uses familiar claude command
claude --mcp-config eas-tools.json "implement auth feature per SPEC.md"
```

**eas-tools.json** (MCP server config):
```json
{
  "mcpServers": {
    "eas": {
      "command": "eas",
      "args": ["mcp", "serve"],
      "env": {
        "EAS_WORKSPACE": "/path/to/feature"
      }
    }
  }
}
```

**EAS MCP Server provides:**
```
Tools:
  - eas_task_list      → List tasks from tasks.json
  - eas_task_start     → Claim a task, create worktree
  - eas_task_complete  → Run tests, commit if pass
  - eas_spec_read      → Read SPEC.md for context
  - eas_deps_check     → Check task dependencies
  - eas_cross_repo     → Query other repos in feature
```

### Implementation with Copilot SDK

```go
// Register EAS tools with Copilot session
session.RegisterTool(copilot.DefineTool("eas_task_list", "List available tasks",
    func(params TaskListParams, inv copilot.ToolInvocation) (any, error) {
        return eas.ListTasks(params.Status)
    }))

session.RegisterTool(copilot.DefineTool("eas_run_tests", "Run tests for current task",
    func(params RunTestsParams, inv copilot.ToolInvocation) (any, error) {
        return eas.RunTests(params.TaskID)
    }))

// Enforce TDD via pre-tool hook
client.SetPreToolUseHandler(func(input copilot.PreToolUseHookInput, inv copilot.HookInvocation) (*copilot.PreToolUseHookOutput, error) {
    if input.ToolName == "git_commit" {
        // Run tests first
        result, _ := eas.RunTests(currentTaskID)
        if !result.AllPassed {
            return &copilot.PreToolUseHookOutput{
                PermissionDecision: "deny",
                PermissionDecisionReason: "Tests must pass before commit",
            }, nil
        }
    }
    return nil, nil
})
```

### Pros
- ✅ **Zero new CLI to learn** — engineers use `claude` or `copilot` they know
- ✅ **Leverage existing agent runtime** — battle-tested orchestration
- ✅ **MCP is standard** — portable across Claude Code, Gemini CLI, etc.
- ✅ **Low build effort** — just build tools, not orchestration

### Cons
- ❌ **Less control over UX** — can't customize prompts/flow deeply
- ❌ **Dependency on external CLI** — version compatibility issues
- ❌ **Tool-limited workflow** — complex multi-step flows are awkward

---

## Option B: New Standalone CLI (`eas`)

### Engineer Workflow

```bash
# New CLI to learn
eas init my-feature
eas task create "Implement OAuth" --repo android --deps auth-api
eas task start ua-001
eas agent run --tdd    # Runs agent loop with TDD enforcement
eas task complete
eas status
```

### Architecture

```
┌────────────────────────────────────┐
│           eas CLI                   │
├────────────────────────────────────┤
│  Commands:                          │
│  • init, task, agent, status        │
│                                     │
│  Agent Runtime:                     │
│  • Copilot SDK (Go) OR              │
│  • Direct LLM API calls             │
│                                     │
│  Storage:                           │
│  • .eas/ directory                  │
│  • Git-native (worktrees, commits)  │
└────────────────────────────────────┘
```

### Pros
- ✅ **Full control** — custom UX, flows, prompts
- ✅ **Unified experience** — one tool for everything
- ✅ **Enterprise features built-in** — cross-repo, DAGs, TDD
- ✅ **No external dependencies** — self-contained

### Cons
- ❌ **New tool to learn** — cognitive load
- ❌ **High build effort** — agent orchestration from scratch
- ❌ **Reinventing wheels** — session management, streaming, etc.

---

## Option C: Thin Wrapper (Recommended)

### Concept
`eas` is a **thin orchestration layer** that:
1. Manages feature/task state (git-native)
2. Delegates agent work to Claude Code or Copilot CLI
3. Enforces TDD and cross-repo coordination

### Engineer Workflow

```bash
# Familiar init/status commands
eas init my-feature
eas task create "Implement OAuth" --repo android

# Delegates to Claude Code with pre-configured context
eas work ua-001
# Equivalent to: claude --mcp-config .eas/mcp.json --system-prompt "$(eas context ua-001)"

# Or use Claude directly with EAS context
claude --mcp-config .eas/mcp.json "work on task ua-001"
```

### Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      eas CLI                             │
│           (thin orchestration layer)                     │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  State Management:          Agent Delegation:            │
│  • eas init                 • eas work → claude/copilot  │
│  • eas task create/list     • eas agent → parallel runs  │
│  • eas status               • eas review → PR creation   │
│  • eas sync (cross-repo)                                 │
│                                                          │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  MCP Server (eas mcp serve):                            │
│  • eas_task_* tools                                      │
│  • eas_test_* tools (TDD enforcement)                   │
│  • eas_repo_* tools (cross-repo)                        │
│                                                          │
└──────────────────────┬──────────────────────────────────┘
                       │
          ┌────────────┴────────────┐
          │                         │
    ┌─────▼─────┐            ┌─────▼─────┐
    │  Claude   │            │  Copilot  │
    │   Code    │            │    CLI    │
    └───────────┘            └───────────┘
```

### Key Commands

```bash
# Project/Task Management (eas-native)
eas init <feature-name>              # Create .eas/ structure
eas task create <title> [--deps]     # Add task to DAG
eas task list [--status] [--repo]    # Query tasks
eas status                           # Overview dashboard

# Agent Delegation (wraps claude/copilot)
eas work <task-id>                   # Start agent on task
eas work --parallel 5                # Run 5 agents on ready tasks
eas agent config                     # Configure which CLI to use

# Cross-Repo Coordination
eas repo add <name> <url>            # Link repo to feature
eas sync                             # Sync task state across repos

# CI/CD Integration
eas ci check                         # Verify all tasks pass
eas ci report                        # Generate status report
```

### MCP Tools (provided by `eas mcp serve`)

```typescript
// Task Management
eas_task_list(status?: string)       // List tasks
eas_task_get(id: string)             // Get task details
eas_task_claim(id: string)           // Mark task in-progress
eas_task_complete(id: string)        // Mark done (runs tests first)

// TDD Enforcement  
eas_test_run(taskId: string)         // Run task's tests
eas_test_status(taskId: string)      // Check if tests pass
eas_pre_commit_check()               // Gate commits on test status

// Context Injection
eas_spec_read()                      // Read SPEC.md
eas_context_get(taskId: string)      // Full context for task
eas_deps_status(taskId: string)      // Check dependency status

// Cross-Repo
eas_repo_list()                      // List linked repos
eas_repo_task_status(repo: string)   // Tasks in other repos
```

### Pros
- ✅ **Low cognitive load** — `eas` for state, familiar `claude`/`copilot` for work
- ✅ **Best of both worlds** — control where needed, delegation where not
- ✅ **Portable** — works with Claude Code, Copilot, or future CLIs
- ✅ **Incremental adoption** — can use MCP tools without `eas` commands

### Cons
- ⚠️ **Two CLIs** — but clear separation of concerns
- ⚠️ **Medium build effort** — MCP server + state management

---

## Cognitive Load Comparison

### Learning Curve

| Approach | New Concepts | Familiar Concepts |
|----------|--------------|-------------------|
| **Extend** | MCP config, context dirs | claude/copilot CLI |
| **New CLI** | All of eas (10+ commands) | Git |
| **Wrapper** | eas init/task/status (5 cmds) | claude/copilot CLI |

### Daily Workflow

**Extend (Conductor-style):**
```bash
cd features/auth
claude "implement next task from SPEC.md"  # EAS tools auto-available via MCP
```

**New CLI:**
```bash
eas task start ua-001
eas agent run --tdd
eas task complete
```

**Wrapper (Recommended):**
```bash
eas work ua-001           # Sets up context, launches claude
# ... claude does the work ...
eas status                # Check progress
```

---

## Recommendation

**Go with Option C (Thin Wrapper)** because:

1. **Minimal new learning** — engineers learn 5 `eas` commands, keep using claude/copilot
2. **TDD enforcement** — MCP tools gate commits on tests
3. **Cross-repo works** — `eas` handles coordination, agent handles code
4. **Future-proof** — swap underlying CLI without changing workflow
5. **Incremental rollout** — start with MCP tools only, add commands later

### Implementation Priority

1. **Phase 1: MCP Server** (`eas mcp serve`)
   - Task management tools
   - TDD enforcement tools
   - Works with vanilla claude/copilot

2. **Phase 2: State Commands** (`eas init/task/status`)
   - Git-native storage
   - Cross-repo linking
   - DAG visualization

3. **Phase 3: Orchestration** (`eas work/agent`)
   - Parallel agent coordination
   - Automatic task assignment
   - Progress monitoring

---

## File Structure

```
.eas/
├── config.json           # Feature config, linked repos
├── SPEC.md               # Human-readable specification
├── tasks/
│   ├── manifest.json     # Task DAG (machine-readable)
│   ├── ua-001.md         # Task details (human-readable)
│   └── ua-002.md
├── context/              # Conductor-style context injection
│   ├── CONSTITUTION.md   # Rules for all agents
│   └── architecture.md   # Technical context
├── mcp.json              # Auto-generated MCP config
└── worktrees/            # Git worktrees per task
    ├── ua-001/
    └── ua-002/
```

---

## Next Steps

1. [ ] Prototype `eas mcp serve` with core tools
2. [ ] Test with Claude Code on real feature
3. [ ] Add TDD hooks (pre-commit test enforcement)
4. [ ] Build `eas init/task/status` commands
5. [ ] Cross-repo sync mechanism
