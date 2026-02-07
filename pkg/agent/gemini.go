package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/richgo/flo/pkg/task"
)

// GeminiConfig holds configuration for the Gemini backend.
type GeminiConfig struct {
	CLIPath   string   // Path to gemini binary
	Model     string   // Model name
	MCPConfig string   // Path to MCP config file
	ExtraArgs []string // Additional CLI arguments
}

// GeminiBackend executes tasks using Gemini CLI.
type GeminiBackend struct {
	config GeminiConfig
}

// NewGeminiBackend creates a new Gemini backend.
func NewGeminiBackend(config GeminiConfig) *GeminiBackend {
	if config.CLIPath == "" {
		config.CLIPath = "gemini"
	}
	return &GeminiBackend{config: config}
}

func (b *GeminiBackend) Name() string {
	return "gemini"
}

func (b *GeminiBackend) Start(ctx context.Context) error {
	return nil
}

func (b *GeminiBackend) Stop() error {
	return nil
}

func (b *GeminiBackend) CreateSession(ctx context.Context, t *task.Task, worktree string) (Session, error) {
	return &GeminiSession{
		backend:  b,
		task:     t,
		worktree: worktree,
		events:   make(chan Event, 100),
	}, nil
}

func (b *GeminiBackend) buildArgs(t *task.Task, worktree, prompt string) []string {
	args := []string{
		"--print",
		"--output-format", "stream-json",
	}

	if b.config.Model != "" {
		args = append(args, "--model", b.config.Model)
	}

	if b.config.MCPConfig != "" {
		args = append(args, "--mcp-config", b.config.MCPConfig)
	}

	if worktree != "" {
		args = append(args, "--cwd", worktree)
	}

	args = append(args, b.config.ExtraArgs...)
	args = append(args, prompt)

	return args
}

// GeminiSession represents a Gemini CLI session.
type GeminiSession struct {
	backend  *GeminiBackend
	task     *task.Task
	worktree string
	events   chan Event
	cmd      *exec.Cmd
}

func (s *GeminiSession) Run(ctx context.Context, prompt string) (*Result, error) {
	args := s.backend.buildArgs(s.task, s.worktree, prompt)
	s.cmd = exec.CommandContext(ctx, s.backend.config.CLIPath, args...)

	stdout, err := s.cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := s.cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start gemini: %w", err)
	}

	// Read and process output
	var lastMessage string
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		
		var event streamEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue // Skip non-JSON lines
		}

		switch event.Type {
		case "assistant":
			if event.Message != nil && event.Message.Content != nil {
				for _, block := range event.Message.Content {
					if block.Type == "text" {
						lastMessage = block.Text
						s.events <- Event{Type: "message", Content: block.Text}
					}
				}
			}
		case "result":
			s.events <- Event{Type: "complete", Content: "done"}
		}
	}
	close(s.events)

	if err := s.cmd.Wait(); err != nil {
		return &Result{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &Result{
		Success: true,
		Output:  lastMessage,
	}, nil
}

func (s *GeminiSession) Events() <-chan Event {
	return s.events
}

func (s *GeminiSession) Destroy(ctx context.Context) error {
	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
	}
	return nil
}
