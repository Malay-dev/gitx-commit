package journal

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

// helper: opens a journal in a temp directory
func setupJournal(t *testing.T) *Journal {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test-commits.db")
	j, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open journal: %v", err)
	}
	t.Cleanup(func() { j.Close() })
	return j
}

func sampleCommit(sha string) CommitRecord {
	return CommitRecord{
		SHA:          sha,
		Repo:         "/Users/malay/my-project",
		RepoName:     "my-project",
		Branch:       "main",
		Message:      "feat: add login",
		Author:       "Malay Kumar",
		Email:        "malay@example.com",
		FilesChanged: 3,
		Additions:    50,
		Deletions:    10,
		AITool:       "Kiro",
		AIModel:      "claude-opus-4.6",
		Confidence:   0.95,
		CommittedAt:  time.Now(),
	}
}

// --- Open tests ---

func TestOpenCreatesDatabase(t *testing.T) {
	j := setupJournal(t)

	count, err := j.Count()
	if err != nil {
		t.Fatalf("Count() failed: %v", err)
	}
	if count != 0 {
		t.Errorf("new journal should be empty, got %d", count)
	}
}

func TestOpenCreatesDirIfMissing(t *testing.T) {
	// Path with nested dirs that don't exist yet
	dbPath := filepath.Join(t.TempDir(), "nested", "deep", "commits.db")
	j, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() should create parent dirs, got: %v", err)
	}
	j.Close()
}

func TestOpenIdempotent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "commits.db")

	// Open twice — should not fail or lose data
	j1, err := Open(dbPath)
	if err != nil {
		t.Fatalf("first Open() failed: %v", err)
	}
	j1.LogCommit(sampleCommit("abc123"))
	j1.Close()

	j2, err := Open(dbPath)
	if err != nil {
		t.Fatalf("second Open() failed: %v", err)
	}
	defer j2.Close()

	count, _ := j2.Count()
	if count != 1 {
		t.Errorf("data should persist across opens, got count=%d", count)
	}
}

// --- LogCommit tests ---

func TestLogCommit(t *testing.T) {
	j := setupJournal(t)

	err := j.LogCommit(sampleCommit("abc123"))
	if err != nil {
		t.Fatalf("LogCommit() failed: %v", err)
	}

	count, _ := j.Count()
	if count != 1 {
		t.Errorf("expected 1 commit, got %d", count)
	}
}

func TestLogCommitDuplicateSHA(t *testing.T) {
	j := setupJournal(t)

	// Log same SHA twice — should not error (INSERT OR IGNORE)
	j.LogCommit(sampleCommit("abc123"))
	err := j.LogCommit(sampleCommit("abc123"))

	if err != nil {
		t.Fatalf("duplicate SHA should be ignored, got: %v", err)
	}

	count, _ := j.Count()
	if count != 1 {
		t.Errorf("duplicate SHA should not create second row, got count=%d", count)
	}
}

func TestLogMultipleCommits(t *testing.T) {
	j := setupJournal(t)

	for i := 0; i < 10; i++ {
		sha := fmt.Sprintf("sha-%03d", i)
		j.LogCommit(sampleCommit(sha))
	}

	count, _ := j.Count()
	if count != 10 {
		t.Errorf("expected 10 commits, got %d", count)
	}
}

func TestLogManualCommit(t *testing.T) {
	j := setupJournal(t)

	// Manual commit — no AI tool
	record := sampleCommit("manual-123")
	record.AITool = ""
	record.AIModel = ""
	record.Confidence = 0

	err := j.LogCommit(record)
	if err != nil {
		t.Fatalf("LogCommit() for manual commit failed: %v", err)
	}
}

// --- CountByMode tests ---

func TestCountByMode(t *testing.T) {
	j := setupJournal(t)

	// 3 AI-assisted commits
	for i := 0; i < 3; i++ {
		j.LogCommit(sampleCommit(fmt.Sprintf("ai-%d", i)))
	}

	// 2 manual commits
	for i := 0; i < 2; i++ {
		r := sampleCommit(fmt.Sprintf("manual-%d", i))
		r.AITool = ""
		r.AIModel = ""
		j.LogCommit(r)
	}

	manual, assisted, err := j.CountByMode()
	if err != nil {
		t.Fatalf("CountByMode() failed: %v", err)
	}
	if manual != 2 {
		t.Errorf("expected 2 manual, got %d", manual)
	}
	if assisted != 3 {
		t.Errorf("expected 3 assisted, got %d", assisted)
	}
}

// --- RecentCommits tests ---

func TestRecentCommits(t *testing.T) {
	j := setupJournal(t)

	// Insert commits with different times
	for i := 0; i < 5; i++ {
		r := sampleCommit(fmt.Sprintf("sha-%d", i))
		r.CommittedAt = time.Now().Add(time.Duration(i) * time.Hour)
		r.Message = fmt.Sprintf("commit %d", i)
		j.LogCommit(r)
	}

	// Get last 3
	records, err := j.RecentCommits(3)
	if err != nil {
		t.Fatalf("RecentCommits() failed: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(records))
	}

	// Should be newest first
	if records[0].Message != "commit 4" {
		t.Errorf("first record should be newest, got: %s", records[0].Message)
	}
}

func TestRecentCommitsEmpty(t *testing.T) {
	j := setupJournal(t)

	records, err := j.RecentCommits(10)
	if err != nil {
		t.Fatalf("RecentCommits() on empty journal failed: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}

// --- CommitsByRepo tests ---

func TestCommitsByRepo(t *testing.T) {
	j := setupJournal(t)

	// Commits from two different repos
	for i := 0; i < 3; i++ {
		r := sampleCommit(fmt.Sprintf("proj-a-%d", i))
		r.RepoName = "project-a"
		j.LogCommit(r)
	}
	for i := 0; i < 2; i++ {
		r := sampleCommit(fmt.Sprintf("proj-b-%d", i))
		r.RepoName = "project-b"
		j.LogCommit(r)
	}

	// Query project-a
	records, err := j.CommitsByRepo("project-a", 10)
	if err != nil {
		t.Fatalf("CommitsByRepo() failed: %v", err)
	}
	if len(records) != 3 {
		t.Errorf("expected 3 records for project-a, got %d", len(records))
	}

	// Query project-b
	records, err = j.CommitsByRepo("project-b", 10)
	if err != nil {
		t.Fatalf("CommitsByRepo() failed: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("expected 2 records for project-b, got %d", len(records))
	}

	// Query non-existent repo
	records, err = j.CommitsByRepo("no-such-repo", 10)
	if err != nil {
		t.Fatalf("CommitsByRepo() failed: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records for non-existent repo, got %d", len(records))
	}
}

// --- Data integrity tests ---

func TestCommitDataRoundtrip(t *testing.T) {
	j := setupJournal(t)

	// Write a commit with all fields populated
	original := CommitRecord{
		SHA:          "roundtrip-test",
		Repo:         "/full/path/to/repo",
		RepoName:     "my-repo",
		Branch:       "feature/auth",
		Message:      "feat(auth): add OAuth login\n\nDetailed body here.",
		Author:       "Malay Kumar",
		Email:        "malay@example.com",
		FilesChanged: 12,
		Additions:    340,
		Deletions:    89,
		AITool:       "Kiro",
		AIModel:      "claude-opus-4.6",
		Confidence:   0.95,
		CommittedAt:  time.Date(2026, 10, 21, 19, 0, 0, 0, time.UTC),
	}

	j.LogCommit(original)

	// Read it back
	records, _ := j.RecentCommits(1)
	if len(records) != 1 {
		t.Fatal("expected 1 record")
	}
	got := records[0]

	// Verify every field survived the roundtrip
	checks := []struct {
		field    string
		got      interface{}
		expected interface{}
	}{
		{"SHA", got.SHA, original.SHA},
		{"Repo", got.Repo, original.Repo},
		{"RepoName", got.RepoName, original.RepoName},
		{"Branch", got.Branch, original.Branch},
		{"Message", got.Message, original.Message},
		{"Author", got.Author, original.Author},
		{"Email", got.Email, original.Email},
		{"FilesChanged", got.FilesChanged, original.FilesChanged},
		{"Additions", got.Additions, original.Additions},
		{"Deletions", got.Deletions, original.Deletions},
		{"AITool", got.AITool, original.AITool},
		{"AIModel", got.AIModel, original.AIModel},
	}

	for _, c := range checks {
		if fmt.Sprintf("%v", c.got) != fmt.Sprintf("%v", c.expected) {
			t.Errorf("%s: got %v, want %v", c.field, c.got, c.expected)
		}
	}
}
