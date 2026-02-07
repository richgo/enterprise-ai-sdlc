# Integration Research: Conductor, Spec-Kit & OpenSpec

## Overview

Research into how our Enterprise AI SDLC architecture can integrate with or learn from existing spec-driven development tools.

---

## 1. Conductor (Gemini CLI Extension)

**Repo:** https://github.com/gemini-cli-extensions/conductor
**Philosophy:** "Measure twice, code once"

### What It Does

Conductor turns Gemini CLI into a context-driven development system with a strict protocol:

```
Context → Spec & Plan → Implement
```

### Key Artifacts

| Artifact | Purpose |
|----------|---------|
| `conductor/product.md` | Product context (users, goals, features) |
| `conductor/product-guidelines.md` | Standards (prose style, branding) |
| `conductor/tech-stack.md` | Technical preferences |
| `conductor/workflow.md` | Team preferences (TDD, commit strategy) |
| `conductor/tracks/<id>/spec.md` | Feature specification |
| `conductor/tracks/<id>/plan.md` | Actionable task list |
| `conductor/tracks.md` | Track status overview |

### Commands

```
/conductor:setup      # One-time project setup (context files)
/conductor:newTrack   # Start feature/bug track → generates spec.md + plan.md
/conductor:implement  # Execute tasks from plan.md
/conductor:status     # Progress overview
/conductor:revert     # Git-aware revert by track/phase/task
/conductor:review     # Review against guidelines
```

### Workflow (TDD-Aligned)

1. **Setup** — Define product, guidelines, tech stack, workflow
2. **New Track** — Create spec + plan for a feature
3. **Implement** — Agent works through plan, checking off tasks
4. **Manual Verification** — Human verifies at end of each phase

### Valuable Concepts for Our Architecture

| Concept | Value | Integration Notes |
|---------|-------|-------------------|
| **Persistent Context** | Product/tech/workflow docs guide every agent | Add `context/` directory to our feature structure |
| **Track = Feature** | Isolates work into discrete units | Maps to our `feature/{id}/` pattern |
| **Spec + Plan Split** | Separates "what" from "how" | Already in our design (SPEC.md + tasks/*.md) |
| **Git-Aware Revert** | Understands logical units (track → phase → task) | Add to WorktreeManager cleanup |
| **Phase Verification** | Human checkpoint between phases | Add optional `--verify` flag to `eas run` |
| **Code Styleguides** | `conductor/code_styleguides/` per language | Add `styleguides/` to workspace |

### Gaps Conductor Has (We Can Fill)

- ❌ Single-repo only (no cross-repo coordination)
- ❌ Gemini CLI specific (not portable)
- ❌ No parallel agent support
- ❌ Flat task list (no DAG dependencies)

---

## 2. GitHub Spec-Kit

**Repo:** https://github.com/github/spec-kit
**Philosophy:** "Specifications become executable"

### What It Does

A comprehensive Python CLI (`specify`) that scaffolds a project for spec-driven development with slash commands for any AI assistant.

### Key Features

- **Agent-agnostic** — Works with 20+ AI assistants (Claude, Gemini, Copilot, Cursor, Codex, etc.)
- **Slash commands** — `/speckit.constitution`, `/speckit.specify`, `/speckit.plan`, `/speckit.tasks`, `/speckit.implement`
- **Phase-based** — Constitution → Specify → Plan → Tasks → Implement
- **Branch per feature** — Auto-creates `001-feature-name` branches

### Workflow

```
1. specify init <project>         # Bootstrap project
2. /speckit.constitution          # Set governing principles
3. /speckit.specify "description" # Create functional spec
4. /speckit.plan "tech choices"   # Create technical plan
5. /speckit.tasks                 # Break down into task list
6. /speckit.implement             # Execute all tasks
```

### Directory Structure

```
.specify/
├── memory/
│   └── constitution.md           # Project principles
├── scripts/
│   └── *.sh                      # Helper scripts
├── specs/
│   └── 001-feature/
│       └── spec.md               # Feature spec
└── templates/
    ├── plan-template.md
    ├── spec-template.md
    └── tasks-template.md
```

### Valuable Concepts for Our Architecture

| Concept | Value | Integration Notes |
|---------|-------|-------------------|
| **Constitution** | Governing principles guide all decisions | Add `CONSTITUTION.md` to workspace root |
| **Agent-agnostic** | Same workflow works across tools | Already our goal with Copilot SDK |
| **Templates** | Consistent spec/plan/task formats | Add `templates/` directory |
| **Clarify Command** | Quiz mode to find underspecified areas | Add `eas clarify` before planning |
| **Analyze Command** | Cross-artifact consistency check | Add `eas analyze` validation step |
| **Feature Branches** | `001-feature-name` naming convention | Adopt for our feature branches |

### Gaps Spec-Kit Has (We Can Fill)

- ❌ No parallel agent execution
- ❌ No dependency graph (flat task list)
- ❌ No cross-repo coordination
- ❌ No worktree isolation per task
- ❌ Python-based (we're Go)

---

## 3. OpenSpec (Fission-AI)

**Repo:** https://github.com/Fission-AI/OpenSpec
**Philosophy:** "Fluid not rigid, iterative not waterfall"

### What It Does

A lightweight Node.js-based spec framework emphasizing iteration over rigid phases. Each change gets its own folder with proposal, specs, design, and tasks.

### Key Features

- **Change-based** — Each feature/fix is a "change" with its own folder
- **Artifact-guided** — proposal → specs → design → tasks
- **Fast-forward** — `/opsx:ff` generates all planning docs at once
- **Archive workflow** — Completed changes archived with timestamp

### Commands

```
/opsx:onboard  # Initial setup
/opsx:new <id> # Start new change → creates openspec/changes/<id>/
/opsx:ff       # Fast-forward: generate proposal → specs → design → tasks
/opsx:apply    # Implement all tasks
/opsx:archive  # Archive completed change
```

### Directory Structure

```
openspec/
├── changes/
│   ├── add-dark-mode/
│   │   ├── proposal.md       # Why we're doing this
│   │   ├── specs/            # Requirements
│   │   ├── design.md         # Technical approach
│   │   └── tasks.md          # Implementation checklist
│   └── archive/
│       └── 2025-01-23-add-dark-mode/
└── dashboard/                # Visual task management
```

### Valuable Concepts for Our Architecture

| Concept | Value | Integration Notes |
|---------|-------|-------------------|
| **Proposal First** | Why before what before how | Add `PROPOSAL.md` to feature template |
| **Fast-Forward** | Generate all docs in one step | Add `eas feature init --full` |
| **Archive Workflow** | Completed features archived with date | Add `eas feature archive <id>` |
| **Fluid Iteration** | Update any artifact anytime | Our markdown approach already allows this |
| **Dashboard** | Visual task management | Consider web UI for `flo status` |

### Gaps OpenSpec Has (We Can Fill)

- ❌ No parallel agent execution
- ❌ No dependency DAG (checklist only)
- ❌ No cross-repo coordination
- ❌ No worktree isolation
- ❌ Node.js (we're Go)

---

## 4. Comparison Matrix

| Feature | Conductor | Spec-Kit | OpenSpec | Our EAS |
|---------|-----------|----------|----------|---------|
| **Language** | TypeScript | Python | Node.js | Go |
| **Agent Support** | Gemini only | 20+ agents | 20+ agents | Copilot SDK |
| **Parallel Agents** | ❌ | ❌ | ❌ | ✅ (1-30) |
| **Dependency Graph** | ❌ | ❌ | ❌ | ✅ (DAG) |
| **Cross-Repo** | ❌ | ❌ | ❌ | ✅ |
| **Worktree Isolation** | ❌ | ❌ | ❌ | ✅ |
| **TDD Enforcement** | Workflow doc | ❌ | ❌ | ✅ (hard) |
| **Context Persistence** | ✅ | ✅ (constitution) | ✅ (proposal) | ✅ |
| **Git-Aware Revert** | ✅ | ❌ | ❌ | Planned |
| **Archive Workflow** | ❌ | ❌ | ✅ | Planned |
| **Markdown-Native** | ✅ | ✅ | ✅ | ✅ |

---

## 5. Recommended Integrations

### 5.1 Adopt from Conductor

```yaml
# Add to feature structure
feature/user-auth/
├── context/
│   ├── product.md        # From Conductor
│   ├── guidelines.md     # From Conductor
│   └── tech-stack.md     # From Conductor
├── SPEC.md
├── tasks/
└── ...
```

**Implementation:**
```go
// WorktreeManager.Revert() - git-aware revert
func (w *WorktreeManager) Revert(target RevertTarget) error {
    // target can be: Feature, Phase, or Task
    // Analyze git history to find commits for that logical unit
    // Revert in correct order
}
```

### 5.2 Adopt from Spec-Kit

```yaml
# Add constitution to workspace root
workspace/
├── CONSTITUTION.md       # Governing principles
├── templates/
│   ├── spec.md
│   ├── task.md
│   └── plan.md
└── features/
    └── ...
```

**New Command:**
```bash
`flo clarify <feature-id>  # Interactive Q&A to find gaps
`flo analyze <feature-id>  # Cross-artifact consistency check
```

### 5.3 Adopt from OpenSpec

```yaml
# Add proposal + archive workflow
feature/user-auth/
├── PROPOSAL.md           # Why before what
├── SPEC.md
├── DESIGN.md             # Technical approach
├── tasks/
└── ...

# Archive structure
archive/
└── 2026-02-05-user-auth/
    └── ...
```

**New Commands:**
```bash
`flo feature init <name> --full  # Fast-forward: create all docs
`flo feature archive <id>        # Archive completed feature
```

---

## 6. Proposed Updated Directory Structure

Incorporating best ideas from all three tools:

```
workspace/
├── CONSTITUTION.md               # From spec-kit: governing principles
├── templates/                    # From spec-kit: consistent formats
│   ├── spec-template.md
│   ├── task-template.md
│   └── proposal-template.md
├── context/                      # From Conductor: persistent context
│   ├── product.md
│   ├── guidelines.md
│   ├── tech-stack.md
│   └── workflow.md
├── features/
│   └── user-auth/
│       ├── PROPOSAL.md           # From OpenSpec: why we're doing this
│       ├── SPEC.md               # Human-readable requirements
│       ├── DESIGN.md             # Technical approach (from OpenSpec)
│       ├── manifest.json         # Machine index
│       ├── repos.json            # Cross-repo links
│       ├── progress.md           # Learnings log
│       └── tasks/
│           ├── auth-001.md
│           └── auth-002.md
└── archive/                      # From OpenSpec: completed features
    └── 2026-02-05-user-auth/
```

---

## 7. Updated CLI Commands

```bash
# Workspace
`flo init                              # Initialize workspace + constitution
`flo context update                    # Update context docs interactively

# Features
`flo feature new <name>                # Create feature with PROPOSAL.md
`flo feature init <name> --full        # Fast-forward: all docs at once
`flo feature clarify <id>              # Q&A to find spec gaps
`flo feature analyze <id>              # Cross-artifact consistency check
`flo feature archive <id>              # Archive completed feature

# Tasks
`flo task create <title>               # Create task
`flo task list [--ready]               # List tasks
`flo task graph                        # Visualize DAG

# Execution
`flo run [--parallel N]                # Execute tasks
`flo run --verify                      # With phase verification
`flo revert <target>                   # Git-aware revert

# Monitoring
`flo status                            # Overall status
`flo dashboard                         # Web UI (future)
```

---

## 8. Next Steps

1. ✅ Research complete
2. ⏳ Add CONSTITUTION.md to workspace template
3. ⏳ Add context/ directory support
4. ⏳ Add PROPOSAL.md to feature template
5. ⏳ Implement `eas feature archive`
6. ⏳ Implement git-aware revert
7. ⏳ Add `--verify` flag for phase checkpoints

---

## References

- [Conductor](https://github.com/gemini-cli-extensions/conductor)
- [GitHub Spec-Kit](https://github.com/github/spec-kit)
- [OpenSpec](https://github.com/Fission-AI/OpenSpec)
