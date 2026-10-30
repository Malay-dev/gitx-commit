package cmd

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/Malay-dev/gitx-commit/internal/hook"
	"github.com/Malay-dev/gitx-commit/internal/ui"
)

var disableGlobal bool

var disableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable gitx-commit in the current repository",
	Long: `Removes the prepare-commit-msg and post-commit hooks installed by
gitx-commit. Hooks from other tools are left untouched.

Use --global to remove global hooks and unset core.hooksPath.
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

	if err := hook.Remove(repoPath); err != nil {
		return fmt.Errorf("failed to remove hooks: %w", err)
	}

	ui.Success("gitx-commit disabled in %s", repoPath)
	return nil
}

func disableGlobalHooks() error {
	hooksDir, err := hook.GlobalHooksDir()
	if err != nil {
		return fmt.Errorf("failed to find global hooks directory: %w", err)
	}

	// Remove hooks from global dir
	hook.RemoveFromDir(hooksDir)

	// Unset git config
	// Unset git config — ignore error (may not be set, which is fine)
	exec.Command("git", "config", "--global", "--unset", "core.hooksPath").Run() //nolint: safe to ignore

	ui.Success("gitx-commit disabled globally")
	return nil
}

func init() {
	disableCmd.Flags().BoolVar(&disableGlobal, "global", false, "disable global hooks and unset core.hooksPath")
	rootCmd.AddCommand(disableCmd)
}
