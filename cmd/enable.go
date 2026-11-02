package cmd

import (
	"fmt"
	"os/exec"
	"strings"

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
	Long: `Installs config-based hooks (Git ≥ 2.54) that coexist safely with
Husky, pre-commit, lefthook, and any other hook manager.

Hooks are registered via git config (hook.<name>.command) — no files
are written to .git/hooks/, so there is never a conflict.

Use --global to enable hooks for all repositories.
Use --force to re-install hooks (e.g. after corruption).
Safe to run multiple times — re-running updates hooks in place.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return enableGitXCommit()
	},
}

func enableGitXCommit() error {
	// Gate on Git version
	if err := requireConfigHooks(); err != nil {
		return err
	}

	if enableGlobal {
		return enableGlobalHooks()
	}

	repoPath, err := requireRepoPath()
	if err != nil {
		return err
	}

	if enableForce {
		// Force: remove existing config hooks and legacy hooks, then re-install.
		hook.RemoveConfig(repoPath, false)
		hook.Remove(repoPath)
	}

	// Migrate: clean up v0.1 legacy hooks from .git/hooks/ if present.
	if hook.IsInstalled(repoPath) {
		hook.Remove(repoPath)
		ui.Info("Migrated from legacy hooks to config hooks")
	}

	if err := hook.InstallConfig(repoPath, false); err != nil {
		return fmt.Errorf("failed to install hooks: %w", err)
	}

	ui.Success("gitx-commit enabled in %s", repoPath)
	return nil
}

// enableGlobalHooks registers config-based hooks at the global scope.
// Never touches core.hooksPath. Migrates old-style global installs.
func enableGlobalHooks() error {
	// Migrate old-style global install (core.hooksPath = ~/.gitx/hooks).
	migrateOldGlobalInstall()

	if err := hook.InstallConfig(".", true); err != nil {
		return fmt.Errorf("failed to install global hooks: %w", err)
	}

	ui.Success("gitx-commit enabled globally (config-based hooks)")
	return nil
}

// migrateOldGlobalInstall checks for the v0.1 global install that used
// core.hooksPath = ~/.gitx/hooks, and removes it. Only touches core.hooksPath
// if it points to our directory — never overwrites another tool's setting.
func migrateOldGlobalInstall() {
	out, err := exec.Command("git", "config", "--global", "core.hooksPath").Output()
	if err != nil {
		return // not set — nothing to migrate
	}

	existing := strings.TrimSpace(string(out))
	globalDir, err := hook.GlobalHooksDir()
	if err != nil {
		return
	}

	if existing == globalDir {
		// It's ours — clean up.
		hook.RemoveFromDir(globalDir)
		exec.Command("git", "config", "--global", "--unset", "core.hooksPath").Run()
		ui.Info("Migrated from core.hooksPath to config hooks")
	}
}

// requireConfigHooks checks Git version and returns a clear error if too old.
func requireConfigHooks() error {
	major, minor, err := hook.GitVersion()
	if err != nil {
		return fmt.Errorf("cannot detect Git version: %w", err)
	}
	if !hook.SupportsConfigHooks() {
		return fmt.Errorf(
			"Git %d.%d is too old — gitx-commit requires Git ≥ 2.54 for config-based hooks\n"+
				"  Upgrade Git: https://git-scm.com/downloads",
			major, minor,
		)
	}
	return nil
}

func init() {
	enableCmd.Flags().BoolVar(&enableGlobal, "global", false, "enable for all repositories via global config hooks")
	enableCmd.Flags().BoolVar(&enableForce, "force", false, "re-install hooks (e.g. after corruption)")
	rootCmd.AddCommand(enableCmd)
}
