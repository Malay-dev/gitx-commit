package hook

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// hookMarker identifies scripts managed by gitx-commit.
// Used by Install, Remove, and IsInstalled to distinguish
// our hooks from hooks written by other tools (Husky, lefthook, etc.).
const hookMarker = "# gitx-commit: managed hook"

// Hook scripts — raw strings, no indentation (content is literal).
const prepareCommitMsgScript = `#!/bin/sh
# gitx-commit: managed hook
exec git-xcommit hook prepare-commit-msg "$1" "$2" "$3"
`

const postCommitScript = `#!/bin/sh
# gitx-commit: managed hook
exec git-xcommit hook post-commit
`

// Install writes the prepare-commit-msg and post-commit hooks
// into the repo's .git/hooks/ directory. If an existing hook
// from another tool is found, it returns an error instead of
// silently overwriting it.
func Install(repoPath string) error {
	hooksDir := filepath.Join(repoPath, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("failed to create hooks directory: %w", err)
	}

	hooks := map[string]string{
		"prepare-commit-msg": prepareCommitMsgScript,
		"post-commit":        postCommitScript,
	}

	for name, script := range hooks {
		hookPath := filepath.Join(hooksDir, name)
		if existing, err := os.ReadFile(hookPath); err == nil {
			if !strings.Contains(string(existing), hookMarker) {
				return fmt.Errorf("existing hook found at %s — use --force to overwrite", hookPath)
			}
		}
		if err := os.WriteFile(hookPath, []byte(script), 0755); err != nil {
			return fmt.Errorf("failed to write %s hook: %w", name, err)
		}
	}
	return nil
}

// Remove deletes hooks that belong to gitx-commit.
// It only removes hooks containing our marker — hooks from
// other tools are left untouched.
func Remove(repoPath string) error {
	hooksDir := filepath.Join(repoPath, ".git", "hooks")

	hookNames := []string{"prepare-commit-msg", "post-commit"}

	for _, name := range hookNames {
		hookPath := filepath.Join(hooksDir, name)
		existing, err := os.ReadFile(hookPath)
		if err != nil {
			continue // file doesn't exist — nothing to remove
		}
		if !strings.Contains(string(existing), hookMarker) {
			continue // not our hook — leave it alone
		}
		if err := os.Remove(hookPath); err != nil {
			return fmt.Errorf("failed to remove %s hook: %w", name, err)
		}
	}
	return nil
}

// IsInstalled checks whether gitx-commit hooks are present
// in the repo. It looks for our marker in the prepare-commit-msg hook.
func IsInstalled(repoPath string) bool {
	hookPath := filepath.Join(repoPath, ".git", "hooks", "prepare-commit-msg")
	existing, err := os.ReadFile(hookPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(existing), hookMarker)
}
