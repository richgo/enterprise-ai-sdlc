// Package agent provides backend abstractions for AI agent execution.
package agent

import (
	"context"

	"github.com/richgo/flo/pkg/task"
)

// Backend is the interface for agent execution backends.
type Backend interface {
	Name() string
	Start(ctx context.Context) error
	Stop() error
	CreateSession(ctx context.Context, task *task.Task, worktree string) (Session, error)
}

// Session represents an agent session for executing a task.
type Session interface {
	Run(ctx context.Context, prompt string) (*Result, error)
	Events() <-chan Event
	Destroy(ctx context.Context) error
}

// Result represents the outcome of an agent run.
type Result struct {
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
}

// Event represents a streaming event during agent execution.
type Event struct {
	Type    string `json:"type"`    // "message", "tool_call", "complete", "error"
	Content string `json:"content"`
}

// Call records a call to a mock backend for verification.
type Call struct {
	TaskID   string
	Worktree string
	Prompt   string
}

// NewBackendByName creates a backend by name.
func NewBackendByName(name string, config any) Backend {
	switch name {
	case "claude":
		if cfg, ok := config.(*ClaudeConfig); ok {
			return NewClaudeBackend(*cfg)
		}
		return NewClaudeBackend(ClaudeConfig{})
	case "copilot":
		if cfg, ok := config.(*CopilotConfig); ok {
			return NewCopilotBackend(*cfg)
		}
		return NewCopilotBackend(CopilotConfig{})
	case "codex":
		if cfg, ok := config.(*CodexConfig); ok {
			return NewCodexBackend(*cfg)
		}
		return NewCodexBackend(CodexConfig{})
	case "gemini":
		if cfg, ok := config.(*GeminiConfig); ok {
			return NewGeminiBackend(*cfg)
		}
		return NewGeminiBackend(GeminiConfig{})
	case "mock":
		return NewMockBackend()
	default:
		return nil
	}
}
