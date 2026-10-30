package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Malay-dev/gitx-commit/internal/detect"
	"github.com/Malay-dev/gitx-commit/internal/ui"
)

var detectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Show all AI coding tools detected in the current environment",
	Long:  "Scans environment variables, repo config files, and home directory for AI coding tool artifacts.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return showDetected()
	},
}

func showDetected() error {
	repoPath := getRepoPath()

	results := detect.DetectAll(repoPath)

	fmt.Println()
	fmt.Println("  AI Tools Detected:")
	fmt.Println()

	if len(results) == 0 {
		fmt.Println("  No AI tools detected.")
	} else {
		for _, r := range results {
			name := r.Tool
			if r.Model != "" {
				name += " (" + r.Model + ")"
			}
			fmt.Printf("  %-36s %-12s confidence: %.2f\n", name, r.Method, r.Confidence)
		}
	}

	fmt.Println()
	override := os.Getenv("GITX_AI_TOOL")
	if override != "" {
		ui.Field("Override", "GITX_AI_TOOL="+override)
	} else {
		ui.Field("Override", "GITX_AI_TOOL is not set")
	}

	if repoPath != "" {
		ui.Field("Repository", repoPath)
	} else {
		ui.Field("Repository", "(not in a git repo)")
	}
	fmt.Println()

	return nil
}

func init() {
	rootCmd.AddCommand(detectCmd)
}
