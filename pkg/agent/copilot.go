package agent

import (
	"context"
	"fmt"

	"github.com/richgo/flo/pkg/task"
)

// CopilotConfig holds configuration for the Copilot backend.
type CopilotConfig struct {
	CLIPath  string          // Path to copilot binary
	Model    string          // Model name
	Provider *ProviderConfig // BYOK settings
}

// ProviderConfig holds BYOK provider settings.
type ProviderConfig struct {
	Type      string // "openai" | "azure" | "anthropic"
	BaseURL   string // API endpoint
	APIKeyEnv string // Environment variable for API key
}

// CopilotBackend executes tasks using GitHub Copilot SDK.
// Note: Full implementation requires the Copilot SDK dependency.
type CopilotBackend struct {
	config CopilotConfig
}

// NewCopilotBackend creates a new Copilot backend.
func NewCopilotBackend(config CopilotConfig) *CopilotBackend {
	if config.CLIPath == "" {
		config.CLIPath = "copilot"
	}
	return &CopilotBackend{config: config}
}

func (b *CopilotBackend) Name() string {
	return "copilot"
}

func (b *CopilotBackend) Start(ctx context.Context) error {
	// TODO: Initialize Copilot SDK client
	return nil
}

func (b *CopilotBackend) Stop() error {
	// TODO: Stop Copilot SDK client
	return nil
}

func (b *CopilotBackend) CreateSession(ctx context.Context, t *task.Task, worktree string) (Session, error) {
	return &CopilotSession{
		backend:  b,
		task:     t,
		worktree: worktree,
		events:   make(chan Event, 100),
	}, nil
}

// CopilotSession represents a Copilot SDK session.
type CopilotSession struct {
	backend  *CopilotBackend
	task     *task.Task
	worktree string
	events   chan Event
}

func (s *CopilotSession) Run(ctx context.Context, prompt string) (*Result, error) {
	// TODO: Implement using Copilot SDK
	// For now, return a placeholder
	close(s.events)
	return &Result{
		Success: false,
		Error:   fmt.Sprintf("Copilot backend not yet implemented - requires SDK dependency"),
	}, nil
}

func (s *CopilotSession) Events() <-chan Event {
	return s.events
}

func (s *CopilotSession) Destroy(ctx context.Context) error {
	return nil
}
