package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Malay-dev/gitx-commit/internal/config"
)

var (
	buildVersion = "dev"
	buildCommit  = "none"
	buildDate    = "unknown"
)

// SetVersionInfo is called from main() to inject build-time variables.
func SetVersionInfo(version, commit, date string) {
	buildVersion = version
	buildCommit = commit
	buildDate = date
}

var rootCmd = &cobra.Command{
	Use:   "git-xcommit",
	Short: "Intelligent commit companion — AI provenance, templates, and commit journal for Git",
	Long: `git-xcommit augments git commit with AI provenance tracking,
commit message templating, and a local commit journal.

Once enabled, it hooks into your normal git commit workflow and
automatically detects AI tools, applies message templates, and
logs every commit to a searchable local journal.

Part of the gitx ecosystem (https://github.com/Malay-dev).

Features:
  • Template-based commit message formatting
  • Automatic AI tool detection (Kiro, Claude, Copilot, Cursor, Gemini, ...)
  • Commit journal across all your repositories
  • AI usage analytics and year-end reporting

Examples:
  git xcommit enable     Enable in the current repository
  git xcommit status     Show what's active and detected
  git xcommit detect     Show all detected AI tools
  git xcommit disable    Remove hooks and disable`,
}

func init() {
	rootCmd.Version = buildVersion
	rootCmd.SetVersionTemplate(fmt.Sprintf(
		"gitx-commit %s (commit: %s, built: %s)\n",
		buildVersion, buildCommit, buildDate,
	))
}

// Execute is the single entry point called from main().
func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}

// loadConfig loads config with fallback to defaults.
// Error is intentionally ignored — Load() returns defaults on any failure
// (missing file, bad TOML, etc.). The tool must always work without a config file.
func loadConfig() *config.Config {
	cfg, _ := config.Load()
	return cfg
}
