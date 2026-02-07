package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/richgo/flo/pkg/workspace"
)

var initBackend string

var initCmd = &cobra.Command{
	Use:   "init <feature-name>",
	Short: "Initialize a new feature workspace",
	Long: `Initialize a new EAS feature workspace in the current directory.

Creates:
  .eas/config.yaml    - Feature configuration
  .eas/SPEC.md        - Feature specification template
  .eas/tasks/         - Task manifest directory`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		featureName := args[0]
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		ws, err := workspace.Init(cwd, featureName, initBackend)
		if err != nil {
			return err
		}

		fmt.Printf("✓ Initialized workspace for feature: %s\n", ws.Feature)
		fmt.Printf("  Backend: %s\n", ws.Backend)
		fmt.Printf("  Config:  .eas/config.yaml\n")
		fmt.Printf("  Spec:    .eas/SPEC.md\n")
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("  1. Edit .eas/SPEC.md with your feature specification")
		fmt.Println("  2. Create tasks: eas task create \"Task title\"")
		fmt.Println("  3. Check status: eas status")

		return nil
	},
}

func init() {
	initCmd.Flags().StringVar(&initBackend, "backend", "claude", "Agent backend (claude or copilot)")
}
