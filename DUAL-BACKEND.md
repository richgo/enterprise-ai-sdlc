# Dual Backend Design: Claude Code + Copilot SDK

EAS supports both Claude Code CLI and Copilot SDK as agent backends, allowing teams to choose based on their existing infrastructure.

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                          eas CLI                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  State Layer (backend-agnostic):                                │
│  ├── Task DAG management                                        │
│  ├── Cross-repo coordination                                    │
│  ├── Worktree lifecycle                                         │
│  └── Progress tracking                                          │
│                                                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Agent Abstraction Layer:                                       │
│  ├── AgentBackend interface                                     │
│  ├── Common tool definitions                                    │
│  └── TDD enforcement hooks                                      │
│                                                                  │
├──────────────────────┬──────────────────────────────────────────┤
│                      │                                          │
│   ClaudeBackend      │         CopilotBackend                   │
│   ─────────────      │         ──────────────                   │
│   • MCP server       │         • Copilot SDK (Go)               │
│   • claude CLI exec  │         • Native Go integration          │
│   • --mcp-config     │         • RegisterTool()                 │
│                      │         • Pre/PostToolUseHandler         │
│                      │                                          │
└──────────┬───────────┴────────────────────┬─────────────────────┘
           │                                │
     ┌─────▼─────┐                   ┌──────▼──────┐
     │  Claude   │                   │  Copilot    │
     │   Code    │                   │    CLI      │
     └───────────┘                   └─────────────┘
```

---

## Agent Backend Interface

```go
// pkg/agent/backend.go

type AgentBackend interface {
    // Initialize the backend
    Start(ctx context.Context, config BackendConfig) error
    Stop() error
    
    // Create a new agent session for a task
    CreateSession(ctx context.Context, task *Task, worktree string) (Session, error)
    
    // Register EAS tools with the backend
    RegisterTools(tools []Tool) error
    
    // Set TDD enforcement hooks
    SetPreCommitHook(hook PreCommitHook) error
}

type Session interface {
    // Execute the task
    Run(ctx context.Context, prompt string) (*Result, error)
    
    // Stream events (for progress monitoring)
    Events() <-chan Event
    
    // Cleanup
    Destroy(ctx context.Context) error
}

type BackendConfig struct {
    // Common settings
    Model       string
    MaxTokens   int
    Temperature float64
    
    // Backend-specific
    Claude  *ClaudeConfig
    Copilot *CopilotConfig
}

type ClaudeConfig struct {
    CLIPath    string   // Path to claude binary
    MCPConfig  string   // Path to MCP config file
    ExtraArgs  []string // Additional CLI args
}

type CopilotConfig struct {
    CLIPath     string          // Path to copilot binary
    Provider    *ProviderConfig // BYOK settings
    GithubToken string          // Or use logged-in user
}
```

---

## Claude Code Backend

Uses MCP server for tools + CLI execution.

```go
// pkg/agent/claude.go

type ClaudeBackend struct {
    config    *ClaudeConfig
    mcpServer *MCPServer
    tools     []Tool
}

func (b *ClaudeBackend) Start(ctx context.Context, cfg BackendConfig) error {
    b.config = cfg.Claude
    
    // Start MCP server
    b.mcpServer = NewMCPServer(b.tools)
    go b.mcpServer.Serve()
    
    // Generate MCP config file
    return b.writeMCPConfig()
}

func (b *ClaudeBackend) CreateSession(ctx context.Context, task *Task, worktree string) (Session, error) {
    return &ClaudeSession{
        backend:  b,
        task:     task,
        worktree: worktree,
    }, nil
}

type ClaudeSession struct {
    backend  *ClaudeBackend
    task     *Task
    worktree string
    cmd      *exec.Cmd
    events   chan Event
}

func (s *ClaudeSession) Run(ctx context.Context, prompt string) (*Result, error) {
    // Build command
    args := []string{
        "--print",                                    // Non-interactive
        "--mcp-config", s.backend.mcpServerConfig(), // EAS tools
        "--cwd", s.worktree,                         // Work in task worktree
        "--model", s.backend.config.Model,
        "--output-format", "stream-json",            // Stream events
    }
    args = append(args, s.backend.config.ExtraArgs...)
    args = append(args, prompt)
    
    s.cmd = exec.CommandContext(ctx, s.backend.config.CLIPath, args...)
    
    // Stream stdout for events
    stdout, _ := s.cmd.StdoutPipe()
    go s.streamEvents(stdout)
    
    err := s.cmd.Run()
    return s.collectResult(), err
}

func (s *ClaudeSession) streamEvents(r io.Reader) {
    scanner := bufio.NewScanner(r)
    for scanner.Scan() {
        var event Event
        json.Unmarshal(scanner.Bytes(), &event)
        s.events <- event
    }
    close(s.events)
}
```

### MCP Server (for Claude)

```go
// pkg/mcp/server.go

type MCPServer struct {
    tools    []Tool
    listener net.Listener
}

func (s *MCPServer) Serve() error {
    for {
        conn, _ := s.listener.Accept()
        go s.handleConnection(conn)
    }
}

func (s *MCPServer) handleConnection(conn net.Conn) {
    decoder := json.NewDecoder(conn)
    encoder := json.NewEncoder(conn)
    
    for {
        var req MCPRequest
        decoder.Decode(&req)
        
        switch req.Method {
        case "tools/list":
            encoder.Encode(s.listTools())
        case "tools/call":
            result := s.callTool(req.Params)
            encoder.Encode(result)
        }
    }
}

// Tool definitions shared with Copilot backend
func (s *MCPServer) listTools() []MCPTool {
    var mcpTools []MCPTool
    for _, t := range s.tools {
        mcpTools = append(mcpTools, MCPTool{
            Name:        t.Name,
            Description: t.Description,
            InputSchema: t.Schema,
        })
    }
    return mcpTools
}
```

---

## Copilot SDK Backend

Native Go integration using the SDK.

```go
// pkg/agent/copilot.go

import copilot "github.com/github/copilot-sdk/go"

type CopilotBackend struct {
    config *CopilotConfig
    client *copilot.Client
    tools  []Tool
}

func (b *CopilotBackend) Start(ctx context.Context, cfg BackendConfig) error {
    b.config = cfg.Copilot
    
    // Create SDK client
    opts := &copilot.ClientOptions{
        CLIPath: b.config.CLIPath,
    }
    if b.config.GithubToken != "" {
        opts.GithubToken = b.config.GithubToken
    }
    
    b.client = copilot.NewClient(opts)
    if err := b.client.Start(ctx); err != nil {
        return err
    }
    
    // Set TDD enforcement hook
    b.client.SetPreToolUseHandler(b.preToolUseHook)
    
    return nil
}

func (b *CopilotBackend) preToolUseHook(input copilot.PreToolUseHookInput, inv copilot.HookInvocation) (*copilot.PreToolUseHookOutput, error) {
    // Enforce TDD: block commits if tests fail
    if input.ToolName == "git_commit" || input.ToolName == "git" {
        args, _ := input.ToolArgs.(map[string]any)
        if isCommitCommand(args) {
            if !b.testsPass(inv.SessionID) {
                return &copilot.PreToolUseHookOutput{
                    PermissionDecision:       "deny",
                    PermissionDecisionReason: "Tests must pass before commit. Run eas_run_tests first.",
                }, nil
            }
        }
    }
    return nil, nil
}

func (b *CopilotBackend) CreateSession(ctx context.Context, task *Task, worktree string) (Session, error) {
    // Create Copilot session
    sessionCfg := &copilot.SessionConfig{
        Model:     b.config.Model,
        Streaming: true,
    }
    
    // BYOK if configured
    if b.config.Provider != nil {
        sessionCfg.Provider = &copilot.ProviderConfig{
            Type:    b.config.Provider.Type,
            BaseURL: b.config.Provider.BaseURL,
            APIKey:  b.config.Provider.APIKey,
        }
    }
    
    session, err := b.client.CreateSession(ctx, sessionCfg)
    if err != nil {
        return nil, err
    }
    
    // Register EAS tools
    for _, tool := range b.tools {
        session.RegisterTool(b.convertTool(tool))
    }
    
    return &CopilotSession{
        backend:  b,
        session:  session,
        task:     task,
        worktree: worktree,
        events:   make(chan Event, 100),
    }, nil
}

func (b *CopilotBackend) convertTool(t Tool) copilot.Tool {
    return copilot.Tool{
        Name:        t.Name,
        Description: t.Description,
        Parameters:  t.Schema,
        Handler: func(inv copilot.ToolInvocation) (copilot.ToolResult, error) {
            result, err := t.Handler(inv.Arguments)
            return copilot.ToolResult{
                TextResultForLLM: result,
                ResultType:       "success",
            }, err
        },
    }
}

type CopilotSession struct {
    backend  *CopilotBackend
    session  *copilot.Session
    task     *Task
    worktree string
    events   chan Event
}

func (s *CopilotSession) Run(ctx context.Context, prompt string) (*Result, error) {
    // Subscribe to events
    s.session.On(func(event copilot.SessionEvent) {
        s.events <- convertEvent(event)
    })
    
    // Send prompt and wait
    response, err := s.session.SendAndWait(ctx, copilot.MessageOptions{
        Prompt: prompt,
    })
    
    return &Result{
        Content: *response.Data.Content,
    }, err
}
```

---

## Shared Tool Definitions

Tools are defined once, work with both backends.

```go
// pkg/tools/eas_tools.go

type Tool struct {
    Name        string
    Description string
    Schema      map[string]any
    Handler     func(args map[string]any) (string, error)
}

var EASTools = []Tool{
    {
        Name:        "eas_task_list",
        Description: "List tasks in current feature. Returns task IDs, titles, status, and dependencies.",
        Schema: map[string]any{
            "type": "object",
            "properties": map[string]any{
                "status": map[string]any{
                    "type":        "string",
                    "description": "Filter by status: pending, in_progress, complete",
                },
                "repo": map[string]any{
                    "type":        "string",
                    "description": "Filter by repository name",
                },
            },
        },
        Handler: handleTaskList,
    },
    {
        Name:        "eas_task_get",
        Description: "Get detailed information about a specific task including spec, acceptance criteria, and dependencies.",
        Schema: map[string]any{
            "type": "object",
            "properties": map[string]any{
                "task_id": map[string]any{
                    "type":        "string",
                    "description": "Task ID (e.g., ua-001)",
                },
            },
            "required": []string{"task_id"},
        },
        Handler: handleTaskGet,
    },
    {
        Name:        "eas_task_complete",
        Description: "Mark current task as complete. Runs tests first - will fail if tests don't pass.",
        Schema: map[string]any{
            "type": "object",
            "properties": map[string]any{},
        },
        Handler: handleTaskComplete,
    },
    {
        Name:        "eas_run_tests",
        Description: "Run tests for the current task. Must pass before task can be completed.",
        Schema: map[string]any{
            "type": "object",
            "properties": map[string]any{
                "verbose": map[string]any{
                    "type":        "boolean",
                    "description": "Show detailed test output",
                },
            },
        },
        Handler: handleRunTests,
    },
    {
        Name:        "eas_spec_read",
        Description: "Read the feature specification (SPEC.md) for context.",
        Schema: map[string]any{
            "type": "object",
            "properties": map[string]any{},
        },
        Handler: handleSpecRead,
    },
    {
        Name:        "eas_deps_check",
        Description: "Check status of task dependencies. Returns which deps are complete/pending.",
        Schema: map[string]any{
            "type": "object",
            "properties": map[string]any{
                "task_id": map[string]any{
                    "type":        "string",
                    "description": "Task ID to check dependencies for",
                },
            },
            "required": []string{"task_id"},
        },
        Handler: handleDepsCheck,
    },
}

func handleTaskComplete(args map[string]any) (string, error) {
    // Get current task from context
    task := getCurrentTask()
    
    // Run tests first (TDD enforcement)
    result, err := runTests(task)
    if err != nil || !result.AllPassed {
        return "", fmt.Errorf("cannot complete task: tests failing\n%s", result.Output)
    }
    
    // Mark complete
    task.Status = "complete"
    saveTask(task)
    
    // Commit the work
    commitMessage := fmt.Sprintf("feat(%s): %s\n\nTask-ID: %s", task.Repo, task.Title, task.ID)
    if err := gitCommit(task.Worktree, commitMessage); err != nil {
        return "", err
    }
    
    return fmt.Sprintf("✓ Task %s completed and committed", task.ID), nil
}
```

---

## Configuration

```yaml
# .flo/config.yaml

feature: user-authentication
version: 1

# Choose backend: "claude" or "copilot"
backend: claude

# Backend-specific settings
claude:
  cli_path: claude           # or full path
  model: claude-sonnet-4-5-20250514
  extra_args:
    - --dangerously-skip-permissions  # For CI environments

copilot:
  cli_path: copilot
  model: gpt-4.1
  # BYOK (optional)
  provider:
    type: azure
    base_url: https://mycompany.openai.azure.com/openai/v1/
    api_key_env: AZURE_OPENAI_KEY  # Read from env var

# TDD settings (apply to both backends)
tdd:
  enforce: true
  test_command: "npm test"
  coverage_threshold: 80

# Cross-repo links
repos:
  android:
    url: git@github.com:myorg/android-app.git
    branch: feature/user-auth
  ios:
    url: git@github.com:myorg/ios-app.git
    branch: feature/user-auth
```

---

## CLI Usage

```bash
# Initialize with backend choice
`flo init my-feature --backend claude
`flo init my-feature --backend copilot

# Override backend for single command
`flo work ua-001 --backend copilot

# Check current config
`flo config show

# Switch backends
`flo config set backend copilot
```

---

## Parallel Execution

Both backends support parallel agents:

```go
// pkg/orchestrator/parallel.go

func (o *Orchestrator) RunParallel(ctx context.Context, maxAgents int) error {
    sem := make(chan struct{}, maxAgents)
    var wg sync.WaitGroup
    
    for {
        // Get tasks ready to work (deps complete, not started)
        readyTasks := o.taskRegistry.GetReady()
        if len(readyTasks) == 0 {
            break
        }
        
        for _, task := range readyTasks {
            sem <- struct{}{}  // Acquire
            wg.Add(1)
            
            go func(t *Task) {
                defer wg.Done()
                defer func() { <-sem }()  // Release
                
                // Create worktree for isolation
                worktree := o.createWorktree(t)
                
                // Create session using configured backend
                session, _ := o.backend.CreateSession(ctx, t, worktree)
                defer session.Destroy(ctx)
                
                // Build prompt with context
                prompt := o.buildPrompt(t)
                
                // Execute
                result, err := session.Run(ctx, prompt)
                o.handleResult(t, result, err)
            }(task)
        }
        
        wg.Wait()  // Wait for batch before checking next ready tasks
    }
    
    return nil
}
```

---

## Summary

| Feature | Claude Backend | Copilot Backend |
|---------|---------------|-----------------|
| Integration | MCP server + CLI exec | Native Go SDK |
| Tool Registration | MCP protocol | `RegisterTool()` |
| TDD Enforcement | MCP tool logic | `PreToolUseHandler` |
| Streaming | `--output-format stream-json` | `Session.On()` events |
| BYOK | N/A (Anthropic billing) | OpenAI/Azure/Anthropic |
| Parallel | Multiple CLI processes | Multiple sessions |

Both backends:
- ✅ Share same tool definitions
- ✅ Same TDD enforcement rules
- ✅ Same task/worktree management
- ✅ Same prompt construction
- ✅ Swappable via config
