package tools

import (
	"encoding/json"
	"fmt"

	"github.com/richgo/enterprise-ai-sdlc/pkg/task"
)

// TestRunner is the interface for running tests.
type TestRunner interface {
	Run(taskID string) (pass bool, output string, err error)
}

// EASToolsConfig holds the configuration for EAS tools.
type EASToolsConfig struct {
	SpecPath string // Path to SPEC.md
}

// NewEASTools creates a tool registry with all EAS tools registered.
func NewEASTools(taskReg *task.Registry, testRunner TestRunner) *Registry {
	reg := NewRegistry()

	// eas_task_list
	reg.Register(New(
		"eas_task_list",
		"List tasks with optional filters. Returns JSON array of tasks.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"status": map[string]any{
					"type":        "string",
					"description": "Filter by status: pending, in_progress, complete, failed",
				},
				"repo": map[string]any{
					"type":        "string",
					"description": "Filter by repository name",
				},
			},
		},
		func(args Args) (string, error) {
			return handleTaskList(taskReg, args)
		},
	))

	// eas_task_get
	reg.Register(New(
		"eas_task_get",
		"Get detailed information about a specific task.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"task_id": map[string]any{
					"type":        "string",
					"description": "Task ID (e.g., ua-001)",
				},
			},
			"required": []any{"task_id"},
		},
		func(args Args) (string, error) {
			return handleTaskGet(taskReg, args)
		},
	))

	// eas_task_claim
	reg.Register(New(
		"eas_task_claim",
		"Claim a task (sets status to in_progress). Task must be pending with all deps complete.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"task_id": map[string]any{
					"type":        "string",
					"description": "Task ID to claim",
				},
			},
			"required": []any{"task_id"},
		},
		func(args Args) (string, error) {
			return handleTaskClaim(taskReg, args)
		},
	))

	// eas_task_complete
	reg.Register(New(
		"eas_task_complete",
		"Mark task as complete. Runs tests first - will fail if tests don't pass.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"task_id": map[string]any{
					"type":        "string",
					"description": "Task ID to complete",
				},
			},
			"required": []any{"task_id"},
		},
		func(args Args) (string, error) {
			return handleTaskComplete(taskReg, testRunner, args)
		},
	))

	// eas_run_tests
	reg.Register(New(
		"eas_run_tests",
		"Run tests for a task. Returns test output and pass/fail status.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"task_id": map[string]any{
					"type":        "string",
					"description": "Task ID to run tests for",
				},
			},
			"required": []any{"task_id"},
		},
		func(args Args) (string, error) {
			return handleRunTests(testRunner, args)
		},
	))

	return reg
}

func handleTaskList(taskReg *task.Registry, args Args) (string, error) {
	var tasks []*task.Task

	// Apply filters
	statusFilter, hasStatus := args["status"].(string)
	repoFilter, hasRepo := args["repo"].(string)

	if hasStatus && hasRepo {
		// Both filters
		allTasks := taskReg.List()
		for _, t := range allTasks {
			if string(t.Status) == statusFilter && t.Repo == repoFilter {
				tasks = append(tasks, t)
			}
		}
	} else if hasStatus {
		tasks = taskReg.ListByStatus(task.Status(statusFilter))
	} else if hasRepo {
		tasks = taskReg.ListByRepo(repoFilter)
	} else {
		tasks = taskReg.List()
	}

	// Handle nil slice
	if tasks == nil {
		tasks = []*task.Task{}
	}

	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to serialize tasks: %w", err)
	}

	return string(data), nil
}

func handleTaskGet(taskReg *task.Registry, args Args) (string, error) {
	taskID, ok := args["task_id"].(string)
	if !ok {
		return "", fmt.Errorf("task_id is required")
	}

	t, err := taskReg.Get(taskID)
	if err != nil {
		return "", err
	}

	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to serialize task: %w", err)
	}

	return string(data), nil
}

func handleTaskClaim(taskReg *task.Registry, args Args) (string, error) {
	taskID, ok := args["task_id"].(string)
	if !ok {
		return "", fmt.Errorf("task_id is required")
	}

	t, err := taskReg.Get(taskID)
	if err != nil {
		return "", err
	}

	// Check if task is pending
	if t.Status != task.StatusPending {
		return "", fmt.Errorf("task '%s' is not pending (status: %s)", taskID, t.Status)
	}

	// Check if all deps are complete
	deps, _ := taskReg.GetDeps(taskID)
	for _, dep := range deps {
		if dep.Status != task.StatusComplete {
			return "", fmt.Errorf("dependency '%s' is not complete (status: %s)", dep.ID, dep.Status)
		}
	}

	// Claim the task
	if err := t.SetStatus(task.StatusInProgress); err != nil {
		return "", err
	}
	if err := taskReg.Update(t); err != nil {
		return "", err
	}

	return fmt.Sprintf("Task '%s' claimed successfully", taskID), nil
}

func handleTaskComplete(taskReg *task.Registry, testRunner TestRunner, args Args) (string, error) {
	taskID, ok := args["task_id"].(string)
	if !ok {
		return "", fmt.Errorf("task_id is required")
	}

	t, err := taskReg.Get(taskID)
	if err != nil {
		return "", err
	}

	// Check if task is in progress
	if t.Status != task.StatusInProgress {
		return "", fmt.Errorf("task '%s' is not in progress (status: %s)", taskID, t.Status)
	}

	// Run tests if test runner is configured
	if testRunner != nil {
		pass, output, err := testRunner.Run(taskID)
		if err != nil {
			return "", fmt.Errorf("failed to run tests: %w", err)
		}
		if !pass {
			return "", fmt.Errorf("tests failed - cannot complete task:\n%s", output)
		}
	}

	// Complete the task
	if err := t.SetStatus(task.StatusComplete); err != nil {
		return "", err
	}
	if err := taskReg.Update(t); err != nil {
		return "", err
	}

	return fmt.Sprintf("Task '%s' completed successfully", taskID), nil
}

func handleRunTests(testRunner TestRunner, args Args) (string, error) {
	taskID, ok := args["task_id"].(string)
	if !ok {
		return "", fmt.Errorf("task_id is required")
	}

	if testRunner == nil {
		return "No test runner configured", nil
	}

	pass, output, err := testRunner.Run(taskID)
	if err != nil {
		return "", fmt.Errorf("failed to run tests: %w", err)
	}

	result := map[string]any{
		"task_id": taskID,
		"pass":    pass,
		"output":  output,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return string(data), nil
}
