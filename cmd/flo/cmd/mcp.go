package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/richgo/flo/pkg/mcp"
	"github.com/richgo/flo/pkg/tools"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "MCP server commands",
	Long:  "Commands for the MCP (Model Context Protocol) server.",
}

var mcpServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start MCP server on stdio",
	Long: `Start an MCP server that exposes EAS tools to Claude Code.

The server communicates over stdio using JSON-RPC 2.0.
Configure in Claude Code with:

  {
    "mcpServers": {
      "eas": {
        "command": "eas",
        "args": ["mcp", "serve"],
        "cwd": "/path/to/feature"
      }
    }
  }`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load workspace
		ws, err := loadWorkspace()
		if err != nil {
			return err
		}

		// Create tools with workspace context
		toolReg := tools.NewEASTools(ws.Tasks, nil)

		// Add eas_spec_read tool
		toolReg.Register(tools.New(
			"eas_spec_read",
			"Read the feature specification (SPEC.md)",
			map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
			func(args tools.Args) (string, error) {
				return ws.ReadSpec()
			},
		))

		// Start MCP server on stdio
		server := mcp.NewServer(toolReg)
		return server.Serve(os.Stdin, os.Stdout)
	},
}

func init() {
	mcpCmd.AddCommand(mcpServeCmd)
	rootCmd.AddCommand(mcpCmd)
}
