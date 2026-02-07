package agent

import (
	"context"
	"testing"

	"github.com/richgo/flo/pkg/task"
)

func TestMockBackend(t *testing.T) {
	backend := NewMockBackend()

	if backend.Name() != "mock" {
		t.Errorf("expected name 'mock', got '%s'", backend.Name())
	}

	ctx := context.Background()

	if err := backend.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	task := task.New("t-001", "Test task")
	session, err := backend.CreateSession(ctx, task, "/tmp/worktree")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Configure mock response
	backend.SetResponse(Result{
		Success: true,
		Output:  "Task completed",
	})

	result, err := session.Run(ctx, "Do the task")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if !result.Success {
		t.Error("expected success")
	}
	if result.Output != "Task completed" {
		t.Errorf("expected 'Task completed', got '%s'", result.Output)
	}

	// Verify calls recorded
	calls := backend.GetCalls()
	if len(calls) != 1 {
		t.Errorf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Prompt != "Do the task" {
		t.Errorf("expected prompt 'Do the task', got '%s'", calls[0].Prompt)
	}

	if err := session.Destroy(ctx); err != nil {
		t.Fatalf("Destroy failed: %v", err)
	}

	if err := backend.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestMockBackendEvents(t *testing.T) {
	backend := NewMockBackend()
	ctx := context.Background()
	backend.Start(ctx)

	task := task.New("t-001", "Test")
	session, _ := backend.CreateSession(ctx, task, "/tmp")

	// Configure events
	backend.SetEvents([]Event{
		{Type: "message", Content: "Starting..."},
		{Type: "tool_call", Content: "run_tests"},
		{Type: "complete", Content: "Done"},
	})

	backend.SetResponse(Result{Success: true})

	// Start run in background
	go session.Run(ctx, "test")

	// Collect events
	var events []Event
	for event := range session.Events() {
		events = append(events, event)
	}

	if len(events) != 3 {
		t.Errorf("expected 3 events, got %d", len(events))
	}
}

func TestClaudeBackendConfig(t *testing.T) {
	config := ClaudeConfig{
		CLIPath:   "/usr/local/bin/claude",
		Model:     "claude-sonnet-4-5-20250514",
		MCPConfig: "/path/to/.eas/mcp.json",
	}

	backend := NewClaudeBackend(config)

	if backend.Name() != "claude" {
		t.Errorf("expected name 'claude', got '%s'", backend.Name())
	}

	// Verify config stored
	if backend.config.CLIPath != "/usr/local/bin/claude" {
		t.Error("CLIPath not stored correctly")
	}
	if backend.config.Model != "claude-sonnet-4-5-20250514" {
		t.Error("Model not stored correctly")
	}
}

func TestClaudeBackendBuildCommand(t *testing.T) {
	config := ClaudeConfig{
		CLIPath:   "claude",
		Model:     "sonnet",
		MCPConfig: "/tmp/mcp.json",
		ExtraArgs: []string{"--dangerously-skip-permissions"},
	}

	backend := NewClaudeBackend(config)

	task := task.New("t-001", "Test")
	args := backend.buildArgs(task, "/tmp/worktree", "Do something")

	// Check required args present
	found := map[string]bool{}
	for i, arg := range args {
		if arg == "--print" {
			found["print"] = true
		}
		if arg == "--model" && i+1 < len(args) {
			if args[i+1] == "sonnet" {
				found["model"] = true
			}
		}
		if arg == "--mcp-config" && i+1 < len(args) {
			if args[i+1] == "/tmp/mcp.json" {
				found["mcp"] = true
			}
		}
		if arg == "--dangerously-skip-permissions" {
			found["extra"] = true
		}
	}

	if !found["print"] {
		t.Error("--print not found in args")
	}
	if !found["model"] {
		t.Error("--model not found in args")
	}
	if !found["mcp"] {
		t.Error("--mcp-config not found in args")
	}
	if !found["extra"] {
		t.Error("extra args not found")
	}
}

func TestNewBackendByName(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"claude", "claude"},
		{"copilot", "copilot"},
		{"mock", "mock"},
	}

	for _, tt := range tests {
		backend := NewBackendByName(tt.name, nil)
		if backend == nil {
			t.Errorf("NewBackendByName(%s) returned nil", tt.name)
			continue
		}
		if backend.Name() != tt.expected {
			t.Errorf("expected name '%s', got '%s'", tt.expected, backend.Name())
		}
	}
}

func TestNewBackendByNameUnknown(t *testing.T) {
	backend := NewBackendByName("unknown", nil)
	if backend != nil {
		t.Error("expected nil for unknown backend")
	}
}
