package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Malay-dev/gitx-commit/internal/hook"
	"github.com/Malay-dev/gitx-commit/internal/ui"
)

var disableGlobal bool

var disableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable gitx-commit in the current repository",
	Long: `Removes config-based hooks registered by gitx-commit.
Also cleans up any legacy hooks from .git/hooks/ (v0.1 installs).
Hooks from other tools are never touched.

Use --global to remove global hooks.
Safe to run even if gitx-commit was never enabled.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return disableGitXCommit()
	},
}

func disableGitXCommit() error {
	if disableGlobal {
		return disableGlobalHooks()
	}

	repoPath, err := requireRepoPath()
	if err != nil {
		return err
	}

	// Remove config-based hooks.
	if err := hook.RemoveConfig(repoPath, false); err != nil {
		return fmt.Errorf("failed to remove config hooks: %w", err)
	}

	// Also clean up legacy hooks from .git/hooks/ (v0.1 installs).
	if err := hook.Remove(repoPath); err != nil {
		return fmt.Errorf("failed to remove legacy hooks: %w", err)
	}

	ui.Success("gitx-commit disabled in %s", repoPath)
	return nil
}

func disableGlobalHooks() error {
	// Remove config-based global hooks.
	if err := hook.RemoveConfig(".", true); err != nil {
		return fmt.Errorf("failed to remove global config hooks: %w", err)
	}

	// Clean up old-style global install if present.
	cleanupOldGlobalInstall()

	ui.Success("gitx-commit disabled globally")
	return nil
}

// cleanupOldGlobalInstall removes the v0.1 global install (core.hooksPath + scripts).
// Only unsets core.hooksPath if it points to our ~/.gitx/hooks/ directory.
func cleanupOldGlobalInstall() {
	out, err := exec.Command("git", "config", "--global", "core.hooksPath").Output()
	if err != nil {
		return // not set
	}

	existing := strings.TrimSpace(string(out))
	globalDir, err := hook.GlobalHooksDir()
	if err != nil {
		return
	}

	if existing == globalDir {
		hook.RemoveFromDir(globalDir)
		exec.Command("git", "config", "--global", "--unset", "core.hooksPath").Run()
	}
}

func init() {
	disableCmd.Flags().BoolVar(&disableGlobal, "global", false, "disable global hooks")
	rootCmd.AddCommand(disableCmd)
}
