// Package task provides the core Task data model for EAS.
package task

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/richgo/flo/pkg/audit"
	"gopkg.in/yaml.v3"
)

// Status represents the current state of a task.
type Status string

const (
	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusComplete   Status = "complete"
	StatusFailed     Status = "failed"
)

// IsValid returns true if the status is a known valid status.
func (s Status) IsValid() bool {
	switch s {
	case StatusPending, StatusInProgress, StatusComplete, StatusFailed:
		return true
	default:
		return false
	}
}

// Task represents a unit of work within a feature.
type Task struct {
	ID          string    `json:"id" yaml:"id"`
	Title       string    `json:"title" yaml:"title"`
	Description string    `json:"description,omitempty" yaml:"description,omitempty"`
	Status      Status    `json:"status" yaml:"status"`
	Priority    int       `json:"priority,omitempty" yaml:"priority,omitempty"`
	Repo        string    `json:"repo,omitempty" yaml:"repo,omitempty"`
	Deps        []string  `json:"deps,omitempty" yaml:"deps,omitempty"`
	SpecRef     string    `json:"spec_ref,omitempty" yaml:"spec_ref,omitempty"`
	Model       string    `json:"model,omitempty" yaml:"model,omitempty"`
	Fallback    string    `json:"fallback,omitempty" yaml:"fallback,omitempty"`
	Type        string    `json:"type,omitempty" yaml:"type,omitempty"`
	CreatedAt   time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" yaml:"updated_at"`
}

// New creates a new Task with the given ID and title.
// Status defaults to pending, timestamps are set automatically.
func New(id, title string) *Task {
	now := time.Now()
	return &Task{
		ID:        id,
		Title:     title,
		Status:    StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Validate checks if the task has valid required fields.
func (t *Task) Validate() error {
	if t.ID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}
	if t.Title == "" {
		return fmt.Errorf("task title cannot be empty")
	}
	if t.Status != "" && !t.Status.IsValid() {
		return fmt.Errorf("invalid status: %s", t.Status)
	}
	return nil
}

// validTransitions defines allowed status transitions.
// Key is current status, value is set of allowed next statuses.
var validTransitions = map[Status]map[Status]bool{
	StatusPending: {
		StatusInProgress: true,
	},
	StatusInProgress: {
		StatusComplete: true,
		StatusFailed:   true,
	},
	StatusComplete: {
		// Terminal state - no transitions allowed
	},
	StatusFailed: {
		StatusPending: true, // Allow retry
	},
}

// SetStatus changes the task status if the transition is valid.
// Returns an error if the transition is not allowed.
func (t *Task) SetStatus(newStatus Status) error {
	if t.Status == newStatus {
		return nil // No change
	}

	allowed, ok := validTransitions[t.Status]
	if !ok {
		audit.Error("task.set_status", "Unknown current status", map[string]interface{}{
			"task_id":        t.ID,
			"current_status": string(t.Status),
			"new_status":     string(newStatus),
		})
		return fmt.Errorf("unknown current status: %s", t.Status)
	}

	if !allowed[newStatus] {
		audit.Warn("task.set_status", "Invalid status transition", map[string]interface{}{
			"task_id":    t.ID,
			"from":       string(t.Status),
			"to":         string(newStatus),
			"task_title": t.Title,
		})
		return fmt.Errorf("invalid status transition: %s -> %s", t.Status, newStatus)
	}

	oldStatus := t.Status
	t.Status = newStatus
	t.UpdatedAt = time.Now()
	
	audit.Info("task.set_status", "Task status changed", map[string]interface{}{
		"task_id":    t.ID,
		"task_title": t.Title,
		"from":       string(oldStatus),
		"to":         string(newStatus),
	})
	
	return nil
}

// IsReady returns true if the task is pending and could be started.
// Note: This doesn't check dependencies - use Registry.IsReady() for that.
func (t *Task) IsReady() bool {
	return t.Status == StatusPending
}

// IsComplete returns true if the task has completed successfully.
func (t *Task) IsComplete() bool {
	return t.Status == StatusComplete
}

// IsTerminal returns true if the task is in a terminal state (complete or failed).
func (t *Task) IsTerminal() bool {
	return t.Status == StatusComplete || t.Status == StatusFailed
}

// ParseTaskFile reads a task from a task.md file with YAML frontmatter.
func ParseTaskFile(path string) (*Task, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read task file: %w", err)
	}

	content := string(data)
	
	// Check for YAML frontmatter (--- ... ---)
	if !strings.HasPrefix(content, "---\n") {
		return nil, fmt.Errorf("task file missing YAML frontmatter")
	}

	// Find end of frontmatter
	endIdx := strings.Index(content[4:], "\n---\n")
	if endIdx == -1 {
		return nil, fmt.Errorf("task file has invalid frontmatter")
	}
	endIdx += 4 // Adjust for the offset

	// Extract frontmatter and body
	frontmatter := content[4:endIdx]
	body := strings.TrimSpace(content[endIdx+5:])

	// Parse YAML frontmatter
	var task Task
	if err := yaml.Unmarshal([]byte(frontmatter), &task); err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Extract title and description from body
	if body != "" {
		lines := strings.Split(body, "\n")
		for i, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "# ") {
				task.Title = strings.TrimPrefix(line, "# ")
				// Rest is description
				if i+1 < len(lines) {
					task.Description = strings.TrimSpace(strings.Join(lines[i+1:], "\n"))
				}
				break
			}
		}
	}

	return &task, nil
}
