package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/richgo/flo/pkg/agent"
	"github.com/richgo/flo/pkg/quota"
	"github.com/richgo/flo/pkg/task"
	"github.com/richgo/flo/pkg/workspace"
)

var workBackend string

var workCmd = &cobra.Command{
	Use:   "work <task-id>",
	Short: "Start agent work on a task",
	Long: `Start an AI agent to work on the specified task.

The agent will:
1. Read the task specification
2. Implement the required changes
3. Run tests (TDD enforcement)
4. Complete the task when tests pass

Uses the configured backend (claude or copilot) unless overridden.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]

		ws, err := loadWorkspace()
		if err != nil {
			return err
		}

		// Get the task
		t, err := ws.GetTask(taskID)
		if err != nil {
			return err
		}

		// Check task is ready
		if t.Status != task.StatusPending {
			return fmt.Errorf("task %s is not pending (status: %s)", taskID, t.Status)
		}

		// Check deps complete
		ready := ws.GetReadyTasks()
		isReady := false
		for _, r := range ready {
			if r.ID == taskID {
				isReady = true
				break
			}
		}
		if !isReady {
			return fmt.Errorf("task %s has incomplete dependencies", taskID)
		}

		// Try to read task.md file to get model from frontmatter
		taskMDPath := filepath.Join(ws.Root, ".flo", "tasks", fmt.Sprintf("TASK-%s.md", taskID))
		if taskFromFile, err := task.ParseTaskFile(taskMDPath); err == nil && taskFromFile.Model != "" {
			// Update task with model from frontmatter
			t.Model = taskFromFile.Model
			t.Fallback = taskFromFile.Fallback
		}

		// Determine backend and model
		backendName := ws.Backend
		model := ""
		
		if workBackend != "" {
			backendName = workBackend
		} else if t.Model != "" {
			// Parse model format: "backend/model" (e.g., "claude/sonnet", "copilot/gpt-4")
			parts := strings.Split(t.Model, "/")
			if len(parts) == 2 {
				backendName = parts[0]
				model = parts[1]
			}
		}

		fmt.Printf("🚀 Starting work on task: %s\n", taskID)
		fmt.Printf("   Title: %s\n", t.Title)
		fmt.Printf("   Backend: %s\n", backendName)
		if model != "" {
			fmt.Printf("   Model: %s\n", model)
		}

		// Claim the task
		if err := t.SetStatus(task.StatusInProgress); err != nil {
			return err
		}
		ws.Tasks.Update(t)
		ws.Save()

		// Initialize quota tracker
		quotaPath := filepath.Join(ws.Root, ".flo", "quota.json")
		quotaTracker := initQuotaTracker(quotaPath, ws)

		// Attempt to run with primary backend, fallback if needed
		ctx := context.Background()
		result, err := runWithFailover(ctx, ws, t, backendName, model, quotaTracker)
		
		if err != nil {
			return fmt.Errorf("agent failed: %w", err)
		}

		if result.Success {
			fmt.Printf("\n✅ Task %s completed successfully\n", taskID)
		} else {
			fmt.Printf("\n❌ Task %s failed: %s\n", taskID, result.Error)
			// Revert status
			t.SetStatus(task.StatusFailed)
			ws.Tasks.Update(t)
			ws.Save()
		}

		return nil
	},
}

// runWithFailover attempts to run a task with the primary backend, and falls back to the fallback model if quota is exhausted.
func runWithFailover(ctx context.Context, ws *workspace.Workspace, t *task.Task, backendName, model string, tracker *quota.Tracker) (*agent.Result, error) {
	// Try primary backend
	result, err := runBackend(ctx, ws, t, backendName, model, tracker)
	
	// Check if we hit quota exhaustion
	if err != nil && isQuotaError(err) && t.Fallback != "" {
		fmt.Printf("\n⚠️  Quota exhausted for %s, failing over to %s\n", backendName, t.Fallback)
		
		// Parse fallback model
		parts := strings.Split(t.Fallback, "/")
		if len(parts) == 2 {
			fallbackBackend := parts[0]
			fallbackModel := parts[1]
			
			// Record the failover
			tracker.RecordError(backendName, time.Hour)
			
			fmt.Printf("🔄 Retrying with fallback backend: %s/%s\n", fallbackBackend, fallbackModel)
			
			// Try fallback
			result, err = runBackend(ctx, ws, t, fallbackBackend, fallbackModel, tracker)
		}
	}
	
	return result, err
}

// runBackend executes a task with a specific backend.
func runBackend(ctx context.Context, ws *workspace.Workspace, t *task.Task, backendName, model string, tracker *quota.Tracker) (*agent.Result, error) {
	// Check if backend is exhausted before starting
	if tracker.IsExhausted(backendName) {
		return nil, fmt.Errorf("quota exhausted for backend %s", backendName)
	}

	// Create backend
	var backend agent.Backend
	switch backendName {
	case "claude":
		mcpConfig := filepath.Join(ws.Root, ".eas", "mcp.json")
		// Generate MCP config
		if err := generateMCPConfig(mcpConfig, ws.Root); err != nil {
			return nil, fmt.Errorf("failed to generate MCP config: %w", err)
		}
		claudeModel := ws.Config.Claude.Model
		if model != "" {
			claudeModel = model
		}
		backend = agent.NewClaudeBackend(agent.ClaudeConfig{
			MCPConfig: mcpConfig,
			Model:     claudeModel,
		})
	case "copilot":
		copilotModel := ws.Config.Copilot.Model
		if model != "" {
			copilotModel = model
		}
		backend = agent.NewCopilotBackend(agent.CopilotConfig{
			Model: copilotModel,
		})
	default:
		return nil, fmt.Errorf("unknown backend: %s", backendName)
	}

	if err := backend.Start(ctx); err != nil {
		// Check if this is a quota error
		if isQuotaError(err) {
			tracker.RecordError(backendName, time.Hour)
		}
		return nil, fmt.Errorf("failed to start backend: %w", err)
	}
	defer backend.Stop()

	// Read spec for context
	spec, _ := ws.ReadSpec()

	// Build prompt
	prompt := fmt.Sprintf(`You are working on task %s in a TDD workflow.

## Task
Title: %s
%s

## Feature Specification
%s

## Instructions
1. Implement the required changes for this task
2. Run tests using eas_run_tests to verify your implementation
3. When tests pass, call eas_task_complete to finish the task

Available tools:
- eas_task_get: Get task details
- eas_run_tests: Run tests for the task
- eas_task_complete: Mark task complete (requires tests to pass)
- eas_spec_read: Read the feature specification

Begin implementing the task.`, t.ID, t.Title, t.Description, spec)

	// Create session
	session, err := backend.CreateSession(ctx, t, ws.Root)
	if err != nil {
		if isQuotaError(err) {
			tracker.RecordError(backendName, time.Hour)
		}
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Destroy(ctx)

	// Stream events
	go func() {
		for event := range session.Events() {
			switch event.Type {
			case "message":
				fmt.Print(event.Content)
			case "tool_call":
				fmt.Printf("\n🔧 %s\n", event.Content)
			case "complete":
				fmt.Println("\n✅ Complete")
			case "error":
				fmt.Printf("\n❌ Error: %s\n", event.Content)
			}
		}
	}()

	// Run the agent
	result, err := session.Run(ctx, prompt)
	if err != nil {
		if isQuotaError(err) {
			tracker.RecordError(backendName, time.Hour)
		}
		return nil, err
	}
	
	// Record successful usage (approximate token count)
	if result.Success {
		tracker.Record(backendName, 10000) // Estimate, actual would come from API
	}
	
	return result, nil
}

// isQuotaError checks if an error is related to quota exhaustion.
func isQuotaError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "429") ||
		strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "quota") ||
		strings.Contains(errStr, "too many requests")
}

// initQuotaTracker initializes the quota tracker with limits from config.
func initQuotaTracker(path string, ws *workspace.Workspace) *quota.Tracker {
	tracker := quota.New(path)
	tracker.Load()
	
	// Set limits from config if available
	// Default limits for common backends
	tracker.SetLimit("claude", 50)  // 50 requests per hour for premium
	tracker.SetLimit("copilot", 100) // Higher limit for copilot
	
	return tracker
}

func init() {
	workCmd.Flags().StringVar(&workBackend, "backend", "", "Override backend (claude or copilot)")
	rootCmd.AddCommand(workCmd)
}

func generateMCPConfig(path, workspaceRoot string) error {
	cwd, _ := os.Getwd()
	easBinary := filepath.Join(cwd, "eas")
	
	// Check if eas exists in current dir, otherwise use PATH
	if _, err := os.Stat(easBinary); os.IsNotExist(err) {
		easBinary = "eas"
	}

	config := map[string]any{
		"mcpServers": map[string]any{
			"eas": map[string]any{
				"command": easBinary,
				"args":    []string{"mcp", "serve"},
				"cwd":     workspaceRoot,
			},
		},
	}

	data, _ := json.MarshalIndent(config, "", "  ")
	return os.WriteFile(path, data, 0644)
}
