# Flo AI SDLC - Architecture Proposal

## Overview

A Go CLI/SDK for AI-assisted software development using the GitHub Copilot SDK. Supports 1-30 parallel agents working across multiple repositories with enforced TDD and git worktree isolation.

**Core Principles:**
- Git-native (worktrees, branches, no external DBs)
- TDD-first (tests define done)
- Context isolation (one worktree per task)
- Cross-repo coordination (features span multiple repos)
- Professional naming (no whimsy)

---

## Component Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                              CLI                                     │
│                         (eas command)                                │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐        │
│  │   Scheduler    │  │  TaskRegistry  │  │  RepoRegistry  │        │
│  │                │  │                │  │                │        │
│  │ • Queue mgmt   │  │ • Task CRUD    │  │ • Multi-repo   │        │
│  │ • Parallelism  │  │ • Dependencies │  │ • Cross-links  │        │
│  │ • Priority     │  │ • State (DAG)  │  │ • Sync         │        │
│  └───────┬────────┘  └───────┬────────┘  └───────┬────────┘        │
│          │                   │                   │                  │
│          └───────────────────┼───────────────────┘                  │
│                              │                                      │
│                    ┌─────────┴─────────┐                           │
│                    │   Orchestrator    │                           │
│                    │                   │                           │
│                    │ • Agent lifecycle │                           │
│                    │ • Work dispatch   │                           │
│                    │ • Result collect  │                           │
│                    └─────────┬─────────┘                           │
│                              │                                      │
│            ┌─────────────────┼─────────────────┐                   │
│            │                 │                 │                    │
│     ┌──────┴──────┐   ┌──────┴──────┐   ┌──────┴──────┐           │
│     │   Agent 1   │   │   Agent 2   │   │   Agent N   │           │
│     │             │   │             │   │             │           │
│     │ • Worktree  │   │ • Worktree  │   │ • Worktree  │           │
│     │ • Copilot   │   │ • Copilot   │   │ • Copilot   │           │
│     │ • TDD Loop  │   │ • TDD Loop  │   │ • TDD Loop  │           │
│     └──────┬──────┘   └──────┬──────┘   └──────┬──────┘           │
│            │                 │                 │                    │
│            └─────────────────┼─────────────────┘                   │
│                              │                                      │
│                    ┌─────────┴─────────┐                           │
│                    │  WorktreeManager  │                           │
│                    │                   │                           │
│                    │ • Create/cleanup  │                           │
│                    │ • Isolation       │                           │
│                    │ • Branch strategy │                           │
│                    └─────────┬─────────┘                           │
│                              │                                      │
│                    ┌─────────┴─────────┐                           │
│                    │       Git         │                           │
│                    │   (filesystem)    │                           │
│                    └───────────────────┘                           │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Core Components

### 1. CLI (`cmd/eas/`)

The command-line interface. Single binary, subcommand pattern.

```bash
`flo init                          # Initialize workspace
`flo feature create <name>         # Create feature (cross-repo)
`flo task create <title>           # Create task
`flo task list [--ready]           # List tasks (--ready = unblocked)
`flo run [--parallel N]            # Execute tasks (1-30 agents)
`flo status                        # Show progress
`flo graph                         # Visualize task DAG
```

### 2. Orchestrator (`pkg/orchestrator/`)

Controls agent lifecycle and work distribution.

```go
type Orchestrator struct {
    scheduler    *Scheduler
    taskRegistry *TaskRegistry
    repoRegistry *RepoRegistry
    worktreeMgr  *WorktreeManager
    agents       []*Agent
    maxParallel  int
}

func (o *Orchestrator) Run(ctx context.Context) error
func (o *Orchestrator) SpawnAgent(task *Task) (*Agent, error)
func (o *Orchestrator) CollectResult(agent *Agent) (*TaskResult, error)
```

**Responsibilities:**
- Spawn/terminate agents based on workload
- Dispatch tasks to available agents
- Collect results and update task state
- Handle failures and retries

### 3. Scheduler (`pkg/scheduler/`)

Manages task queue with priority and dependency awareness.

```go
type Scheduler struct {
    queue      *PriorityQueue
    inProgress map[string]*Task
    completed  map[string]*TaskResult
}

func (s *Scheduler) Next() (*Task, error)          // Get next ready task
func (s *Scheduler) Ready() []*Task                // All unblocked tasks
func (s *Scheduler) Complete(id string, result *TaskResult)
func (s *Scheduler) Block(id string, reason string)
```

**Scheduling Rules:**
1. Only tasks with `status=pending` and all deps `status=done`
2. Higher priority (P0 > P1 > P2) first
3. Within same priority, FIFO
4. Respect `maxParallel` limit

### 4. TaskRegistry (`pkg/task/`)

Task storage with **markdown files as source of truth** and manifest index for fast queries.

```go
type TaskRegistry struct {
    featureDir  string              // Path to feature directory
    tasks       map[string]*Task    // In-memory cache
    manifest    *Manifest           // Index for fast lookups
    graph       *DAG                // Dependency graph
}

// Task represents a single work item (parsed from markdown)
type Task struct {
    // Frontmatter (YAML header in markdown)
    ID          string       `yaml:"id"`
    Title       string       `yaml:"title"`
    Status      TaskStatus   `yaml:"status"`
    Priority    int          `yaml:"priority"`
    Repo        string       `yaml:"repo"`
    Path        string       `yaml:"path"`
    TestFile    string       `yaml:"test_file"`
    SpecRef     string       `yaml:"spec_ref"`
    Deps        []string     `yaml:"dependencies"`
    Assignee    string       `yaml:"assignee,omitempty"`
    CreatedAt   time.Time    `yaml:"created"`
    UpdatedAt   time.Time    `yaml:"updated"`
    
    // Body (parsed from markdown sections)
    Description       string      // ## Description
    AcceptanceCriteria []Criterion // ## Acceptance Criteria
    TestScenarios     string      // ## Test Scenarios
    DefinitionOfDone  []Criterion // ## Definition of Done
    Plan              string      // ## Implementation Plan
    Notes             string      // ## Notes
    Result            *TaskResult // ## Result
    
    // Internal
    FilePath    string       // Path to .md file
}

type Criterion struct {
    Text    string
    Checked bool
}

type TaskStatus string
const (
    StatusPending  TaskStatus = "pending"
    StatusRunning  TaskStatus = "running"
    StatusDone     TaskStatus = "done"
    StatusFailed   TaskStatus = "failed"
    StatusBlocked  TaskStatus = "blocked"
)

type TaskResult struct {
    Success     bool          `yaml:"success"`
    Commit      string        `yaml:"commit"`
    Branch      string        `yaml:"branch"`
    TestsPassed int           `yaml:"tests_passed"`
    Duration    time.Duration `yaml:"duration"`
    Error       string        `yaml:"error,omitempty"`
    Learnings   string        `yaml:"learnings"`
}

// Registry operations
func (r *TaskRegistry) Load(featureDir string) error           // Parse all task/*.md files
func (r *TaskRegistry) Get(id string) (*Task, error)           // Get single task
func (r *TaskRegistry) List(filter TaskFilter) []*Task         // List with filters
func (r *TaskRegistry) Ready() []*Task                         // Tasks with deps satisfied
func (r *TaskRegistry) Create(task *Task) error                // Create new task file
func (r *TaskRegistry) Update(task *Task) error                // Update task file
func (r *TaskRegistry) SyncManifest() error                    // Regenerate manifest.json

// Manifest for fast queries (auto-generated)
type Manifest struct {
    Version     string                 `json:"version"`
    FeatureID   string                 `json:"feature_id"`
    GeneratedAt time.Time              `json:"generated_at"`
    Stats       ManifestStats          `json:"stats"`
    Tasks       []ManifestTask         `json:"tasks"`
    Graph       ManifestGraph          `json:"graph"`
}

type ManifestStats struct {
    Total   int `json:"total"`
    Pending int `json:"pending"`
    Running int `json:"running"`
    Done    int `json:"done"`
    Failed  int `json:"failed"`
}

type ManifestTask struct {
    ID       string   `json:"id"`
    Title    string   `json:"title"`
    Status   string   `json:"status"`
    Priority int      `json:"priority"`
    Repo     string   `json:"repo"`
    Deps     []string `json:"deps"`
    File     string   `json:"file"`
}

type ManifestGraph struct {
    Roots []string            `json:"roots"`
    Edges map[string][]string `json:"edges"`
}
```

### Markdown Parsing

Tasks are stored as markdown with YAML frontmatter:

```go
func ParseTaskFile(path string) (*Task, error) {
    content, _ := os.ReadFile(path)
    
    // Split frontmatter and body
    parts := bytes.SplitN(content, []byte("---"), 3)
    
    // Parse YAML frontmatter
    var task Task
    yaml.Unmarshal(parts[1], &task)
    
    // Parse markdown body into sections
    task.Description = extractSection(parts[2], "## Description")
    task.AcceptanceCriteria = extractChecklist(parts[2], "## Acceptance Criteria")
    task.TestScenarios = extractSection(parts[2], "## Test Scenarios")
    task.DefinitionOfDone = extractChecklist(parts[2], "## Definition of Done")
    task.Plan = extractSection(parts[2], "## Implementation Plan")
    task.Notes = extractSection(parts[2], "## Notes")
    
    task.FilePath = path
    return &task, nil
}

func (t *Task) WriteFile() error {
    // Render back to markdown with frontmatter
    var buf bytes.Buffer
    buf.WriteString("---\n")
    yaml.NewEncoder(&buf).Encode(t.frontmatter())
    buf.WriteString("---\n\n")
    buf.WriteString(t.renderBody())
    return os.WriteFile(t.FilePath, buf.Bytes(), 0644)
}
```

### 5. RepoRegistry (`pkg/repo/`)

Multi-repository coordination.

```go
type Repository struct {
    Name      string `json:"name"`       // e.g., "mobile-android"
    URL       string `json:"url"`        // Git remote URL
    LocalPath string `json:"local_path"` // Cloned location
    Type      string `json:"type"`       // android|ios|web|service|shared
}

type RepoRegistry struct {
    repos    map[string]*Repository
    features map[string]*Feature
}

type Feature struct {
    ID          string              `json:"id"`
    Name        string              `json:"name"`
    Description string              `json:"description"`
    Spec        string              `json:"spec"`      // Path to SPEC.md
    Repos       map[string]RepoRef  `json:"repos"`     // repo_name -> branch/path
    Tasks       []string            `json:"tasks"`     // Task IDs
    Status      FeatureStatus       `json:"status"`
}

type RepoRef struct {
    Branch string `json:"branch"`
    Path   string `json:"path"`
}
```

### 6. WorktreeManager (`pkg/worktree/`)

Git worktree lifecycle management. **One worktree per task** for context isolation.

```go
type WorktreeManager struct {
    baseDir string  // e.g., ~/.flo/worktrees/
}

type Worktree struct {
    ID       string // Matches task ID
    RepoName string
    Branch   string
    Path     string // Filesystem path
}

func (w *WorktreeManager) Create(task *Task) (*Worktree, error)
func (w *WorktreeManager) Cleanup(id string) error
func (w *WorktreeManager) List() ([]*Worktree, error)
```

**Worktree Strategy:**
```bash
# For task "auth-001" in repo "mobile-android":
git worktree add ~/.flo/worktrees/auth-001 -b task/auth-001

# Agent works in isolated directory with clean context
# On completion: commit, push branch, remove worktree
```

### 7. Agent (`pkg/agent/`)

The AI worker. Wraps GitHub Copilot SDK. Executes single task in TDD loop.

```go
type Agent struct {
    ID          string
    Task        *Task
    Worktree    *Worktree
    CopilotClient *copilot.Client
    Status      AgentStatus
}

type AgentStatus string
const (
    AgentIdle     AgentStatus = "idle"
    AgentRunning  AgentStatus = "running"
    AgentComplete AgentStatus = "complete"
    AgentFailed   AgentStatus = "failed"
)

func (a *Agent) Execute(ctx context.Context) (*TaskResult, error)
```

**Agent Execution Loop (TDD-enforced):**
```
1. Read task spec + test file
2. Run tests → expect FAIL (red)
3. Generate implementation via Copilot
4. Run tests → check PASS (green)
5. If fail, iterate (max N attempts)
6. Run linter/typecheck
7. If all green: commit + report success
8. If stuck: report failure + learnings
```

---

## Data Model & Storage

### Design Philosophy

**Hybrid Markdown + Manifest approach** (inspired by Backlog.md):
- **Markdown files** — human-readable, editable in any editor, git-diff friendly
- **Manifest index** — machine-readable for fast queries and DAG operations
- **Single source of truth** — markdown files; manifest is regenerated/synced

### Directory Structure

```
~/.flo/                              # Global config
├── config.yaml                      # CLI config
└── worktrees/                       # Active worktrees (one per task)
    ├── auth-001/                    
    └── auth-002/

./eas/                               # Project workspace
├── config.yaml                      # Workspace config
├── repos.yaml                       # Repository registry
└── features/
    └── user-auth/                   # Feature directory
        ├── SPEC.md                  # Human-readable specification
        ├── manifest.json            # Machine index (auto-generated)
        ├── repos.json               # Cross-repo links
        ├── progress.md              # Learnings log (append-only)
        └── tasks/                   # Task files (source of truth)
            ├── auth-001.md
            ├── auth-002.md
            ├── auth-003.md
            └── ...
```

### File Formats

#### config.yaml (workspace)
```yaml
version: "1"
workspace:
  name: "myproject"
  parallel: 10                       # Default parallelism
  
copilot:
  model: "gpt-4.1"
  timeout: 300s
  
tdd:
  required: true
  test_commands:                     # Per-repo-type defaults
    go: "go test ./..."
    node: "npm test"
    python: "pytest"
  lint_commands:
    go: "golangci-lint run"
    node: "npm run lint"
    python: "ruff check ."
  
git:
  auto_push: true
  branch_prefix: "task/"
  worktree_dir: "~/.flo/worktrees"
```

#### Task Markdown Files (Source of Truth)

**tasks/auth-001.md**
```markdown
---
id: auth-001
title: Implement OAuth2 token exchange
status: pending
priority: 0
repo: api-gateway
path: internal/auth
test_file: internal/auth/oauth_test.go
spec_ref: "SPEC.md#oauth2-flow"
dependencies: []
assignee: null
created: 2026-02-04T10:00:00Z
updated: 2026-02-04T10:00:00Z
---

# Implement OAuth2 token exchange

## Description

Implement the OAuth2 authorization code flow for Google authentication.
This is the foundational auth component that other tasks depend on.

## Acceptance Criteria

- [ ] POST /auth/google/callback exchanges code for tokens
- [ ] Access token and refresh token are returned
- [ ] Invalid codes return 400 with error message
- [ ] Token expiry is correctly set

## Test Scenarios (TDD)

```go
// internal/auth/oauth_test.go
func TestGoogleCallback_ValidCode_ReturnsTokens(t *testing.T)
func TestGoogleCallback_InvalidCode_Returns400(t *testing.T)
func TestGoogleCallback_ExpiredCode_Returns400(t *testing.T)
```

## Definition of Done

- [ ] All tests pass
- [ ] Code reviewed
- [ ] No lint errors
- [ ] Documentation updated

## Implementation Plan

1. Create handler for POST /auth/google/callback
2. Implement token exchange with Google OAuth API
3. Store refresh token securely
4. Return access token with expiry

## Notes

_Agent notes and learnings go here_

## Result

_Filled on completion_
- Commit: 
- Branch: 
- Duration: 
```

**tasks/auth-002.md**
```markdown
---
id: auth-002
title: Add token validation middleware
status: pending
priority: 0
repo: api-gateway
path: internal/middleware
test_file: internal/middleware/auth_test.go
spec_ref: "SPEC.md#middleware"
dependencies:
  - auth-001
assignee: null
created: 2026-02-04T10:00:00Z
updated: 2026-02-04T10:00:00Z
---

# Add token validation middleware

## Description

Create middleware that validates JWT access tokens on protected routes.

## Acceptance Criteria

- [ ] Valid tokens allow request through
- [ ] Expired tokens return 401
- [ ] Invalid signatures return 401
- [ ] User context is set from token claims

## Test Scenarios (TDD)

```go
// internal/middleware/auth_test.go
func TestAuthMiddleware_ValidToken_PassesThrough(t *testing.T)
func TestAuthMiddleware_ExpiredToken_Returns401(t *testing.T)
func TestAuthMiddleware_InvalidSignature_Returns401(t *testing.T)
func TestAuthMiddleware_SetsUserContext(t *testing.T)
```

## Definition of Done

- [ ] All tests pass
- [ ] Code reviewed
- [ ] No lint errors

## Implementation Plan

1. Parse JWT from Authorization header
2. Validate signature and expiry
3. Extract claims and set user context
4. Return 401 on any validation failure

## Notes

Depends on auth-001 for token format specification.

## Result

_Filled on completion_
```

#### manifest.json (Auto-generated Index)

Generated by `eas manifest sync` or automatically on task operations.
Provides fast lookups without parsing all markdown files.

```json
{
  "version": "1",
  "feature_id": "user-auth",
  "generated_at": "2026-02-04T12:00:00Z",
  "stats": {
    "total": 8,
    "pending": 5,
    "running": 1,
    "done": 2,
    "failed": 0
  },
  "tasks": [
    {
      "id": "auth-001",
      "title": "Implement OAuth2 token exchange",
      "status": "done",
      "priority": 0,
      "repo": "api-gateway",
      "deps": [],
      "file": "tasks/auth-001.md"
    },
    {
      "id": "auth-002",
      "title": "Add token validation middleware",
      "status": "running",
      "priority": 0,
      "repo": "api-gateway",
      "deps": ["auth-001"],
      "file": "tasks/auth-002.md"
    },
    {
      "id": "auth-003",
      "title": "Implement Google Sign-In button",
      "status": "pending",
      "priority": 1,
      "repo": "web-app",
      "deps": ["auth-001"],
      "file": "tasks/auth-003.md"
    }
  ],
  "graph": {
    "roots": ["auth-001"],
    "edges": {
      "auth-001": ["auth-002", "auth-003", "auth-005", "auth-007"],
      "auth-002": [],
      "auth-003": ["auth-004"],
      "auth-005": ["auth-006"]
    }
  }
}
```

#### repos.json (Cross-repo Links)

```json
{
  "version": "1",
  "feature_id": "user-auth",
  "repos": {
    "api-gateway": {
      "url": "git@github.com:acme/api-gateway.git",
      "local": "~/code/api-gateway",
      "branch": "feature/user-auth",
      "path": "internal/auth",
      "type": "service",
      "tasks": ["auth-001", "auth-002"]
    },
    "web-app": {
      "url": "git@github.com:acme/web-app.git",
      "local": "~/code/web-app",
      "branch": "feature/user-auth",
      "path": "src/auth",
      "type": "web",
      "tasks": ["auth-003", "auth-004"]
    },
    "mobile-ios": {
      "url": "git@github.com:acme/mobile-ios.git",
      "local": "~/code/mobile-ios",
      "branch": "feature/user-auth",
      "path": "Sources/Auth",
      "type": "ios",
      "tasks": ["auth-005", "auth-006"]
    },
    "mobile-android": {
      "url": "git@github.com:acme/mobile-android.git",
      "local": "~/code/mobile-android",
      "branch": "feature/user-auth",
      "path": "app/src/main/java/com/acme/auth",
      "type": "android",
      "tasks": ["auth-007", "auth-008"]
    }
  }
}
```

#### progress.md (Append-only Learnings)

```markdown
# Progress Log: user-auth

## 2026-02-04 10:30 — auth-001 completed

**Agent:** agent-1
**Duration:** 4m 32s
**Commit:** abc1234

### Learnings
- Google OAuth requires `access_type=offline` for refresh tokens
- Token endpoint needs `Content-Type: application/x-www-form-urlencoded`

### Files Changed
- internal/auth/google.go (new)
- internal/auth/oauth_test.go (new)
- go.mod (updated)

---

## 2026-02-04 10:45 — auth-002 started

**Agent:** agent-2
**Worktree:** ~/.flo/worktrees/auth-002

---
```

---

## Execution Flow

### Single Task (`eas run --task auth-001`)

```
┌─────────────┐
│   CLI       │
└──────┬──────┘
       │ 1. Load task
       ▼
┌─────────────┐
│ Orchestrator│
└──────┬──────┘
       │ 2. Create worktree
       ▼
┌─────────────┐     ┌─────────────┐
│ Worktree    │────▶│ git worktree│
│ Manager     │     │ add ...     │
└──────┬──────┘     └─────────────┘
       │ 3. Spawn agent
       ▼
┌─────────────┐
│   Agent     │
│             │
│ ┌─────────┐ │
│ │ Read    │ │ 4. Load spec + test
│ │ Context │ │
│ └────┬────┘ │
│      ▼      │
│ ┌─────────┐ │
│ │ TDD     │ │ 5. Red → Green loop
│ │ Loop    │ │
│ └────┬────┘ │
│      ▼      │
│ ┌─────────┐ │
│ │ Commit  │ │ 6. Commit + push
│ └─────────┘ │
└──────┬──────┘
       │ 7. Report result
       ▼
┌─────────────┐
│ Task        │
│ Registry    │ 8. Update status → done
└─────────────┘
```

### Parallel Execution (`eas run --parallel 10`)

```
┌─────────────┐
│ Scheduler   │
│             │
│ Ready Queue:│
│ [001, 003,  │◀─── Tasks with deps satisfied
│  005, 007]  │
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────────┐
│           Orchestrator              │
│                                     │
│  Dispatch up to 10 parallel agents  │
│                                     │
│  ┌───┐ ┌───┐ ┌───┐ ┌───┐ ┌───┐    │
│  │A1 │ │A2 │ │A3 │ │A4 │ │...│    │
│  └─┬─┘ └─┬─┘ └─┬─┘ └─┬─┘ └───┘    │
│    │     │     │     │             │
│    ▼     ▼     ▼     ▼             │
│  ┌───┐ ┌───┐ ┌───┐ ┌───┐          │
│  │WT1│ │WT2│ │WT3│ │WT4│ Worktrees│
│  └───┘ └───┘ └───┘ └───┘          │
└─────────────────────────────────────┘
       │
       │ On completion:
       │ • Update task status
       │ • Cleanup worktree
       │ • Dispatch next ready task
       ▼
```

---

## TDD Enforcement

Each task MUST have a `test_file` that defines acceptance.

**Agent TDD Loop:**
```go
func (a *Agent) ExecuteTDD(ctx context.Context) (*TaskResult, error) {
    // 1. Run test first - MUST FAIL (proves test is meaningful)
    result := a.runTests()
    if result.Passed {
        return nil, errors.New("test already passes - nothing to implement")
    }
    
    // 2. Generate implementation via Copilot
    for attempt := 0; attempt < maxAttempts; attempt++ {
        code := a.generateWithCopilot(ctx)
        a.writeCode(code)
        
        // 3. Run tests
        result := a.runTests()
        if result.Passed {
            // 4. Run lint/typecheck
            if a.runLint() && a.runTypecheck() {
                // 5. Commit
                return a.commit()
            }
        }
        
        // 6. Feed error back to Copilot for next attempt
        a.feedbackError(result.Error)
    }
    
    return nil, errors.New("max attempts exceeded")
}
```

---

## Cross-Repo Workflow

### Feature spanning 4 repos

```
Feature: user-auth
├── api-gateway      (tasks: auth-001, auth-002)
├── web-app          (tasks: auth-003, auth-004)  
├── mobile-ios       (tasks: auth-005, auth-006)
└── mobile-android   (tasks: auth-007, auth-008)

Dependency graph:
                    ┌─────────┐
                    │auth-001 │  API: token exchange
                    │(P0)     │
                    └────┬────┘
           ┌─────────────┼─────────────┐
           ▼             ▼             ▼
      ┌─────────┐   ┌─────────┐   ┌─────────┐
      │auth-002 │   │auth-003 │   │auth-005 │
      │API:     │   │Web:     │   │iOS:     │
      │middleware   │button   │   │signin   │
      └─────────┘   └────┬────┘   └────┬────┘
                         ▼             ▼
                    ┌─────────┐   ┌─────────┐
                    │auth-004 │   │auth-006 │
                    │Web:     │   │iOS:     │
                    │session  │   │token    │
                    └─────────┘   └─────────┘
```

**Execution order (respecting deps):**
1. auth-001 (no deps) → runs first
2. auth-002, auth-003, auth-005, auth-007 → parallel (all depend only on 001)
3. auth-004, auth-006, auth-008 → parallel (depend on completed tasks)

---

## CLI Commands

```bash
# Workspace management
`flo init                              # Initialize workspace
`flo config set parallel 20            # Configure

# Repository management  
`flo repo add <name> <url> [--type service|web|ios|android]
`flo repo list
`flo repo sync                         # Fetch all repos

# Feature management
`flo feature create <name>             # Create feature
`flo feature list
`flo feature status <id>               # Show feature progress

# Task management
`flo task create <title> --repo <repo> --test <file> [--deps id1,id2]
`flo task list [--ready] [--feature <id>]
`flo task show <id>
`flo task graph [--feature <id>]       # ASCII DAG visualization

# Execution
`flo run                               # Run all ready tasks (default parallel)
`flo run --task <id>                   # Run single task
`flo run --parallel 30                 # Run with 30 agents
`flo run --feature <id>                # Run all tasks for feature
`flo run --dry-run                     # Show what would run

# Monitoring
`flo status                            # Overall status
`flo agents                            # List active agents
`flo logs <task-id>                    # View agent logs
```

---

## Go Project Structure

```
github.com/acme/eas/
├── cmd/
│   └── eas/
│       └── main.go                   # CLI entrypoint
├── pkg/
│   ├── orchestrator/
│   │   └── orchestrator.go
│   ├── scheduler/
│   │   ├── scheduler.go
│   │   └── priority_queue.go
│   ├── task/
│   │   ├── task.go
│   │   ├── registry.go
│   │   └── graph.go                  # DAG operations
│   ├── repo/
│   │   ├── repo.go
│   │   ├── registry.go
│   │   └── feature.go
│   ├── worktree/
│   │   └── manager.go
│   ├── agent/
│   │   ├── agent.go
│   │   ├── tdd.go                    # TDD loop
│   │   └── copilot.go                # Copilot SDK wrapper
│   ├── config/
│   │   └── config.go
│   └── git/
│       └── git.go                    # Git operations
├── internal/
│   └── cli/
│       ├── root.go
│       ├── init.go
│       ├── task.go
│       ├── run.go
│       └── ...
├── go.mod
├── go.sum
└── README.md
```

---

## Next Steps

1. **Validate architecture** - Does this meet your requirements?
2. **Copilot SDK research** - Confirm Go SDK availability/capabilities
3. **Prototype core** - Start with TaskRegistry + WorktreeManager
4. **TDD loop** - Implement agent execution with test enforcement
5. **Parallel execution** - Add Orchestrator + Scheduler
6. **Cross-repo** - Add RepoRegistry + Feature coordination

---

## Open Questions

1. **Copilot SDK**: Is there an official Go SDK, or do we wrap the CLI/API?
2. **Test framework detection**: Auto-detect (go test, npm test, pytest) or configure?
3. **PR creation**: Auto-create PRs when tasks complete, or manual?
4. **Failure handling**: Retry policy? Human escalation?
5. **Spec format**: Pure markdown, or structured (OpenAPI, Gherkin)?
