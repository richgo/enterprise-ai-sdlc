package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show workspace status",
	Long:  "Display an overview of the current feature workspace.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := loadWorkspace()
		if err != nil {
			return err
		}

		status := ws.Status()

		fmt.Printf("Feature: %s\n", status.Feature)
		fmt.Printf("Backend: %s\n", status.Backend)
		fmt.Println()
		fmt.Printf("Tasks: %d total\n", status.TotalTasks)
		fmt.Printf("  📋 Pending:     %d\n", status.PendingTasks)
		fmt.Printf("  🔄 In Progress: %d\n", status.InProgressTasks)
		fmt.Printf("  ✅ Complete:    %d\n", status.CompleteTasks)
		fmt.Printf("  ❌ Failed:      %d\n", status.FailedTasks)
		fmt.Println()
		fmt.Printf("Ready to start: %d\n", status.ReadyTasks)

		if status.ReadyTasks > 0 {
			fmt.Println()
			fmt.Println("Ready tasks:")
			for _, t := range ws.GetReadyTasks() {
				fmt.Printf("  %s: %s\n", t.ID, t.Title)
			}
		}

		return nil
	},
}
