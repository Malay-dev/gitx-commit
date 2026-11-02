package hook

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Config-based hooks (hook.<name>.command / hook.<name>.event) were added
// in Git 2.54. They run even when core.hooksPath is set (Husky etc.) and
// coexist with hookdir hooks, so they are the preferred install method.
const (
	minConfigHookMajor = 2
	minConfigHookMinor = 54
)

// configHooks maps our config hook name to its event and command.
// Names must not equal a Git event name. Git appends the hook args to the command.
var configHooks = []struct {
	name    string
	event   string
	command string
}{
	{"gitx-commit-prepare", "prepare-commit-msg", "git-xcommit hook prepare-commit-msg"},
	{"gitx-commit-post", "post-commit", "git-xcommit hook post-commit"},
}

var versionRe = regexp.MustCompile(`(\d+)\.(\d+)`)

// parseGitVersion extracts major and minor from `git --version` output.
func parseGitVersion(out string) (int, int, error) {
	m := versionRe.FindStringSubmatch(out)
	if m == nil {
		return 0, 0, fmt.Errorf("cannot parse git version from %q", strings.TrimSpace(out))
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	return major, minor, nil
}

// versionSupportsConfigHooks reports whether major.minor is >= 2.54.
func versionSupportsConfigHooks(major, minor int) bool {
	if major != minConfigHookMajor {
		return major > minConfigHookMajor
	}
	return minor >= minConfigHookMinor
}

// SupportsConfigHooks reports whether the installed Git supports config-based hooks.
// Returns false if Git cannot be queried.
func SupportsConfigHooks() bool {
	major, minor, err := GitVersion()
	if err != nil {
		return false
	}
	return versionSupportsConfigHooks(major, minor)
}

// GitVersion returns the installed Git major and minor version.
func GitVersion() (int, int, error) {
	out, err := exec.Command("git", "--version").Output()
	if err != nil {
		return 0, 0, fmt.Errorf("git not found: %w", err)
	}
	return parseGitVersion(string(out))
}

// scopeFlag returns the git config scope flag.
func scopeFlag(global bool) string {
	if global {
		return "--global"
	}
	return "--local"
}

// InstallConfig registers our hooks via git config. It is idempotent:
// existing entries for our hook names are replaced.
func InstallConfig(repoPath string, global bool) error {
	// Remove first so re-running never duplicates event entries.
	if err := RemoveConfig(repoPath, global); err != nil {
		return err
	}
	for _, h := range configHooks {
		if err := gitConfig(repoPath, global, "hook."+h.name+".command", h.command); err != nil {
			_ = RemoveConfig(repoPath, global) // roll back: all or nothing
			return err
		}
		if err := gitConfig(repoPath, global, "--add", "hook."+h.name+".event", h.event); err != nil {
			_ = RemoveConfig(repoPath, global)
			return err
		}
	}
	return nil
}

// RemoveConfig unsets our config-based hooks. Missing keys are not an error.
func RemoveConfig(repoPath string, global bool) error {
	for _, h := range configHooks {
		for _, key := range []string{"command", "event"} {
			args := []string{"config", scopeFlag(global), "--unset-all", "hook." + h.name + "." + key}
			cmd := exec.Command("git", args...)
			cmd.Dir = repoPath
			if err := cmd.Run(); err != nil {
				// Exit code 5 means the key was not set.
				if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 5 {
					continue
				}
				return fmt.Errorf("failed to unset hook.%s.%s: %w", h.name, key, err)
			}
		}
	}
	return nil
}

// IsConfigInstalled reports whether our config-based hooks are registered
// at the given scope.
func IsConfigInstalled(repoPath string, global bool) bool {
	cmd := exec.Command("git", "config", scopeFlag(global), "--get", "hook."+configHooks[0].name+".command")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(out)) != ""
}

func gitConfig(repoPath string, global bool, args ...string) error {
	full := append([]string{"config", scopeFlag(global)}, args...)
	cmd := exec.Command("git", full...)
	cmd.Dir = repoPath
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git config %s failed: %s", strings.Join(args, " "), strings.TrimSpace(string(out)))
	}
	return nil
}
