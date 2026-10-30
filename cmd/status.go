package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Malay-dev/gitx-commit/internal/detect"
	"github.com/Malay-dev/gitx-commit/internal/hook"
	"github.com/Malay-dev/gitx-commit/internal/ui"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current gitx-commit configuration and state",
	Long: `Displays a dashboard of the current gitx-commit configuration,
hook installation status, detected AI tools, and journal settings.

Works both inside and outside a git repository.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return showStatus()
	},
}

func showStatus() error {
	repoPath := getRepoPath()

	cfg := loadConfig()

	// Header
	fmt.Println()
	if cfg.General.Enabled {
		ui.Success("gitx-commit is active")
	} else {
		ui.Error("gitx-commit is inactive")
	}
	fmt.Println()

	// Hooks
	if repoPath != "" && hook.IsInstalled(repoPath) {
		ui.Field("Hooks", "installed")
	} else {
		ui.Field("Hooks", "not installed")
	}

	// Config
	ui.Field("Template", cfg.Template.Format)
	ui.Field("AI Detect", fmt.Sprintf("%s (confidence ≥ %.2f)", cfg.AI.Mode, cfg.AI.ConfidenceThreshold))

	trailerStr := cfg.Template.Trailer.Format
	if cfg.Template.Trailer.IncludeModel {
		trailerStr += " (tool + model)"
	} else {
		trailerStr += " (tool only)"
	}
	ui.Field("Trailer", trailerStr)
	ui.Field("Journal", cfg.Journal.DBPath)

	// Detection
	fmt.Println()
	if repoPath != "" {
		result := detect.Detect(repoPath, nil)
		if result != nil {
			detectedStr := result.Tool
			if result.Version != "" {
				detectedStr += "/" + result.Version
			}
			if result.Model != "" {
				detectedStr += " (" + result.Model + ")"
			}
			detectedStr += fmt.Sprintf(" [%s, confidence: %.2f]", result.Method, result.Confidence)
			ui.Field("Detected", detectedStr)
		} else {
			ui.Field("Detected", "none")
		}
	} else {
		ui.Field("Detected", "n/a (not in a git repo)")
	}

	// Repository
	fmt.Println()
	if repoPath != "" {
		ui.Field("Repository", repoPath)
	} else {
		ui.Field("Repository", "(not in a git repo)")
	}
	fmt.Println()

	return nil
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
