package cmd

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/Malay-dev/gitx-commit/internal/hook"
	"github.com/Malay-dev/gitx-commit/internal/ui"
)

var (
	enableGlobal bool
	enableForce  bool
)

var enableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable gitx-commit in the current repository",
	Long: `Installs prepare-commit-msg and post-commit hooks into the current
repository's .git/hooks/ directory. Existing hooks from other tools
(Husky, lefthook, etc.) are never overwritten unless --force is used.

Use --global to install hooks for all repositories via git config.
Safe to run multiple times — re-running updates the hooks in place.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return enableGitXCommit()
	},
}

func enableGitXCommit() error {
	if enableGlobal {
		return enableGlobalHooks()
	}

	repoPath, err := requireRepoPath()
	if err != nil {
		return err
	}

	if enableForce {
		// Force mode — remove existing hooks first
		hook.Remove(repoPath)
	}

	if err := hook.Install(repoPath); err != nil {
		return fmt.Errorf("failed to install hooks: %w", err)
	}

	ui.Success("gitx-commit enabled in %s", repoPath)
	return nil
}

// enableGlobalHooks sets up a global hooks directory that applies to all repos.
// Uses git config --global core.hooksPath to point to ~/.gitx/hooks/
func enableGlobalHooks() error {
	hooksDir, err := hook.GlobalHooksDir()
	if err != nil {
		return fmt.Errorf("failed to create global hooks directory: %w", err)
	}

	if err := hook.InstallGlobal(hooksDir); err != nil {
		return fmt.Errorf("failed to install global hooks: %w", err)
	}

	// Set git config
	if err := exec.Command("git", "config", "--global", "core.hooksPath", hooksDir).Run(); err != nil {
		return fmt.Errorf("failed to set core.hooksPath: %w", err)
	}

	ui.Success("gitx-commit enabled globally (hooks in %s)", hooksDir)
	return nil
}

func init() {
	enableCmd.Flags().BoolVar(&enableGlobal, "global", false, "enable for all repositories via global hooks")
	enableCmd.Flags().BoolVar(&enableForce, "force", false, "overwrite existing hooks from other tools")
	rootCmd.AddCommand(enableCmd)
}
