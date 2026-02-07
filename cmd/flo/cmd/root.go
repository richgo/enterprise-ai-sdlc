package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "flo",
	Short: "Flo - Engineer Flow for AI-powered development",
	Long: `Flo enables engineer flow by orchestrating AI agents for spec-driven,
test-driven development.

Create tasks, define specs, and let AI agents implement them while
you stay in the zone.`,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(taskCmd)
	rootCmd.AddCommand(statusCmd)
}
