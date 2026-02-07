package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/richgo/flo/pkg/agent"
	"github.com/richgo/flo/pkg/task"
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

		// Create backend
		ctx := context.Background()
		
		var backend agent.Backend
		switch backendName {
		case "claude":
			mcpConfig := filepath.Join(ws.Root, ".eas", "mcp.json")
			// Generate MCP config
			if err := generateMCPConfig(mcpConfig, ws.Root); err != nil {
				return fmt.Errorf("failed to generate MCP config: %w", err)
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
			return fmt.Errorf("unknown backend: %s", backendName)
		}

		if err := backend.Start(ctx); err != nil {
			return fmt.Errorf("failed to start backend: %w", err)
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
			return fmt.Errorf("failed to create session: %w", err)
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
