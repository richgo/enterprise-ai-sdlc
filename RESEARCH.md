# Enterprise AI SDLC - Research Notes

## Overview

Research into autonomous AI coding patterns for building an enterprise-grade CLI/SDK approach to **research → design → build → test** workflows.

**Key sources:**
- Ralph Loop (snarktank/ralph) - 9.3k ⭐
- Gastown (steveyegge/gastown) - Multi-agent orchestrator
- Beads (steveyegge/beads) - Git-backed task persistence

---

## 1. Ralph Loop

**Repo:** https://github.com/snarktank/ralph

### Core Concept
Autonomous AI agent loop that runs repeatedly until all PRD items are complete. Each iteration spawns a **fresh agent instance** with clean context.

### Key Architecture
```
┌─────────────────────────────────────────────┐
│                  ralph.sh                    │
│  (bash loop - spawns fresh AI each time)    │
└─────────────────┬───────────────────────────┘
                  │
    ┌─────────────┴─────────────┐
    │     Memory Persistence     │
    ├────────────────────────────┤
    │  • prd.json (task status)  │
    │  • progress.txt (learnings)│
    │  • git history (commits)   │
    │  • AGENTS.md (conventions) │
    └────────────────────────────┘
```

### Workflow
1. Create PRD (markdown) via skill
2. Convert to `prd.json` (structured user stories)
3. Run `ralph.sh` loop:
   - Pick highest priority story where `passes: false`
   - Implement that single story
   - Run quality checks (typecheck, tests) ← **TDD enforced**
   - Commit if checks pass
   - Update `prd.json` to `passes: true`
   - Append learnings to `progress.txt`
   - Repeat until all pass

### Key Files
| File | Purpose |
|------|---------|
| `prd.json` | User stories with `passes` status |
| `progress.txt` | Append-only learnings |
| `AGENTS.md` | Conventions for future iterations |

### Strengths for Enterprise
- ✅ Spec-driven development (PRD → tasks)
- ✅ TDD mandatory (typecheck + tests must pass)
- ✅ Git-native persistence
- ✅ Clean context prevents hallucination drift
- ✅ Small, atomic tasks (context window friendly)

### Weaknesses
- ❌ Single repo focus
- ❌ No cross-project coordination
- ❌ Limited task graph (flat list, no dependencies)

---

## 2. Gastown (Steve Yegge)

**Repo:** https://github.com/steveyegge/gastown

### Core Concept
Multi-agent workspace manager. Coordinates 20-30 Claude Code agents working on different tasks. Work state persists in git-backed "hooks".

### Architecture
```
┌─────────────────────────────────────────────────┐
│                    TOWN                          │
│               (~/gt/ workspace)                  │
├─────────────────────────────────────────────────┤
│                                                  │
│   ┌─────────┐    ┌─────────┐    ┌─────────┐    │
│   │  Mayor  │    │  Rig A  │    │  Rig B  │    │
│   │(coord.) │    │(Project)│    │(Project)│    │
│   └────┬────┘    └────┬────┘    └────┬────┘    │
│        │              │              │          │
│        │         ┌────┴────┐    ┌────┴────┐    │
│        │         │Polecats │    │Polecats │    │
│        │         │(workers)│    │(workers)│    │
│        │         └────┬────┘    └────┬────┘    │
│        │              │              │          │
│        └──────────────┴──────────────┘          │
│                    BEADS                         │
│             (task ledger/graph)                  │
└─────────────────────────────────────────────────┘
```

### Terminology
| Concept | Meaning |
|---------|---------|
| **Mayor** | Primary AI coordinator with full context |
| **Town** | Workspace directory containing all projects |
| **Rig** | Project container wrapping a git repo |
| **Crew** | Your personal workspace within a rig |
| **Polecats** | Ephemeral worker agents (spawn, complete, disappear) |
| **Hooks** | Git worktree-based persistent storage |
| **Convoy** | Work tracking unit (bundles multiple beads) |

### Strengths for Enterprise
- ✅ Multi-repo coordination
- ✅ Scales to 20-30 agents
- ✅ Git-backed persistence (hooks)
- ✅ Agent identity/mailbox system
- ✅ Uses Beads for task persistence

### Weaknesses
- ❌ Heavy (Go binary, tmux, sqlite)
- ❌ Tight coupling to Claude Code
- ❌ Complex terminology/learning curve

---

## 3. Beads (Steve Yegge)

**Repo:** https://github.com/steveyegge/beads

### Core Concept
Distributed, git-backed **graph issue tracker** for AI agents. Replaces markdown plans with dependency-aware structured data.

### Storage
```
your-project/
├── .beads/
│   ├── issues.jsonl     # All issues as JSONL
│   ├── meta.json        # Repo config
│   └── .cache/          # SQLite local cache
└── ...
```

### Key Features
- **Git as Database**: Issues stored as JSONL, versioned with code
- **Dependency Graph**: Tasks can block/relate to each other
- **Hash-based IDs**: `bd-a1b2` format prevents merge conflicts
- **Hierarchical IDs**: Epics → Tasks → Subtasks (`bd-a3f8.1.1`)
- **Agent-optimized**: JSON output, `bd ready` shows unblocked tasks
- **Compaction**: Summarizes old closed tasks (context window friendly)

### Commands
```bash
bd init                    # Initialize in project
bd create "Title" -p 0     # Create P0 task
bd ready                   # List tasks with no blockers
bd dep add <child> <parent> # Link dependencies
bd show <id>               # View task details
```

### Strengths for Enterprise
- ✅ No external DB (git-native)
- ✅ Dependency graph built-in
- ✅ Works across branches
- ✅ Agent-optimized output
- ✅ Stealth mode for shared repos

### Weaknesses
- ❌ JSONL not human-readable like markdown
- ❌ Requires `bd` CLI everywhere
- ❌ Single-repo focus (no native cross-repo linking)

---

## 4. Design Recommendations for Enterprise AI SDLC

### Your Requirements Mapping

| Requirement | Solution |
|-------------|----------|
| Spec-driven + TDD mandatory | Ralph's PRD → prd.json pattern |
| Save agent tasks/progress | Beads or custom `.tasks/` directory |
| Multiple repos (Android/iOS/Web/Services) | Cross-repo linking layer |
| Lightweight graph view | Beads-style dependency tracking |
| No external DBs | Git-native storage |
| Prefer .md files | Hybrid: .md for specs, .json for machine state |

### Proposed Architecture

```
Feature: "User Authentication"
├── .feature/
│   ├── SPEC.md              # Human-readable spec (PRD)
│   ├── tasks.json           # Machine-readable task graph
│   ├── progress.md          # Append-only learnings
│   └── repos.json           # Cross-repo links
│
├── repos.json example:
│   {
│     "feature": "user-auth",
│     "repos": [
│       {"name": "mobile-android", "path": "features/auth", "tasks": ["ua-001", "ua-002"]},
│       {"name": "mobile-ios", "path": "features/auth", "tasks": ["ua-003", "ua-004"]},
│       {"name": "web-app", "path": "src/auth", "tasks": ["ua-005"]},
│       {"name": "api-gateway", "path": "services/auth", "tasks": ["ua-006", "ua-007"]}
│     ]
│   }
```

### Cross-Repo Linking Options

**Option A: Central Feature Repo**
```
features/
├── user-auth/
│   ├── SPEC.md
│   ├── tasks.json
│   └── links/
│       ├── android.md  → symlink or path ref
│       ├── ios.md
│       └── web.md
```

**Option B: Distributed with Sync**
Each repo has `.feature/user-auth/` that syncs via git submodule or simple file refs.

**Option C: Beads + Custom Cross-Repo Layer**
Use Beads for task tracking, add a lightweight `feature-graph.json` that maps beads across repos.

### Recommended Stack

```
┌────────────────────────────────────────────────────┐
│                    CLI / SDK                        │
│         (flo / eas CLI)             │
├────────────────────────────────────────────────────┤
│                                                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐         │
│  │ Research │  │  Design  │  │  Build   │  Test   │
│  │  Agent   │  │  Agent   │  │  Agent   │  Agent  │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  ─┬───  │
│       │             │             │          │     │
│       └─────────────┴─────────────┴──────────┘     │
│                         │                          │
│              ┌──────────┴──────────┐               │
│              │   Feature Context    │               │
│              ├─────────────────────┤               │
│              │ • SPEC.md (human)   │               │
│              │ • tasks.json (graph)│               │
│              │ • repos.json (links)│               │
│              │ • progress.md       │               │
│              └─────────────────────┘               │
│                         │                          │
│              ┌──────────┴──────────┐               │
│              │    Git (storage)     │               │
│              └─────────────────────┘               │
└────────────────────────────────────────────────────┘
```

### File Formats

**SPEC.md** (human-readable, PRD-style)
```markdown
# Feature: User Authentication

## Overview
OAuth2 + MFA for all platforms.

## User Stories
1. As a user, I can log in with Google
2. As a user, I can enable 2FA
...

## Acceptance Criteria
- [ ] All platforms share auth token format
- [ ] Session expires after 24h
```

**tasks.json** (machine-readable, Beads-inspired)
```json
{
  "feature": "user-auth",
  "tasks": [
    {
      "id": "ua-001",
      "title": "Implement Google OAuth flow",
      "repo": "mobile-android",
      "status": "pending",
      "priority": 0,
      "deps": [],
      "spec_ref": "SPEC.md#user-stories-1"
    },
    {
      "id": "ua-002", 
      "title": "Add token storage",
      "repo": "mobile-android",
      "status": "pending",
      "deps": ["ua-001"]
    }
  ]
}
```

**repos.json** (cross-repo links)
```json
{
  "feature": "user-auth",
  "repos": {
    "mobile-android": {
      "url": "git@github.com:org/android.git",
      "branch": "feature/user-auth",
      "tasks": ["ua-001", "ua-002"]
    },
    "mobile-ios": {
      "url": "git@github.com:org/ios.git", 
      "branch": "feature/user-auth",
      "tasks": ["ua-003", "ua-004"]
    }
  }
}
```

---

## 5. Next Steps

1. **Decide on architecture**: Central feature repo vs distributed?
2. **Prototype CLI**: `flo init`, `flo task create`, `flo status`
3. **Define agent handoff protocol**: How does Research → Design → Build flow?
4. **TDD integration**: How to enforce tests before task completion?
5. **Graph visualization**: Terminal UI (like beads_viewer) or web?

---

---

## 6. Backlog.md (MrLesk/Backlog.md)

**Repo:** https://github.com/MrLesk/Backlog.md

### Core Concept
Markdown-native task manager for Git repos. Each task is a plain `.md` file. Zero-config CLI with terminal Kanban and web UI.

### Key Difference from Beads
- **Beads**: JSONL storage, hash IDs, graph-focused
- **Backlog.md**: Pure markdown files, human-readable, AI-agent-ready

### Storage Format
```
backlog/
├── tasks/
│   ├── task-1 - Implement OAuth.md
│   ├── task-2 - Add token validation.md
│   └── task-3 - Build login UI.md
├── docs/
│   └── architecture.md
└── config.yaml
```

### Task File Format
```markdown
# Task-1: Implement OAuth

**Status:** In Progress
**Priority:** High
**Assignee:** @agent-1
**Labels:** auth, backend
**Dependencies:** none

## Description
Implement OAuth2 token exchange with Google.

## Acceptance Criteria
- [ ] Token exchange endpoint works
- [ ] Refresh tokens supported
- [ ] Error handling complete

## Definition of Done
- [ ] Tests pass
- [ ] Code reviewed
- [ ] Docs updated

## Plan
1. Research OAuth2 flow
2. Implement endpoint
3. Add tests

## Notes
Started implementation 2026-02-04.
```

### Features
- 📝 **Markdown-native** — human-readable, git-friendly
- 🤖 **AI-Ready** — MCP integration for Claude Code, Gemini CLI, Codex
- 📊 **Terminal Kanban** — `backlog board`
- 🌐 **Web UI** — `backlog browser`
- 🔗 **Dependencies** — `--dep task-1,task-2`
- ✅ **Definition of Done** — reusable checklists
- 🏷️ **Labels, Priority, Assignee** — full metadata

### CLI
```bash
backlog init "My Project"
backlog task create "Feature" --priority high --dep task-1
backlog task list --status "To Do"
backlog task edit 7 --check-ac 1  # Mark acceptance criterion done
backlog board                      # Terminal Kanban
backlog browser                    # Web UI
```

### MCP Integration
```json
{
  "mcpServers": {
    "backlog": {
      "command": "backlog",
      "args": ["mcp", "start"]
    }
  }
}
```

### Strengths for Enterprise
- ✅ Pure markdown (Rich's preference!)
- ✅ Human-readable in any editor
- ✅ AI-agent-ready out of the box
- ✅ Dependencies built-in
- ✅ Definition of Done = TDD alignment
- ✅ MCP integration for modern AI tools

### Weaknesses
- ❌ Node.js based (not Go)
- ❌ Single-repo focus (no native cross-repo)
- ❌ No worktree integration

### Relevance to Our Design
Backlog.md's file format is excellent inspiration for our TaskRegistry:
- Use markdown for human readability
- Structured frontmatter for machine parsing
- Acceptance criteria = TDD scenarios
- Dependencies = DAG edges

**Hybrid approach:**
- `SPEC.md` — feature specification (like Backlog.md docs)
- `tasks/*.md` — individual task files (like Backlog.md tasks)
- `manifest.json` — machine-readable index for fast queries
- Cross-repo links in frontmatter

---

## References

- [Ralph Loop](https://github.com/snarktank/ralph) - 9.3k ⭐
- [Gastown](https://github.com/steveyegge/gastown) - Multi-agent orchestrator
- [Beads](https://github.com/steveyegge/beads) - Git-backed task tracker
- [Beads Viewer](https://github.com/Dicklesworthstone/beads_viewer) - Terminal UI for Beads
- [Backlog.md](https://github.com/MrLesk/Backlog.md) - Markdown-native task manager
- [awesome-ralph](https://github.com/snwfdhmp/awesome-ralph) - Curated Ralph resources
