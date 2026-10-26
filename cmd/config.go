package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Malay-dev/gitx-commit/internal/config"
	"github.com/Malay-dev/gitx-commit/internal/ui"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View and manage gitx-commit configuration",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long: `Displays the current gitx-commit configuration from all sources:
global (~/.gitx/commit-config.toml) and repo-level (.gitx-commit.toml).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return showConfig()
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show configuration file paths",
	RunE: func(cmd *cobra.Command, args []string) error {
		return showConfigPaths()
	},
}

func showConfig() error {
	cfg := loadConfig()

	fmt.Println()
	fmt.Println("  Current Configuration:")
	fmt.Println()

	// General
	ui.Field("Enabled", fmt.Sprintf("%v", cfg.General.Enabled))

	// Template
	fmt.Println()
	ui.Field("Template format", cfg.Template.Format)
	if cfg.Template.CustomTemplate != "" {
		ui.Field("Custom template", cfg.Template.CustomTemplate)
	}
	ui.Field("Trailer format", cfg.Template.Trailer.Format)
	ui.Field("Include model", fmt.Sprintf("%v", cfg.Template.Trailer.IncludeModel))

	// AI
	fmt.Println()
	ui.Field("AI enabled", fmt.Sprintf("%v", cfg.AI.Enabled))
	ui.Field("AI mode", cfg.AI.Mode)
	ui.Field("Confidence", fmt.Sprintf("%.2f", cfg.AI.ConfidenceThreshold))
	if cfg.AI.OverrideTool != "" {
		ui.Field("Override tool", cfg.AI.OverrideTool)
	}

	// Journal
	fmt.Println()
	ui.Field("Journal enabled", fmt.Sprintf("%v", cfg.Journal.Enabled))
	ui.Field("Journal path", cfg.Journal.DBPath)

	fmt.Println()
	return nil
}

func showConfigPaths() error {
	globalPath, err := config.GlobalConfigPath()
	if err != nil {
		return err
	}

	fmt.Println()
	// Global
	exists := "exists"
	if _, err := os.Stat(globalPath); err != nil {
		exists = "not created yet"
	}
	ui.Field("Global config", fmt.Sprintf("%s (%s)", globalPath, exists))

	// Repo
	repoPath := config.RepoConfigPath()
	exists = "exists"
	if _, err := os.Stat(repoPath); err != nil {
		exists = "not created"
	}
	ui.Field("Repo config", fmt.Sprintf("%s (%s)", repoPath, exists))

	fmt.Println()
	return nil
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configPathCmd)
	rootCmd.AddCommand(configCmd)
}
