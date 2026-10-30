package journal

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

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
		CommitMode:   "AI Assisted",
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
	dbPath := filepath.Join(t.TempDir(), "nested", "deep", "commits.db")
	j, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() should create parent dirs, got: %v", err)
	}
	j.Close()
}

func TestOpenIdempotent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "commits.db")
	j1, _ := Open(dbPath)
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

func TestOpenExpandsTilde(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", tmpHome)
	}

	j, err := Open("~/test-journal/commits.db")
	if err != nil {
		t.Fatalf("Open() with ~ failed: %v", err)
	}
	defer j.Close()

	// Verify it created the file under tmpHome
	expected := filepath.Join(tmpHome, "test-journal", "commits.db")
	if _, err := os.Stat(expected); err != nil {
		t.Fatalf("database not created at expected path: %v", err)
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
		j.LogCommit(sampleCommit(fmt.Sprintf("sha-%03d", i)))
	}
	count, _ := j.Count()
	if count != 10 {
		t.Errorf("expected 10 commits, got %d", count)
	}
}

func TestLogManualCommit(t *testing.T) {
	j := setupJournal(t)
	record := sampleCommit("manual-123")
	record.AITool = ""
	record.AIModel = ""
	record.Confidence = 0
	record.CommitMode = "Manual"
	err := j.LogCommit(record)
	if err != nil {
		t.Fatalf("LogCommit() for manual commit failed: %v", err)
	}
}

// --- CountByMode tests ---

func TestCountByMode(t *testing.T) {
	j := setupJournal(t)

	for i := 0; i < 3; i++ {
		j.LogCommit(sampleCommit(fmt.Sprintf("ai-%d", i)))
	}
	for i := 0; i < 2; i++ {
		r := sampleCommit(fmt.Sprintf("manual-%d", i))
		r.AITool = ""
		r.AIModel = ""
		r.CommitMode = "Manual"
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

// --- CountByTool tests ---

func TestCountByTool(t *testing.T) {
	j := setupJournal(t)

	for i := 0; i < 3; i++ {
		j.LogCommit(sampleCommit(fmt.Sprintf("kiro-%d", i)))
	}
	for i := 0; i < 2; i++ {
		r := sampleCommit(fmt.Sprintf("claude-%d", i))
		r.AITool = "Claude Code"
		j.LogCommit(r)
	}

	stats, err := j.CountByTool()
	if err != nil {
		t.Fatalf("CountByTool() failed: %v", err)
	}
	if len(stats) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(stats))
	}
	// Ordered by count desc — Kiro first
	if stats[0].Tool != "Kiro" || stats[0].Count != 3 {
		t.Errorf("expected Kiro:3, got %s:%d", stats[0].Tool, stats[0].Count)
	}
}

// --- CountByModel tests ---

func TestCountByModel(t *testing.T) {
	j := setupJournal(t)

	for i := 0; i < 4; i++ {
		j.LogCommit(sampleCommit(fmt.Sprintf("opus-%d", i)))
	}
	r := sampleCommit("gemini-0")
	r.AIModel = "gemini-3-pro"
	j.LogCommit(r)

	stats, err := j.CountByModel()
	if err != nil {
		t.Fatalf("CountByModel() failed: %v", err)
	}
	if len(stats) != 2 {
		t.Fatalf("expected 2 models, got %d", len(stats))
	}
	if stats[0].Model != "claude-opus-4.6" || stats[0].Count != 4 {
		t.Errorf("expected claude-opus-4.6:4, got %s:%d", stats[0].Model, stats[0].Count)
	}
}

// --- CountRepos tests ---

func TestCountRepos(t *testing.T) {
	j := setupJournal(t)

	r1 := sampleCommit("repo1")
	r1.RepoName = "project-a"
	j.LogCommit(r1)

	r2 := sampleCommit("repo2")
	r2.RepoName = "project-b"
	j.LogCommit(r2)

	count, err := j.CountRepos()
	if err != nil {
		t.Fatalf("CountRepos() failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 repos, got %d", count)
	}
}

// --- GetDateRange tests ---

func TestGetDateRange(t *testing.T) {
	j := setupJournal(t)

	r1 := sampleCommit("early")
	r1.CommittedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	j.LogCommit(r1)

	r2 := sampleCommit("late")
	r2.CommittedAt = time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	j.LogCommit(r2)

	dr, err := j.GetDateRange()
	if err != nil {
		t.Fatalf("GetDateRange() failed: %v", err)
	}
	if dr == nil {
		t.Fatal("expected date range, got nil")
	}
	if dr.Earliest.Year() != 2026 || dr.Earliest.Month() != 1 {
		t.Errorf("expected earliest Jan 2026, got %v", dr.Earliest)
	}
	if dr.Latest.Year() != 2026 || dr.Latest.Month() != 12 {
		t.Errorf("expected latest Dec 2026, got %v", dr.Latest)
	}
}

func TestGetDateRangeEmpty(t *testing.T) {
	j := setupJournal(t)
	dr, err := j.GetDateRange()
	if err != nil {
		t.Fatalf("GetDateRange() on empty failed: %v", err)
	}
	if dr != nil {
		t.Error("expected nil date range on empty journal")
	}
}

// --- RecentCommits tests ---

func TestRecentCommits(t *testing.T) {
	j := setupJournal(t)
	for i := 0; i < 5; i++ {
		r := sampleCommit(fmt.Sprintf("sha-%d", i))
		r.CommittedAt = time.Now().Add(time.Duration(i) * time.Hour)
		r.Message = fmt.Sprintf("commit %d", i)
		j.LogCommit(r)
	}

	records, err := j.RecentCommits(3)
	if err != nil {
		t.Fatalf("RecentCommits() failed: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(records))
	}
	if records[0].Message != "commit 4" {
		t.Errorf("first should be newest, got: %s", records[0].Message)
	}
}

func TestRecentCommitsEmpty(t *testing.T) {
	j := setupJournal(t)
	records, err := j.RecentCommits(10)
	if err != nil {
		t.Fatalf("RecentCommits() on empty failed: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0, got %d", len(records))
	}
}

// --- CommitsByRepo tests ---

func TestCommitsByRepo(t *testing.T) {
	j := setupJournal(t)

	for i := 0; i < 3; i++ {
		r := sampleCommit(fmt.Sprintf("a-%d", i))
		r.RepoName = "project-a"
		j.LogCommit(r)
	}
	for i := 0; i < 2; i++ {
		r := sampleCommit(fmt.Sprintf("b-%d", i))
		r.RepoName = "project-b"
		j.LogCommit(r)
	}

	records, _ := j.CommitsByRepo("project-a", 10)
	if len(records) != 3 {
		t.Errorf("expected 3 for project-a, got %d", len(records))
	}

	records, _ = j.CommitsByRepo("nonexistent", 10)
	if len(records) != 0 {
		t.Errorf("expected 0 for nonexistent, got %d", len(records))
	}
}

// --- PurgeRepo tests ---

func TestPurgeRepo(t *testing.T) {
	j := setupJournal(t)

	for i := 0; i < 3; i++ {
		r := sampleCommit(fmt.Sprintf("test-%d", i))
		r.RepoName = "test-repo"
		j.LogCommit(r)
	}
	j.LogCommit(sampleCommit("keep-me"))

	deleted, err := j.PurgeRepo("test-repo")
	if err != nil {
		t.Fatalf("PurgeRepo() failed: %v", err)
	}
	if deleted != 3 {
		t.Errorf("expected 3 deleted, got %d", deleted)
	}

	count, _ := j.Count()
	if count != 1 {
		t.Errorf("expected 1 remaining, got %d", count)
	}
}

func TestPurgeRepoNonexistent(t *testing.T) {
	j := setupJournal(t)
	deleted, err := j.PurgeRepo("nonexistent")
	if err != nil {
		t.Fatalf("PurgeRepo() failed: %v", err)
	}
	if deleted != 0 {
		t.Errorf("expected 0 deleted, got %d", deleted)
	}
}

// --- Backup tests ---

func TestBackup(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "commits.db")
	j, _ := Open(dbPath)
	j.LogCommit(sampleCommit("backup-test"))

	backupPath, err := j.Backup()
	if err != nil {
		t.Fatalf("Backup() failed: %v", err)
	}
	if _, err := os.Stat(backupPath); err != nil {
		t.Fatalf("backup file not created: %v", err)
	}

	j.Close()

	// Verify backup is a valid database
	j2, err := Open(backupPath)
	if err != nil {
		t.Fatalf("backup is not a valid database: %v", err)
	}
	defer j2.Close()
	count, _ := j2.Count()
	if count != 1 {
		t.Errorf("backup should have 1 commit, got %d", count)
	}
}

// --- QueryFiltered tests ---

func TestQueryFilteredByRepo(t *testing.T) {
	j := setupJournal(t)
	r := sampleCommit("filtered-1")
	r.RepoName = "target-repo"
	j.LogCommit(r)
	j.LogCommit(sampleCommit("other"))

	records, _ := j.QueryFiltered(QueryFilter{Repo: "target-repo", Limit: 10})
	if len(records) != 1 {
		t.Errorf("expected 1, got %d", len(records))
	}
}

func TestQueryFilteredByMode(t *testing.T) {
	j := setupJournal(t)

	j.LogCommit(sampleCommit("ai-1"))

	r := sampleCommit("manual-1")
	r.CommitMode = "Manual"
	j.LogCommit(r)

	records, _ := j.QueryFiltered(QueryFilter{Mode: "Manual", Limit: 10})
	if len(records) != 1 {
		t.Errorf("expected 1 manual, got %d", len(records))
	}
}

func TestQueryFilteredByTool(t *testing.T) {
	j := setupJournal(t)
	j.LogCommit(sampleCommit("kiro-1"))

	r := sampleCommit("claude-1")
	r.AITool = "Claude Code"
	j.LogCommit(r)

	records, _ := j.QueryFiltered(QueryFilter{Tool: "Kiro", Limit: 10})
	if len(records) != 1 {
		t.Errorf("expected 1, got %d", len(records))
	}
}

func TestQueryFilteredBySearch(t *testing.T) {
	j := setupJournal(t)

	r1 := sampleCommit("auth-1")
	r1.Message = "feat(auth): add OAuth"
	j.LogCommit(r1)

	r2 := sampleCommit("db-1")
	r2.Message = "fix(db): connection pool"
	j.LogCommit(r2)

	records, _ := j.QueryFiltered(QueryFilter{Search: "auth", Limit: 10})
	if len(records) != 1 {
		t.Errorf("expected 1 matching 'auth', got %d", len(records))
	}
}

func TestQueryFilteredBySince(t *testing.T) {
	j := setupJournal(t)

	old := sampleCommit("old-1")
	old.CommittedAt = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	j.LogCommit(old)

	recent := sampleCommit("new-1")
	recent.CommittedAt = time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	j.LogCommit(recent)

	records, _ := j.QueryFiltered(QueryFilter{Since: "2026-01-01T00:00:00Z", Limit: 10})
	if len(records) != 1 {
		t.Errorf("expected 1 since 2026, got %d", len(records))
	}
}

func TestQueryFilteredCombined(t *testing.T) {
	j := setupJournal(t)

	// AI commit in target repo
	r1 := sampleCommit("match")
	r1.RepoName = "target"
	r1.CommitMode = "AI Assisted"
	j.LogCommit(r1)

	// Manual commit in target repo
	r2 := sampleCommit("no-match")
	r2.RepoName = "target"
	r2.CommitMode = "Manual"
	r2.AITool = ""
	j.LogCommit(r2)

	// AI commit in other repo
	r3 := sampleCommit("wrong-repo")
	r3.RepoName = "other"
	j.LogCommit(r3)

	records, _ := j.QueryFiltered(QueryFilter{Repo: "target", Mode: "AI Assisted", Limit: 10})
	if len(records) != 1 {
		t.Errorf("expected 1 matching both filters, got %d", len(records))
	}
}

func TestQueryFilteredEmpty(t *testing.T) {
	j := setupJournal(t)
	records, _ := j.QueryFiltered(QueryFilter{Limit: 10})
	if len(records) != 0 {
		t.Errorf("expected 0 on empty journal, got %d", len(records))
	}
}

// --- AllCommits tests ---

func TestAllCommits(t *testing.T) {
	j := setupJournal(t)
	for i := 0; i < 5; i++ {
		j.LogCommit(sampleCommit(fmt.Sprintf("all-%d", i)))
	}
	records, err := j.AllCommits()
	if err != nil {
		t.Fatalf("AllCommits() failed: %v", err)
	}
	if len(records) != 5 {
		t.Errorf("expected 5, got %d", len(records))
	}
}

// --- Data roundtrip ---

func TestCommitDataRoundtrip(t *testing.T) {
	j := setupJournal(t)

	original := CommitRecord{
		SHA:          "roundtrip",
		Repo:         "/full/path",
		RepoName:     "my-repo",
		Branch:       "feature/auth",
		Message:      "feat(auth): add OAuth",
		Author:       "Malay Kumar",
		Email:        "malay@example.com",
		FilesChanged: 12,
		Additions:    340,
		Deletions:    89,
		AITool:       "Kiro",
		AIModel:      "claude-opus-4.6",
		Confidence:   0.95,
		CommitMode:   "AI Assisted",
		CommittedAt:  time.Date(2026, 10, 21, 19, 0, 0, 0, time.UTC),
	}

	j.LogCommit(original)
	records, _ := j.RecentCommits(1)
	if len(records) != 1 {
		t.Fatal("expected 1 record")
	}
	got := records[0]

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
		{"CommitMode", got.CommitMode, original.CommitMode},
	}

	for _, c := range checks {
		if fmt.Sprintf("%v", c.got) != fmt.Sprintf("%v", c.expected) {
			t.Errorf("%s: got %v, want %v", c.field, c.got, c.expected)
		}
	}
}

// --- Migration tests ---

func TestMigrationCreatesSchemaVersion(t *testing.T) {
	j := setupJournal(t)
	// Schema version should be set after Open
	var version int
	j.db.QueryRow("SELECT version FROM schema_version").Scan(&version)
	if version != schemaVersion {
		t.Errorf("expected schema version %d, got %d", schemaVersion, version)
	}
}

func TestMigrationIdempotent(t *testing.T) {
	// Open same DB twice — migrations should not re-run
	dbPath := filepath.Join(t.TempDir(), "commits.db")
	j1, _ := Open(dbPath)
	j1.LogCommit(sampleCommit("m1"))
	j1.Close()

	j2, err := Open(dbPath)
	if err != nil {
		t.Fatalf("second Open should not fail: %v", err)
	}
	defer j2.Close()

	count, _ := j2.Count()
	if count != 1 {
		t.Errorf("data should survive re-open, got count=%d", count)
	}

	var version int
	j2.db.QueryRow("SELECT version FROM schema_version").Scan(&version)
	if version != schemaVersion {
		t.Errorf("schema version should be %d, got %d", schemaVersion, version)
	}
}

func TestOpenInvalidPath(t *testing.T) {
	// Try opening in a read-only location (may vary by OS)
	_, err := Open("/dev/null/impossible/path/db.sqlite")
	if err == nil {
		t.Error("Open with invalid path should fail")
	}
}

func TestPurgeRepoReturnsCount(t *testing.T) {
	j := setupJournal(t)

	for i := 0; i < 5; i++ {
		r := sampleCommit(fmt.Sprintf("purge-%d", i))
		r.RepoName = "to-purge"
		j.LogCommit(r)
	}

	deleted, err := j.PurgeRepo("to-purge")
	if err != nil {
		t.Fatalf("PurgeRepo failed: %v", err)
	}
	if deleted != 5 {
		t.Errorf("expected 5 deleted, got %d", deleted)
	}
}

func TestLogCommitErrorHandling(t *testing.T) {
	j := setupJournal(t)

	// Valid commit should succeed
	err := j.LogCommit(sampleCommit("valid"))
	if err != nil {
		t.Fatalf("valid commit should succeed: %v", err)
	}
}

func TestCountByModeEmpty(t *testing.T) {
	j := setupJournal(t)
	manual, assisted, err := j.CountByMode()
	if err != nil {
		t.Fatalf("CountByMode on empty failed: %v", err)
	}
	if manual != 0 || assisted != 0 {
		t.Errorf("empty journal should have 0/0, got %d/%d", manual, assisted)
	}
}

func TestAllCommitsOrdered(t *testing.T) {
	j := setupJournal(t)

	for i := 0; i < 3; i++ {
		r := sampleCommit(fmt.Sprintf("order-%d", i))
		r.CommittedAt = time.Now().Add(time.Duration(i) * time.Hour)
		r.Message = fmt.Sprintf("commit %d", i)
		j.LogCommit(r)
	}

	records, _ := j.AllCommits()
	if len(records) != 3 {
		t.Fatalf("expected 3, got %d", len(records))
	}
	// Should be newest first
	if records[0].Message != "commit 2" {
		t.Errorf("should be newest first, got %s", records[0].Message)
	}
}

func TestQueryFilteredByModel(t *testing.T) {
	j := setupJournal(t)

	r1 := sampleCommit("model-1")
	r1.AIModel = "opus"
	j.LogCommit(r1)

	r2 := sampleCommit("model-2")
	r2.AIModel = "gemini"
	j.LogCommit(r2)

	records, _ := j.QueryFiltered(QueryFilter{Model: "opus", Limit: 10})
	if len(records) != 1 {
		t.Errorf("expected 1 matching opus, got %d", len(records))
	}
}

func TestQueryFilteredByUntil(t *testing.T) {
	j := setupJournal(t)

	old := sampleCommit("old-until")
	old.CommittedAt = time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	j.LogCommit(old)

	new := sampleCommit("new-until")
	new.CommittedAt = time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	j.LogCommit(new)

	records, _ := j.QueryFiltered(QueryFilter{Until: "2026-01-01T00:00:00Z", Limit: 10})
	if len(records) != 1 {
		t.Errorf("expected 1 before 2026, got %d", len(records))
	}
}

func TestCountByToolEmpty(t *testing.T) {
	j := setupJournal(t)
	stats, err := j.CountByTool()
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	if len(stats) != 0 {
		t.Errorf("expected 0 on empty, got %d", len(stats))
	}
}

func TestCountByModelEmpty(t *testing.T) {
	j := setupJournal(t)
	stats, err := j.CountByModel()
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	if len(stats) != 0 {
		t.Errorf("expected 0 on empty, got %d", len(stats))
	}
}

func TestBackupCreatesTimestampedFile(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "commits.db")
	j, _ := Open(dbPath)
	j.LogCommit(sampleCommit("bak-ts"))

	path1, _ := j.Backup()
	time.Sleep(time.Second)
	path2, _ := j.Backup()

	j.Close()

	// Two backups should have different names
	if path1 == path2 {
		t.Error("backups should have different timestamps")
	}
}
