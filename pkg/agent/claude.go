package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/richgo/flo/pkg/task"
)

// ClaudeConfig holds configuration for the Claude backend.
type ClaudeConfig struct {
	CLIPath   string   // Path to claude binary
	Model     string   // Model name
	MCPConfig string   // Path to MCP config file
	ExtraArgs []string // Additional CLI arguments
}

// ClaudeBackend executes tasks using Claude Code CLI.
type ClaudeBackend struct {
	config ClaudeConfig
}

// NewClaudeBackend creates a new Claude backend.
func NewClaudeBackend(config ClaudeConfig) *ClaudeBackend {
	if config.CLIPath == "" {
		config.CLIPath = "claude"
	}
	return &ClaudeBackend{config: config}
}

func (b *ClaudeBackend) Name() string {
	return "claude"
}

func (b *ClaudeBackend) Start(ctx context.Context) error {
	return nil
}

func (b *ClaudeBackend) Stop() error {
	return nil
}

func (b *ClaudeBackend) CreateSession(ctx context.Context, t *task.Task, worktree string) (Session, error) {
	return &ClaudeSession{
		backend:  b,
		task:     t,
		worktree: worktree,
		events:   make(chan Event, 100),
	}, nil
}

func (b *ClaudeBackend) buildArgs(t *task.Task, worktree, prompt string) []string {
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

// ClaudeSession represents a Claude CLI session.
type ClaudeSession struct {
	backend  *ClaudeBackend
	task     *task.Task
	worktree string
	events   chan Event
	cmd      *exec.Cmd
}

func (s *ClaudeSession) Run(ctx context.Context, prompt string) (*Result, error) {
	args := s.backend.buildArgs(s.task, s.worktree, prompt)
	s.cmd = exec.CommandContext(ctx, s.backend.config.CLIPath, args...)

	stdout, err := s.cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := s.cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start claude: %w", err)
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

func (s *ClaudeSession) Events() <-chan Event {
	return s.events
}

func (s *ClaudeSession) Destroy(ctx context.Context) error {
	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
	}
	return nil
}

// streamEvent represents a Claude CLI stream-json event.
type streamEvent struct {
	Type    string        `json:"type"`
	Message *streamMessage `json:"message,omitempty"`
}

type streamMessage struct {
	Content []contentBlock `json:"content,omitempty"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}
