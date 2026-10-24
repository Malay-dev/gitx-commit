package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/Malay-dev/gitx-commit/internal/hook"
	"github.com/Malay-dev/gitx-commit/internal/journal"
)

// setupGitRepo creates a real temporary git repo for integration tests.
func setupGitRepo(t *testing.T) string {
	t.Helper()
	repoPath := t.TempDir()

	// git init
	cmd := exec.Command("git", "init")
	cmd.Dir = repoPath
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, out)
	}

	return repoPath
}

// runCmdInDir runs a cobra command in a specific directory.
// We change the working directory so "git rev-parse" finds the repo.
func runCmdInDir(t *testing.T, dir string) {
	t.Helper()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		os.Chdir(originalDir)
	})
}

// captureOutput runs a function and captures both stdout and stderr.
func captureOutput(t *testing.T, fn func()) string {
	t.Helper()

	// Capture stdout
	oldOut := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	// Capture stderr
	oldErr := os.Stderr
	rErr, wErr, _ := os.Pipe()
	os.Stderr = wErr

	fn()

	wOut.Close()
	wErr.Close()
	os.Stdout = oldOut
	os.Stderr = oldErr

	var bufOut, bufErr bytes.Buffer
	bufOut.ReadFrom(rOut)
	bufErr.ReadFrom(rErr)

	return bufOut.String() + bufErr.String()
}

// --- Enable tests ---

func TestEnableInGitRepo(t *testing.T) {
	repoPath := setupGitRepo(t)
	runCmdInDir(t, repoPath)

	err := enableGitXCommit()

	if err != nil {
		t.Fatalf("enableGitXCommit() failed: %v", err)
	}

	// Verify hooks were installed
	if !hook.IsInstalled(repoPath) {
		t.Error("hooks should be installed after enable")
	}
}

func TestEnableNotGitRepo(t *testing.T) {
	dir := t.TempDir() // not a git repo
	runCmdInDir(t, dir)

	err := enableGitXCommit()

	if err == nil {
		t.Fatal("enableGitXCommit() should fail outside a git repo")
	}
}

func TestEnableTwice(t *testing.T) {
	repoPath := setupGitRepo(t)
	runCmdInDir(t, repoPath)

	// First enable
	if err := enableGitXCommit(); err != nil {
		t.Fatalf("first enable failed: %v", err)
	}

	// Second enable — should succeed (idempotent)
	if err := enableGitXCommit(); err != nil {
		t.Fatalf("second enable should be idempotent, got: %v", err)
	}
}

// --- Disable tests ---

func TestDisableInGitRepo(t *testing.T) {
	repoPath := setupGitRepo(t)
	runCmdInDir(t, repoPath)

	// Enable first
	enableGitXCommit()

	// Disable
	err := disableGitXCommit()

	if err != nil {
		t.Fatalf("disableGitXCommit() failed: %v", err)
	}

	// Verify hooks were removed
	if hook.IsInstalled(repoPath) {
		t.Error("hooks should be removed after disable")
	}
}

func TestDisableNotGitRepo(t *testing.T) {
	dir := t.TempDir()
	runCmdInDir(t, dir)

	err := disableGitXCommit()

	if err == nil {
		t.Fatal("disableGitXCommit() should fail outside a git repo")
	}
}

func TestDisableWhenNotEnabled(t *testing.T) {
	repoPath := setupGitRepo(t)
	runCmdInDir(t, repoPath)

	// Disable without enabling first — should succeed (nothing to remove)
	err := disableGitXCommit()

	if err != nil {
		t.Fatalf("disable on clean repo should succeed, got: %v", err)
	}
}

// --- Status tests ---

func TestStatusInGitRepo(t *testing.T) {
	repoPath := setupGitRepo(t)
	runCmdInDir(t, repoPath)

	output := captureOutput(t, func() {
		showStatus()
	})

	// Should contain key sections
	checks := []string{
		"gitx-commit",
		"Hooks",
		"Template",
		"AI Detect",
		"Trailer",
		"Journal",
		"Repository",
	}
	for _, check := range checks {
		if !bytes.Contains([]byte(output), []byte(check)) {
			t.Errorf("status output missing '%s'", check)
		}
	}
}

func TestStatusShowsHooksInstalled(t *testing.T) {
	repoPath := setupGitRepo(t)
	runCmdInDir(t, repoPath)

	// Before enable
	output := captureOutput(t, func() {
		showStatus()
	})
	if !bytes.Contains([]byte(output), []byte("not installed")) {
		t.Error("status should show hooks not installed before enable")
	}

	// After enable
	enableGitXCommit()
	output = captureOutput(t, func() {
		showStatus()
	})
	if !bytes.Contains([]byte(output), []byte("installed")) || bytes.Contains([]byte(output), []byte("not installed")) {
		t.Error("status should show hooks installed after enable")
	}
}

func TestStatusOutsideGitRepo(t *testing.T) {
	dir := t.TempDir()
	runCmdInDir(t, dir)

	output := captureOutput(t, func() {
		showStatus()
	})

	if !bytes.Contains([]byte(output), []byte("not in a git repo")) {
		t.Error("status should indicate not in a git repo")
	}
}

// --- Root command tests ---

func TestRootHelpContainsAllCommands(t *testing.T) {
	// Reset command output to capture help
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"--help"})

	rootCmd.Execute()

	output := buf.String()
	commands := []string{"enable", "disable", "status"}
	for _, cmd := range commands {
		if !bytes.Contains([]byte(output), []byte(cmd)) {
			t.Errorf("help output missing command '%s'", cmd)
		}
	}
}

func TestSetVersionInfo(t *testing.T) {
	SetVersionInfo("1.2.3", "abc123", "2026-10-21")

	if buildVersion != "1.2.3" {
		t.Errorf("expected version '1.2.3', got '%s'", buildVersion)
	}
	if buildCommit != "abc123" {
		t.Errorf("expected commit 'abc123', got '%s'", buildCommit)
	}
	if buildDate != "2026-10-21" {
		t.Errorf("expected date '2026-10-21', got '%s'", buildDate)
	}
}

// --- Edge case: enable then disable then status ---

func TestFullLifecycle(t *testing.T) {
	repoPath := setupGitRepo(t)
	runCmdInDir(t, repoPath)

	// Enable
	if err := enableGitXCommit(); err != nil {
		t.Fatalf("enable failed: %v", err)
	}
	if !hook.IsInstalled(repoPath) {
		t.Fatal("hooks should be installed")
	}

	// Status should show installed
	output := captureOutput(t, func() { showStatus() })
	if !bytes.Contains([]byte(output), []byte("installed")) || bytes.Contains([]byte(output), []byte("not installed")) {
		t.Error("status should show installed")
	}

	// Disable
	if err := disableGitXCommit(); err != nil {
		t.Fatalf("disable failed: %v", err)
	}
	if hook.IsInstalled(repoPath) {
		t.Fatal("hooks should be removed")
	}

	// Status should show not installed
	output = captureOutput(t, func() { showStatus() })
	if !bytes.Contains([]byte(output), []byte("not installed")) {
		t.Error("status should show not installed")
	}

	// Re-enable should work
	if err := enableGitXCommit(); err != nil {
		t.Fatalf("re-enable failed: %v", err)
	}
	if !hook.IsInstalled(repoPath) {
		t.Fatal("hooks should be installed after re-enable")
	}
}

// --- Detect tests ---

func TestDetectInGitRepo(t *testing.T) {
	repoPath := setupGitRepo(t)
	runCmdInDir(t, repoPath)

	output := captureOutput(t, func() {
		showDetected()
	})

	if !bytes.Contains([]byte(output), []byte("AI Tools Detected")) {
		t.Error("detect output missing header")
	}
	if !bytes.Contains([]byte(output), []byte("Override:")) {
		t.Error("detect output missing override status")
	}
}

func TestDetectOutsideGitRepo(t *testing.T) {
	dir := t.TempDir()
	runCmdInDir(t, dir)

	output := captureOutput(t, func() {
		showDetected()
	})

	if !bytes.Contains([]byte(output), []byte("AI Tools Detected")) {
		t.Error("detect should work outside a git repo")
	}
	if !bytes.Contains([]byte(output), []byte("not in a git repo")) {
		t.Error("detect should indicate not in a git repo")
	}
}

func TestRootHelpContainsDetect(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"--help"})

	rootCmd.Execute()

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("detect")) {
		t.Error("help output missing 'detect' command")
	}
}

// --- parseShortStat tests ---

func TestParseShortStat(t *testing.T) {
	tests := []struct {
		input string
		files int
		adds  int
		dels  int
	}{
		{"3 files changed, 50 insertions(+), 10 deletions(-)", 3, 50, 10},
		{"1 file changed, 5 insertions(+)", 1, 5, 0},
		{"1 file changed, 3 deletions(-)", 1, 0, 3},
		{"10 files changed, 200 insertions(+), 100 deletions(-)", 10, 200, 100},
		{"", 0, 0, 0},
	}

	for _, tt := range tests {
		var r journal.CommitRecord
		parseShortStat(tt.input, &r)
		if r.FilesChanged != tt.files {
			t.Errorf("input %q: files expected %d, got %d", tt.input, tt.files, r.FilesChanged)
		}
		if r.Additions != tt.adds {
			t.Errorf("input %q: additions expected %d, got %d", tt.input, tt.adds, r.Additions)
		}
		if r.Deletions != tt.dels {
			t.Errorf("input %q: deletions expected %d, got %d", tt.input, tt.dels, r.Deletions)
		}
	}
}

// --- resolveDate tests ---

func TestResolveDate(t *testing.T) {
	// Empty input
	if resolveDate("") != "" {
		t.Error("empty should return empty")
	}

	// Specific date
	result := resolveDate("2026-06-15")
	if result == "" {
		t.Error("should parse YYYY-MM-DD")
	}

	// Natural language — just verify they return non-empty
	naturals := []string{"today", "yesterday", "last week", "last month", "last year"}
	for _, n := range naturals {
		if resolveDate(n) == "" {
			t.Errorf("resolveDate(%q) should return a date", n)
		}
	}

	// RFC3339 passthrough
	rfc := "2026-10-21T12:00:00Z"
	if resolveDate(rfc) != rfc {
		t.Errorf("RFC3339 should pass through, got %q", resolveDate(rfc))
	}

	// Unknown string passes through
	if resolveDate("gibberish") != "gibberish" {
		t.Error("unknown input should pass through")
	}
}

// --- Export tests ---

func TestExportJSON(t *testing.T) {
	repoPath := setupGitRepo(t)
	runCmdInDir(t, repoPath)

	// Create a journal with test data
	tmpDB := filepath.Join(t.TempDir(), "test.db")
	t.Setenv("GITX_JOURNAL_PATH", tmpDB)

	j, _ := journal.Open(tmpDB)
	j.LogCommit(journal.CommitRecord{
		SHA: "export-test", Repo: repoPath, RepoName: "test",
		Branch: "main", Message: "feat: test", Author: "Test", Email: "test@test.com",
		CommitMode: "Manual", CommittedAt: time.Now(),
	})
	j.Close()

	outFile := filepath.Join(t.TempDir(), "out.json")
	exportFormat = "json"
	exportOutput = outFile
	err := runExport()
	if err != nil {
		t.Fatalf("runExport(json) failed: %v", err)
	}

	data, _ := os.ReadFile(outFile)
	if !bytes.Contains(data, []byte("export-test")) {
		t.Error("JSON export should contain the commit SHA")
	}
	if !bytes.Contains(data, []byte("commit_mode")) {
		t.Error("JSON export should contain commit_mode field")
	}
}

func TestExportCSV(t *testing.T) {
	tmpDB := filepath.Join(t.TempDir(), "test.db")
	t.Setenv("GITX_JOURNAL_PATH", tmpDB)

	j, _ := journal.Open(tmpDB)
	j.LogCommit(journal.CommitRecord{
		SHA: "csv-test", Repo: "/test", RepoName: "test",
		Branch: "main", Message: "fix: csv", Author: "Test", Email: "t@t.com",
		CommitMode: "AI Assisted", AITool: "Kiro", CommittedAt: time.Now(),
	})
	j.Close()

	outFile := filepath.Join(t.TempDir(), "out.csv")
	exportFormat = "csv"
	exportOutput = outFile
	err := runExport()
	if err != nil {
		t.Fatalf("runExport(csv) failed: %v", err)
	}

	data, _ := os.ReadFile(outFile)
	if !bytes.Contains(data, []byte("csv-test")) {
		t.Error("CSV should contain SHA")
	}
	if !bytes.Contains(data, []byte("commit_mode")) {
		t.Error("CSV should have commit_mode header")
	}
}

func TestExportMarkdown(t *testing.T) {
	tmpDB := filepath.Join(t.TempDir(), "test.db")
	t.Setenv("GITX_JOURNAL_PATH", tmpDB)

	j, _ := journal.Open(tmpDB)
	j.LogCommit(journal.CommitRecord{
		SHA: "md-test-sha-1234567890", Repo: "/test", RepoName: "test",
		Branch: "main", Message: "docs: readme", Author: "Test", Email: "t@t.com",
		CommitMode: "Manual", CommittedAt: time.Now(),
	})
	j.Close()

	outFile := filepath.Join(t.TempDir(), "out.md")
	exportFormat = "md"
	exportOutput = outFile
	err := runExport()
	if err != nil {
		t.Fatalf("runExport(md) failed: %v", err)
	}

	data, _ := os.ReadFile(outFile)
	if !bytes.Contains(data, []byte("md-test")) {
		t.Error("Markdown should contain SHA prefix")
	}
	if !bytes.Contains(data, []byte("Mode")) {
		t.Error("Markdown should have Mode column")
	}
}

func TestExportUnsupportedFormat(t *testing.T) {
	tmpDB := filepath.Join(t.TempDir(), "test.db")
	t.Setenv("GITX_JOURNAL_PATH", tmpDB)
	j, _ := journal.Open(tmpDB)
	j.LogCommit(journal.CommitRecord{
		SHA: "bad-fmt", Repo: "/test", RepoName: "test",
		Branch: "main", Message: "test", Author: "t", Email: "t@t.com",
		CommitMode: "Manual", CommittedAt: time.Now(),
	})
	j.Close()

	exportFormat = "xml"
	exportOutput = filepath.Join(t.TempDir(), "out.xml")
	err := runExport()
	if err == nil {
		t.Fatal("unsupported format should error")
	}
}

func TestExportEmptyJournal(t *testing.T) {
	tmpDB := filepath.Join(t.TempDir(), "test.db")
	t.Setenv("GITX_JOURNAL_PATH", tmpDB)
	j, _ := journal.Open(tmpDB)
	j.Close()

	exportFormat = "json"
	exportOutput = filepath.Join(t.TempDir(), "out.json")

	output := captureOutput(t, func() {
		runExport()
	})
	if !bytes.Contains([]byte(output), []byte("No commits")) {
		t.Error("empty journal should say no commits")
	}
}

// --- Stats tests ---

func TestStatsWithData(t *testing.T) {
	tmpDB := filepath.Join(t.TempDir(), "test.db")
	t.Setenv("GITX_JOURNAL_PATH", tmpDB)

	j, _ := journal.Open(tmpDB)
	j.LogCommit(journal.CommitRecord{
		SHA: "s1", Repo: "/test", RepoName: "test", Branch: "main",
		Message: "feat: a", Author: "t", Email: "t@t.com",
		AITool: "Kiro", AIModel: "opus", CommitMode: "AI Assisted",
		CommittedAt: time.Now(),
	})
	j.LogCommit(journal.CommitRecord{
		SHA: "s2", Repo: "/test", RepoName: "test", Branch: "main",
		Message: "fix: b", Author: "t", Email: "t@t.com",
		CommitMode: "Manual", CommittedAt: time.Now(),
	})
	j.Close()

	output := captureOutput(t, func() {
		showStats()
	})

	checks := []string{"Total commits", "Manual", "AI Assisted", "Kiro", "opus", "Repositories"}
	for _, c := range checks {
		if !bytes.Contains([]byte(output), []byte(c)) {
			t.Errorf("stats missing '%s'", c)
		}
	}
}

func TestStatsEmpty(t *testing.T) {
	tmpDB := filepath.Join(t.TempDir(), "test.db")
	t.Setenv("GITX_JOURNAL_PATH", tmpDB)
	j, _ := journal.Open(tmpDB)
	j.Close()

	output := captureOutput(t, func() {
		showStats()
	})
	if !bytes.Contains([]byte(output), []byte("No commits")) {
		t.Error("empty stats should say no commits")
	}
}
