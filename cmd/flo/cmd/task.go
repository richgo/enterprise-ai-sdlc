package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/richgo/flo/pkg/workspace"
)

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage tasks",
	Long:  "Create, list, and manage tasks in the current workspace.",
}

// List flags
var listStatus string
var listRepo string
var listJSON bool

var taskListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := loadWorkspace()
		if err != nil {
			return err
		}

		tasks := ws.ListTasks(listStatus, listRepo)

		if listJSON {
			data, _ := json.MarshalIndent(tasks, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(tasks) == 0 {
			fmt.Println("No tasks found.")
			return nil
		}

		fmt.Printf("Tasks (%d):\n", len(tasks))
		for _, t := range tasks {
			deps := ""
			if len(t.Deps) > 0 {
				deps = fmt.Sprintf(" [deps: %s]", strings.Join(t.Deps, ", "))
			}
			repo := ""
			if t.Repo != "" {
				repo = fmt.Sprintf(" (%s)", t.Repo)
			}
			fmt.Printf("  %s [%s] %s%s%s\n", t.ID, t.Status, t.Title, repo, deps)
		}

		return nil
	},
}

// Create flags
var createRepo string
var createDeps string
var createPriority int

var taskCreateCmd = &cobra.Command{
	Use:   "create <title>",
	Short: "Create a new task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := loadWorkspace()
		if err != nil {
			return err
		}

		title := args[0]
		var deps []string
		if createDeps != "" {
			deps = strings.Split(createDeps, ",")
			for i := range deps {
				deps[i] = strings.TrimSpace(deps[i])
			}
		}

		task, err := ws.CreateTask(title, createRepo, deps, createPriority)
		if err != nil {
			return err
		}

		fmt.Printf("✓ Created task: %s\n", task.ID)
		fmt.Printf("  Title: %s\n", task.Title)
		if task.Repo != "" {
			fmt.Printf("  Repo:  %s\n", task.Repo)
		}
		if len(task.Deps) > 0 {
			fmt.Printf("  Deps:  %s\n", strings.Join(task.Deps, ", "))
		}

		return nil
	},
}

var taskGetCmd = &cobra.Command{
	Use:   "get <task-id>",
	Short: "Get task details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := loadWorkspace()
		if err != nil {
			return err
		}

		task, err := ws.GetTask(args[0])
		if err != nil {
			return err
		}

		data, _ := json.MarshalIndent(task, "", "  ")
		fmt.Println(string(data))

		return nil
	},
}

func init() {
	// List command
	taskListCmd.Flags().StringVar(&listStatus, "status", "", "Filter by status (pending, in_progress, complete, failed)")
	taskListCmd.Flags().StringVar(&listRepo, "repo", "", "Filter by repository")
	taskListCmd.Flags().BoolVar(&listJSON, "json", false, "Output as JSON")

	// Create command
	taskCreateCmd.Flags().StringVar(&createRepo, "repo", "", "Target repository")
	taskCreateCmd.Flags().StringVar(&createDeps, "deps", "", "Comma-separated dependency task IDs")
	taskCreateCmd.Flags().IntVar(&createPriority, "priority", 0, "Task priority (0 = highest)")

	taskCmd.AddCommand(taskListCmd)
	taskCmd.AddCommand(taskCreateCmd)
	taskCmd.AddCommand(taskGetCmd)
}

func loadWorkspace() (*workspace.Workspace, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}
	return workspace.Load(cwd)
}
