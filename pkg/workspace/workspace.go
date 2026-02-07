// Package workspace manages EAS feature workspaces.
package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/richgo/flo/pkg/audit"
	"github.com/richgo/flo/pkg/config"
	"github.com/richgo/flo/pkg/task"
)

const (
	easDir      = ".flo"
	configFile  = "config.yaml"
	specFile    = "SPEC.md"
	tasksDir    = "tasks"
	manifestFile = "manifest.json"
)

// Workspace represents an EAS feature workspace.
type Workspace struct {
	Root     string
	Feature  string
	Backend  string
	Config   *config.Config
	Tasks    *task.Registry
	nextID   int
}

// Status holds workspace status information.
type Status struct {
	Feature        string
	Backend        string
	TotalTasks     int
	PendingTasks   int
	InProgressTasks int
	CompleteTasks  int
	FailedTasks    int
	ReadyTasks     int
}

// Init initializes a new workspace in the given directory.
func Init(root, feature, backend string) (*Workspace, error) {
	easPath := filepath.Join(root, easDir)
	
	// Check if already initialized
	if _, err := os.Stat(easPath); err == nil {
		return nil, fmt.Errorf("workspace already initialized at %s", root)
	}

	// Create directory structure
	if err := os.MkdirAll(filepath.Join(easPath, tasksDir), 0755); err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	// Create config
	cfg := config.New(feature)
	cfg.Backend = backend
	if err := cfg.Save(filepath.Join(easPath, configFile)); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	// Create SPEC.md template
	specContent := fmt.Sprintf(`# Feature: %s

## Overview

_Describe the feature here._

## User Stories

1. As a user, I can...

## Acceptance Criteria

- [ ] Criterion 1
- [ ] Criterion 2

## Technical Notes

_Add technical details here._
`, feature)
	if err := os.WriteFile(filepath.Join(easPath, specFile), []byte(specContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to create SPEC.md: %w", err)
	}

	// Create empty task registry
	taskReg := task.NewRegistry()
	if err := taskReg.Save(filepath.Join(easPath, tasksDir, manifestFile)); err != nil {
		return nil, fmt.Errorf("failed to save task manifest: %w", err)
	}

	// Initialize audit logger
	if err := audit.Init(root); err != nil {
		// Log initialization failure but don't fail workspace init
		fmt.Fprintf(os.Stderr, "Warning: failed to initialize audit log: %v\n", err)
	} else {
		audit.Info("workspace.init", "Workspace initialized", map[string]interface{}{
			"feature": feature,
			"backend": backend,
			"root":    root,
		})
	}

	return &Workspace{
		Root:    root,
		Feature: feature,
		Backend: backend,
		Config:  cfg,
		Tasks:   taskReg,
		nextID:  1,
	}, nil
}

// Load loads an existing workspace from the given directory.
func Load(root string) (*Workspace, error) {
	easPath := filepath.Join(root, easDir)
	
	// Check if initialized
	if _, err := os.Stat(easPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("no workspace found at %s", root)
	}

	// Load config
	cfg, err := config.Load(filepath.Join(easPath, configFile))
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Load task registry
	taskReg := task.NewRegistry()
	manifestPath := filepath.Join(easPath, tasksDir, manifestFile)
	if _, err := os.Stat(manifestPath); err == nil {
		if err := taskReg.Load(manifestPath); err != nil {
			return nil, fmt.Errorf("failed to load tasks: %w", err)
		}
	}

	// Find highest task ID for next ID generation
	nextID := 1
	for _, t := range taskReg.List() {
		var id int
		if _, err := fmt.Sscanf(t.ID, "t-%d", &id); err == nil {
			if id >= nextID {
				nextID = id + 1
			}
		}
	}

	// Initialize audit logger
	if err := audit.Init(root); err != nil {
		// Log initialization failure but don't fail workspace load
		fmt.Fprintf(os.Stderr, "Warning: failed to initialize audit log: %v\n", err)
	} else {
		audit.Info("workspace.load", "Workspace loaded", map[string]interface{}{
			"feature":    cfg.Feature,
			"backend":    cfg.Backend,
			"task_count": len(taskReg.List()),
		})
	}

	return &Workspace{
		Root:    root,
		Feature: cfg.Feature,
		Backend: cfg.Backend,
		Config:  cfg,
		Tasks:   taskReg,
		nextID:  nextID,
	}, nil
}

// Save persists the workspace state.
func (w *Workspace) Save() error {
	easPath := filepath.Join(w.Root, easDir)
	
	if err := w.Config.Save(filepath.Join(easPath, configFile)); err != nil {
		audit.Error("workspace.save", "Failed to save config", map[string]interface{}{
			"error": err.Error(),
		})
		return fmt.Errorf("failed to save config: %w", err)
	}
	
	if err := w.Tasks.Save(filepath.Join(easPath, tasksDir, manifestFile)); err != nil {
		audit.Error("workspace.save", "Failed to save tasks", map[string]interface{}{
			"error": err.Error(),
		})
		return fmt.Errorf("failed to save tasks: %w", err)
	}
	
	audit.Info("workspace.save", "Workspace saved", map[string]interface{}{
		"task_count": len(w.Tasks.List()),
	})
	
	return nil
}

// CreateTask creates a new task in the workspace.
func (w *Workspace) CreateTask(title, repo string, deps []string, priority int) (*task.Task, error) {
	return w.CreateTaskWithType(title, "", repo, deps, priority)
}

// CreateTaskWithType creates a new task with a specific type.
func (w *Workspace) CreateTaskWithType(title, taskType, repo string, deps []string, priority int) (*task.Task, error) {
	id := fmt.Sprintf("t-%03d", w.nextID)
	w.nextID++

	t := task.New(id, title)
	t.Repo = repo
	t.Deps = deps
	t.Priority = priority
	t.Type = taskType
	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()

	// Set model based on task type
	if taskType != "" && w.Config.TaskTypes != nil {
		if typeConfig, ok := w.Config.TaskTypes[taskType]; ok {
			t.Model = typeConfig.Model
		}
	}

	if err := w.Tasks.Add(t); err != nil {
		w.nextID-- // Rollback ID
		audit.Error("workspace.create_task", "Failed to add task", map[string]interface{}{
			"task_id": id,
			"title":   title,
			"error":   err.Error(),
		})
		return nil, err
	}

	// Write task.md file
	if err := w.writeTaskFile(t); err != nil {
		audit.Error("workspace.create_task", "Failed to write task file", map[string]interface{}{
			"task_id": id,
			"error":   err.Error(),
		})
		// Don't fail the task creation if file write fails
	}

	// Auto-save
	if err := w.Save(); err != nil {
		audit.Error("workspace.create_task", "Failed to save after task creation", map[string]interface{}{
			"task_id": id,
			"error":   err.Error(),
		})
		return nil, err
	}

	audit.Info("workspace.create_task", "Task created", map[string]interface{}{
		"task_id":  id,
		"title":    title,
		"type":     taskType,
		"model":    t.Model,
		"repo":     repo,
		"deps":     deps,
		"priority": priority,
	})

	return t, nil
}

// GetTask returns a task by ID.
func (w *Workspace) GetTask(id string) (*task.Task, error) {
	return w.Tasks.Get(id)
}

// ListTasks returns tasks with optional filters.
func (w *Workspace) ListTasks(status, repo string) []*task.Task {
	if status != "" && repo != "" {
		// Both filters
		var result []*task.Task
		for _, t := range w.Tasks.List() {
			if string(t.Status) == status && t.Repo == repo {
				result = append(result, t)
			}
		}
		return result
	}
	if status != "" {
		return w.Tasks.ListByStatus(task.Status(status))
	}
	if repo != "" {
		return w.Tasks.ListByRepo(repo)
	}
	return w.Tasks.List()
}

// GetReadyTasks returns tasks that are ready to be worked on.
func (w *Workspace) GetReadyTasks() []*task.Task {
	return w.Tasks.GetReady()
}

// SetTaskStatus updates the status of a task and saves.
func (w *Workspace) SetTaskStatus(id string, status string) error {
	t, err := w.Tasks.Get(id)
	if err != nil {
		return err
	}
	
	oldStatus := t.Status
	if err := t.SetStatus(task.Status(status)); err != nil {
		return err
	}
	
	if err := w.Tasks.Update(t); err != nil {
		return err
	}
	
	if err := w.Save(); err != nil {
		return err
	}
	
	audit.Info("workspace.task_status", "Task status changed", map[string]interface{}{
		"task_id":    id,
		"old_status": oldStatus,
		"new_status": status,
	})
	
	return nil
}

// Status returns the current workspace status.
func (w *Workspace) Status() *Status {
	tasks := w.Tasks.List()
	
	status := &Status{
		Feature:    w.Feature,
		Backend:    w.Backend,
		TotalTasks: len(tasks),
	}

	for _, t := range tasks {
		switch t.Status {
		case task.StatusPending:
			status.PendingTasks++
		case task.StatusInProgress:
			status.InProgressTasks++
		case task.StatusComplete:
			status.CompleteTasks++
		case task.StatusFailed:
			status.FailedTasks++
		}
	}

	status.ReadyTasks = len(w.GetReadyTasks())

	return status
}

// SpecPath returns the path to the SPEC.md file.
func (w *Workspace) SpecPath() string {
	return filepath.Join(w.Root, easDir, specFile)
}

// ReadSpec reads the SPEC.md contents.
func (w *Workspace) ReadSpec() (string, error) {
	data, err := os.ReadFile(w.SpecPath())
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// writeTaskFile writes a task.md file with YAML frontmatter.
func (w *Workspace) writeTaskFile(t *task.Task) error {
	easPath := filepath.Join(w.Root, easDir)
	taskPath := filepath.Join(easPath, tasksDir, fmt.Sprintf("TASK-%s.md", t.ID))

	// Build YAML frontmatter
	frontmatter := fmt.Sprintf(`---
id: %s
status: %s`, t.ID, t.Status)

	if t.Model != "" {
		frontmatter += fmt.Sprintf("\nmodel: %s", t.Model)
	}
	if t.Fallback != "" {
		frontmatter += fmt.Sprintf("\nfallback: %s", t.Fallback)
	}
	if t.Type != "" {
		frontmatter += fmt.Sprintf("\ntype: %s", t.Type)
	}
	if t.Priority > 0 {
		frontmatter += fmt.Sprintf("\npriority: %d", t.Priority)
	}
	if t.Repo != "" {
		frontmatter += fmt.Sprintf("\nrepo: %s", t.Repo)
	}
	if len(t.Deps) > 0 {
		frontmatter += "\ndeps:"
		for _, dep := range t.Deps {
			frontmatter += fmt.Sprintf("\n  - %s", dep)
		}
	}

	frontmatter += "\n---\n\n"

	// Build body
	body := fmt.Sprintf("# %s\n", t.Title)
	if t.Description != "" {
		body += fmt.Sprintf("\n%s\n", t.Description)
	}

	// Add TDD enforcement section
	body += `
## TDD Requirements

**This task MUST follow Test-Driven Development:**

1. **Write tests first** - Before implementing any feature, write failing tests
2. **Red → Green → Refactor** - Follow the TDD cycle strictly
3. **Run tests continuously** - Use ` + "`flo test`" + ` or ` + "`make test`" + ` after each change
4. **No implementation without tests** - Every new function/method needs test coverage
5. **Tests must pass before completion** - Task cannot be marked complete with failing tests

### Completion Checklist
- [ ] Tests written for new functionality
- [ ] All tests passing
- [ ] Coverage maintained or improved
- [ ] No regressions introduced
`

	content := frontmatter + body

	if err := os.WriteFile(taskPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write task file: %w", err)
	}

	return nil
}
