# GitHub Copilot SDK - Deep Dive

## Overview

**Status:** Technical Preview (not production-ready)  
**Go Package:** `github.com/github/copilot-sdk/go`  
**Architecture:** SDK Client → JSON-RPC → Copilot CLI (server mode)

The Copilot SDK exposes the same engine behind Copilot CLI as a programmable Go interface.

---

## Installation

```bash
# Prerequisites: Copilot CLI installed and authenticated
copilot --version

# Install Go SDK
go get github.com/github/copilot-sdk/go
```

---

## Core Types (from source)

### Client Configuration

```go
type ClientOptions struct {
    // CLIPath is the path to the Copilot CLI executable (default: "copilot")
    CLIPath string
    
    // Cwd is the working directory for the CLI process
    Cwd string
    
    // Port for TCP transport (default: 0 = random port)
    Port int
    
    // UseStdio controls transport: true = stdio, false = TCP
    UseStdio *bool
    
    // CLIUrl connects to existing CLI server (format: "host:port")
    // Mutually exclusive with CLIPath
    CLIUrl string
    
    // GithubToken for authentication (priority over other methods)
    GithubToken string
    
    // UseLoggedInUser uses stored OAuth tokens from CLI login
    UseLoggedInUser *bool
    
    // AutoStart/AutoRestart control CLI lifecycle
    AutoStart   *bool
    AutoRestart *bool
    
    // Env is environment variables for CLI process
    Env []string
}
```

### Session Configuration

```go
type SessionConfig struct {
    Model     string          // e.g., "gpt-4.1"
    Streaming bool            // Enable streaming responses
    Provider  *ProviderConfig // BYOK configuration
    
    // System message customization
    SystemMessage *SystemMessageConfig
}

type ProviderConfig struct {
    Type    string // "openai" | "azure" | "anthropic"
    BaseURL string // API endpoint
    APIKey  string // API key
    WireApi string // "completions" | "responses"
}
```

### Session Object

```go
type Session struct {
    SessionID string
    
    // WorkspacePath returns path for infinite sessions
    // Contains checkpoints/, plan.md, files/
    WorkspacePath() string
}

// Key methods
func (s *Session) Send(ctx context.Context, opts MessageOptions) (string, error)
func (s *Session) SendAndWait(ctx context.Context, opts MessageOptions) (*SessionEvent, error)
func (s *Session) On(handler SessionEventHandler) func()  // Returns unsubscribe
func (s *Session) RegisterTool(tool Tool)
func (s *Session) Destroy(ctx context.Context) error
```

### Message Options

```go
type MessageOptions struct {
    Prompt      string
    Attachments []Attachment
    Mode        string  // Optional mode hint
}

type Attachment struct {
    Type string // "file", "url", etc.
    Path string
}
```

---

## Event System

### Event Types

```go
const (
    AssistantMessage      = "assistant.message"       // Complete message
    AssistantMessageDelta = "assistant.message_delta" // Streaming chunk
    SessionIdle           = "session.idle"            // Processing complete
    SessionError          = "session.error"           // Error occurred
    ToolCall              = "tool.call"               // Tool invocation
    ToolResult            = "tool.result"             // Tool returned
)
```

### Event Handler Pattern

```go
session.On(func(event copilot.SessionEvent) {
    switch event.Type {
    case copilot.AssistantMessageDelta:
        fmt.Print(*event.Data.DeltaContent)
    case copilot.SessionIdle:
        fmt.Println() // Done
    case copilot.SessionError:
        log.Printf("Error: %s", *event.Data.Message)
    }
})
```

---

## Custom Tools

### DefineTool Helper

```go
// Automatic JSON schema generation from Go struct
type GetWeatherParams struct {
    City string `json:"city" jsonschema:"city name"`
    Unit string `json:"unit" jsonschema:"temperature unit"`
}

tool := copilot.DefineTool("get_weather", "Get weather for a city",
    func(params GetWeatherParams, inv copilot.ToolInvocation) (any, error) {
        return fmt.Sprintf("Weather in %s: 22°%s", params.City, params.Unit), nil
    })

session.RegisterTool(tool)
```

### Manual Tool Definition

```go
type Tool struct {
    Name        string
    Description string
    Parameters  map[string]any  // JSON Schema
    Handler     ToolHandler
}

type ToolHandler func(inv ToolInvocation) (ToolResult, error)

type ToolInvocation struct {
    ToolCallID string
    Name       string
    Arguments  map[string]any
}

type ToolResult struct {
    TextResultForLLM string
    ResultType       string // "success" | "error"
}
```

---

## Hooks (Interceptors)

### Pre-Tool Use Hook

Intercept before any tool executes. Can allow, deny, or modify.

```go
type PreToolUseHookInput struct {
    Timestamp int64
    Cwd       string
    ToolName  string
    ToolArgs  any
}

type PreToolUseHookOutput struct {
    PermissionDecision       string // "allow" | "deny" | "ask"
    PermissionDecisionReason string
    ModifiedArgs             any
    AdditionalContext        string
    SuppressOutput           bool
}

client.SetPreToolUseHandler(func(input PreToolUseHookInput, inv HookInvocation) (*PreToolUseHookOutput, error) {
    if input.ToolName == "git_commit" {
        // Enforce TDD: run tests before allowing commit
        if !testsPass() {
            return &PreToolUseHookOutput{
                PermissionDecision: "deny",
                PermissionDecisionReason: "Tests must pass before commit",
            }, nil
        }
    }
    return nil, nil // Allow by default
})
```

### Post-Tool Use Hook

Modify results or add context after tool executes.

```go
type PostToolUseHookInput struct {
    Timestamp  int64
    Cwd        string
    ToolName   string
    ToolArgs   any
    ToolResult any
}

type PostToolUseHookOutput struct {
    ModifiedResult    any
    AdditionalContext string
    SuppressOutput    bool
}

client.SetPostToolUseHandler(func(input PostToolUseHookInput, inv HookInvocation) (*PostToolUseHookOutput, error) {
    if input.ToolName == "bash" {
        // Log all bash commands for audit
        logCommand(input.ToolArgs, input.ToolResult)
    }
    return nil, nil
})
```

### Permission Handler

Handle permission requests from the agent.

```go
client.SetPermissionHandler(func(req PermissionRequest, inv PermissionInvocation) (PermissionRequestResult, error) {
    // Custom permission logic
    if req.Kind == "file_write" && isProtectedPath(req.Extra["path"]) {
        return PermissionRequestResult{Kind: "denied"}, nil
    }
    return PermissionRequestResult{Kind: "allowed"}, nil
})
```

### User Input Handler

Handle questions from the agent.

```go
client.SetUserInputHandler(func(req UserInputRequest, inv UserInputInvocation) (UserInputResponse, error) {
    // Could prompt user or auto-respond
    if req.Question == "Which database?" {
        return UserInputResponse{Answer: "PostgreSQL"}, nil
    }
    return UserInputResponse{}, fmt.Errorf("unknown question")
})
```

---

## BYOK (Bring Your Own Key)

Use your own LLM API keys instead of GitHub Copilot billing.

### Supported Providers

| Provider | Type | Notes |
|----------|------|-------|
| OpenAI | `"openai"` | Direct or compatible endpoints |
| Azure OpenAI | `"azure"` | Azure-hosted models |
| Anthropic | `"anthropic"` | Claude models |
| Ollama | `"openai"` | Local models |

### Azure Example

```go
session, _ := client.CreateSession(ctx, &copilot.SessionConfig{
    Model: "gpt-4.1-turbo",
    Provider: &copilot.ProviderConfig{
        Type:    "openai",
        BaseURL: "https://your-resource.openai.azure.com/openai/v1/",
        APIKey:  os.Getenv("AZURE_OPENAI_KEY"),
        WireApi: "responses",
    },
})
```

### Anthropic Example

```go
session, _ := client.CreateSession(ctx, &copilot.SessionConfig{
    Model: "claude-sonnet-4-20250514",
    Provider: &copilot.ProviderConfig{
        Type:    "anthropic",
        BaseURL: "https://api.anthropic.com",
        APIKey:  os.Getenv("ANTHROPIC_API_KEY"),
    },
})
```

---

## Complete Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    copilot "github.com/github/copilot-sdk/go"
)

func main() {
    ctx := context.Background()
    
    // Create client
    client := copilot.NewClient(&copilot.ClientOptions{
        Cwd: "/path/to/project",
    })
    if err := client.Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer client.Stop()
    
    // Enforce TDD via hook
    client.SetPreToolUseHandler(func(input copilot.PreToolUseHookInput, inv copilot.HookInvocation) (*copilot.PreToolUseHookOutput, error) {
        if input.ToolName == "git_commit" {
            // Check tests pass
            // ...
        }
        return nil, nil
    })
    
    // Create session with streaming
    session, err := client.CreateSession(ctx, &copilot.SessionConfig{
        Model:     "gpt-4.1",
        Streaming: true,
    })
    if err != nil {
        log.Fatal(err)
    }
    defer session.Destroy(ctx)
    
    // Register custom tool
    session.RegisterTool(copilot.DefineTool("run_tests", "Run project tests",
        func(params struct{}, inv copilot.ToolInvocation) (any, error) {
            // Run tests...
            return "All 42 tests passed", nil
        }))
    
    // Stream response
    session.On(func(event copilot.SessionEvent) {
        if event.Type == copilot.AssistantMessageDelta {
            fmt.Print(*event.Data.DeltaContent)
        }
    })
    
    // Send prompt
    _, err = session.SendAndWait(ctx, copilot.MessageOptions{
        Prompt: "Implement the auth module. Run tests before committing.",
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

---

## Integration with EAS

### Using Copilot SDK as Agent Runtime

```go
// EAS orchestrator using Copilot SDK
type Agent struct {
    TaskID   string
    Worktree string
    Session  *copilot.Session
}

func (a *Agent) Execute(ctx context.Context, task *Task) error {
    // 1. Create session in worktree
    session, _ := client.CreateSession(ctx, &copilot.SessionConfig{
        Model: "gpt-4.1",
    })
    a.Session = session
    
    // 2. Register EAS tools
    session.RegisterTool(copilot.DefineTool("eas_task_complete", "Mark task complete",
        func(p struct{}, inv copilot.ToolInvocation) (any, error) {
            return a.completeTask()
        }))
    
    // 3. Inject task context
    prompt := fmt.Sprintf(`
You are working on task %s in a TDD workflow.

## Specification
%s

## Acceptance Criteria
%s

## Instructions
1. Implement the feature
2. Run tests (they must pass)
3. Call eas_task_complete when done
`, task.ID, task.Spec, task.AcceptanceCriteria)
    
    // 4. Execute
    _, err := session.SendAndWait(ctx, copilot.MessageOptions{
        Prompt: prompt,
    })
    return err
}
```

### Parallel Orchestration

```go
func (o *Orchestrator) RunParallel(ctx context.Context, maxAgents int) {
    sem := make(chan struct{}, maxAgents)
    
    for {
        readyTasks := o.getReadyTasks()  // No blocking deps, tests not passing
        if len(readyTasks) == 0 {
            break
        }
        
        for _, task := range readyTasks {
            sem <- struct{}{}  // Acquire slot
            
            go func(t *Task) {
                defer func() { <-sem }()  // Release slot
                
                agent := o.spawnAgent(t)
                agent.Execute(ctx, t)
            }(task)
        }
    }
}
```

---

## Limitations

- ⚠️ **Technical Preview** — API may change
- ⚠️ **Requires Copilot CLI** — must be installed separately
- ⚠️ **Billing** — each prompt counts toward premium quota (unless BYOK)

---

## References

- [copilot-sdk repo](https://github.com/github/copilot-sdk)
- [Go cookbook](https://github.com/github/awesome-copilot/tree/main/cookbook/copilot-sdk/go)
- [Getting Started](https://github.com/github/copilot-sdk/blob/main/docs/getting-started.md)
- [BYOK Docs](https://github.com/github/copilot-sdk/blob/main/docs/auth/byok.md)
- [Authentication](https://github.com/github/copilot-sdk/blob/main/docs/auth/index.md)
