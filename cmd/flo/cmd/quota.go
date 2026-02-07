package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/richgo/flo/pkg/quota"
	"github.com/spf13/cobra"
)

var quotaCmd = &cobra.Command{
	Use:   "quota",
	Short: "Show backend usage and quota status",
	Long: `Display usage statistics for each AI backend including requests,
tokens consumed, and remaining quota.`,
	RunE: runQuota,
}

func init() {
	rootCmd.AddCommand(quotaCmd)
}

func runQuota(cmd *cobra.Command, args []string) error {
	// Get quota file path from .flo directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	
	quotaPath := filepath.Join(homeDir, ".flo", "quota.json")
	tracker := quota.New(quotaPath)
	
	// Load existing quota data
	if err := tracker.Load(); err != nil {
		return fmt.Errorf("failed to load quota data: %w", err)
	}
	
	// Get all usage data
	allUsage := tracker.ListUsage()
	
	if len(allUsage) == 0 {
		fmt.Println("No usage data recorded yet.")
		return nil
	}
	
	// Create table writer
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	defer w.Flush()
	
	fmt.Fprintln(w, "BACKEND\tREQUESTS\tTOKENS\tSTATUS\tLAST REQUEST\tWINDOW")
	fmt.Fprintln(w, "-------\t--------\t------\t------\t------------\t------")
	
	for backend, usage := range allUsage {
		status := "✓ OK"
		if usage.IsExhausted {
			status = fmt.Sprintf("✗ EXHAUSTED (retry after %s)", 
				formatDuration(time.Until(usage.RetryAfter)))
		}
		
		lastReq := "never"
		if !usage.LastRequest.IsZero() {
			lastReq = formatRelativeTime(usage.LastRequest)
		}
		
		windowAge := formatDuration(time.Since(usage.WindowStart))
		
		fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%s\t%s\n",
			backend,
			usage.Requests,
			usage.Tokens,
			status,
			lastReq,
			windowAge,
		)
	}
	
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Use 'flo config' to set backend limits and quotas.")
	
	return nil
}

func formatRelativeTime(t time.Time) string {
	dur := time.Since(t)
	
	if dur < time.Minute {
		return "just now"
	}
	if dur < time.Hour {
		mins := int(dur.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	}
	if dur < 24*time.Hour {
		hours := int(dur.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	}
	
	days := int(dur.Hours() / 24)
	if days == 1 {
		return "1 day ago"
	}
	return fmt.Sprintf("%d days ago", days)
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		return "expired"
	}
	
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%.1fh", d.Hours())
	}
	
	days := d.Hours() / 24
	return fmt.Sprintf("%.1fd", days)
}
