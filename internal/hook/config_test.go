package hook

import (
	"os/exec"
	"testing"
)

func TestParseGitVersion(t *testing.T) {
	cases := []struct {
		in         string
		major, min int
		ok         bool
	}{
		{"git version 2.55.0\n", 2, 55, true},
		{"git version 2.39.5 (Apple Git-154)", 2, 39, true},
		{"garbage", 0, 0, false},
	}
	for _, c := range cases {
		maj, mi, err := parseGitVersion(c.in)
		if (err == nil) != c.ok || maj != c.major || mi != c.min {
			t.Errorf("parseGitVersion(%q) = %d,%d,%v", c.in, maj, mi, err)
		}
	}
}

func TestVersionSupportsConfigHooks(t *testing.T) {
	cases := []struct {
		major, minor int
		want         bool
	}{
		{2, 53, false}, {2, 54, true}, {2, 55, true}, {3, 0, true}, {1, 99, false},
	}
	for _, c := range cases {
		if got := versionSupportsConfigHooks(c.major, c.minor); got != c.want {
			t.Errorf("%d.%d = %v, want %v", c.major, c.minor, got, c.want)
		}
	}
}

func TestConfigInstallRemoveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := exec.Command("git", "init", dir).Run(); err != nil {
		t.Skip("git not available")
	}
	if err := InstallConfig(dir, false); err != nil {
		t.Fatalf("InstallConfig: %v", err)
	}
	if !IsConfigInstalled(dir, false) {
		t.Fatal("expected installed")
	}
	// Idempotent: second install must not duplicate events.
	if err := InstallConfig(dir, false); err != nil {
		t.Fatalf("second InstallConfig: %v", err)
	}
	out, _ := exec.Command("git", "-C", dir, "config", "--local", "--get-all", "hook.gitx-commit-prepare.event").Output()
	if string(out) != "prepare-commit-msg\n" {
		t.Errorf("duplicate events: %q", out)
	}
	if err := RemoveConfig(dir, false); err != nil {
		t.Fatalf("RemoveConfig: %v", err)
	}
	if IsConfigInstalled(dir, false) {
		t.Fatal("expected removed")
	}
	// Removing when absent is not an error.
	if err := RemoveConfig(dir, false); err != nil {
		t.Fatalf("RemoveConfig on empty: %v", err)
	}
}
