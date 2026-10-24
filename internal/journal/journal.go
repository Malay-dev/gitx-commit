// Package journal stores commit metadata in a local SQLite database.
// Every commit across all repositories is logged here, enabling
// cross-repo analytics and year-end review exports.
//
// Database location: ~/.gitx/commits.db (override with GITX_JOURNAL_PATH)
//
// Schema is versioned — migrations run automatically on Open().
package journal

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // SQLite driver — blank import registers it with database/sql
)

// Current schema version. Bump this when adding migrations.
const schemaVersion = 2

// CommitRecord is a single commit entry in the journal.
type CommitRecord struct {
	SHA          string
	Repo         string
	RepoName     string
	Branch       string
	Message      string
	Author       string
	Email        string
	FilesChanged int
	Additions    int
	Deletions    int
	AITool       string
	AIModel      string
	Confidence   float64
	CommitMode   string
	CommittedAt  time.Time
}

// ToolStats holds the count of commits per AI tool.
type ToolStats struct {
	Tool  string
	Count int
}

// ModelStats holds the count of commits per AI model.
type ModelStats struct {
	Model string
	Count int
}

// DateRange holds the earliest and latest commit dates.
type DateRange struct {
	Earliest time.Time
	Latest   time.Time
}

// Journal manages the SQLite commit database.
type Journal struct {
	db   *sql.DB
	path string
}

// Open creates or opens the journal database at the given path.
// If the path starts with ~, it is expanded to the user's home directory.
// Migrations run automatically to bring the schema up to date.
func Open(dbPath string) (*Journal, error) {
	// Expand ~ to home directory
	if len(dbPath) > 0 && dbPath[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		dbPath = filepath.Join(home, dbPath[1:])
	}

	// Create parent directories if needed
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create journal directory: %w", err)
	}

	// Open SQLite database
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open journal database: %w", err)
	}

	// Run migrations (auto-backup before schema changes)
	if err := migrate(db, dbPath); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to migrate journal schema: %w", err)
	}

	return &Journal{db: db, path: dbPath}, nil
}

// Close closes the database connection.
func (j *Journal) Close() error {
	return j.db.Close()
}

// Backup copies the database file to a timestamped backup.
// Example: commits.db → commits.db.2026-10-21T230800.bak
// Each backup is unique — previous backups are not overwritten.
func (j *Journal) Backup() (string, error) {
	timestamp := time.Now().Format("2006-01-02T150405")
	backupPath := j.path + "." + timestamp + ".bak"

	src, err := os.Open(j.path)
	if err != nil {
		return "", fmt.Errorf("failed to open database for backup: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("failed to copy database: %w", err)
	}
	return backupPath, nil
}

// --- Write operations ---

// LogCommit writes a single commit record to the journal.
func (j *Journal) LogCommit(record CommitRecord) error {
	_, err := j.db.Exec(`
		INSERT OR IGNORE INTO commits (
			sha, repo, repo_name, branch, message,
			author, email, files_changed, additions, deletions,
			ai_tool, ai_model, confidence, commit_mode, committed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.SHA,
		record.Repo,
		record.RepoName,
		record.Branch,
		record.Message,
		record.Author,
		record.Email,
		record.FilesChanged,
		record.Additions,
		record.Deletions,
		record.AITool,
		record.AIModel,
		record.Confidence,
		record.CommitMode,
		record.CommittedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to log commit: %w", err)
	}
	return nil
}

// PurgeRepo deletes all journal entries for a specific repository.
// Returns the number of rows deleted.
func (j *Journal) PurgeRepo(repoName string) (int64, error) {
	result, err := j.db.Exec("DELETE FROM commits WHERE repo_name = ?", repoName)
	if err != nil {
		return 0, fmt.Errorf("failed to purge repo %s: %w", repoName, err)
	}
	return result.RowsAffected()
}

// --- Read operations ---

// Count returns the total number of commits in the journal.
func (j *Journal) Count() (int, error) {
	var count int
	err := j.db.QueryRow("SELECT COUNT(*) FROM commits").Scan(&count)
	return count, err
}

// CountByMode returns the number of commits grouped by commit mode.
func (j *Journal) CountByMode() (manual, assisted int, err error) {
	err = j.db.QueryRow("SELECT COUNT(*) FROM commits WHERE commit_mode = 'Manual' OR (commit_mode = '' AND ai_tool = '')").Scan(&manual)
	if err != nil {
		return
	}
	err = j.db.QueryRow("SELECT COUNT(*) FROM commits WHERE commit_mode LIKE '%AI%' OR (commit_mode = '' AND ai_tool != '')").Scan(&assisted)
	return
}

// CountByTool returns commit counts grouped by AI tool, ordered by count descending.
func (j *Journal) CountByTool() ([]ToolStats, error) {
	rows, err := j.db.Query(`
		SELECT ai_tool, COUNT(*) as cnt
		FROM commits
		WHERE ai_tool != ''
		GROUP BY ai_tool
		ORDER BY cnt DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []ToolStats
	for rows.Next() {
		var s ToolStats
		if err := rows.Scan(&s.Tool, &s.Count); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// CountByModel returns commit counts grouped by AI model, ordered by count descending.
func (j *Journal) CountByModel() ([]ModelStats, error) {
	rows, err := j.db.Query(`
		SELECT ai_model, COUNT(*) as cnt
		FROM commits
		WHERE ai_model != ''
		GROUP BY ai_model
		ORDER BY cnt DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []ModelStats
	for rows.Next() {
		var s ModelStats
		if err := rows.Scan(&s.Model, &s.Count); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// CountRepos returns the number of distinct repositories in the journal.
func (j *Journal) CountRepos() (int, error) {
	var count int
	err := j.db.QueryRow("SELECT COUNT(DISTINCT repo_name) FROM commits").Scan(&count)
	return count, err
}

// GetDateRange returns the earliest and latest commit dates.
func (j *Journal) GetDateRange() (*DateRange, error) {
	var earliest, latest sql.NullString
	err := j.db.QueryRow(`
		SELECT MIN(committed_at), MAX(committed_at)
		FROM commits`).Scan(&earliest, &latest)
	if err != nil {
		return nil, err
	}
	if !earliest.Valid || earliest.String == "" {
		return nil, nil // no commits
	}

	dr := &DateRange{}
	dr.Earliest, _ = time.Parse(time.RFC3339, earliest.String)
	dr.Latest, _ = time.Parse(time.RFC3339, latest.String)
	return dr, nil
}

// RecentCommits returns the last N commits, newest first.
func (j *Journal) RecentCommits(limit int) ([]CommitRecord, error) {
	rows, err := j.db.Query(`
		SELECT sha, repo, repo_name, branch, message,
			   author, email, files_changed, additions, deletions,
			   ai_tool, ai_model, confidence, commit_mode, committed_at
		FROM commits
		ORDER BY committed_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query commits: %w", err)
	}
	defer rows.Close()

	return scanCommits(rows)
}

// CommitsByRepo returns all commits for a specific repository.
func (j *Journal) CommitsByRepo(repoName string, limit int) ([]CommitRecord, error) {
	rows, err := j.db.Query(`
		SELECT sha, repo, repo_name, branch, message,
			   author, email, files_changed, additions, deletions,
			   ai_tool, ai_model, confidence, commit_mode, committed_at
		FROM commits
		WHERE repo_name = ?
		ORDER BY committed_at DESC
		LIMIT ?`, repoName, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query commits: %w", err)
	}
	defer rows.Close()

	return scanCommits(rows)
}

// AllCommits returns all commits in the journal, newest first.
func (j *Journal) AllCommits() ([]CommitRecord, error) {
	rows, err := j.db.Query(`
		SELECT sha, repo, repo_name, branch, message,
			   author, email, files_changed, additions, deletions,
			   ai_tool, ai_model, confidence, commit_mode, committed_at
		FROM commits
		ORDER BY committed_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("failed to query all commits: %w", err)
	}
	defer rows.Close()

	return scanCommits(rows)
}

// QueryFilter holds filter options for journal queries.
type QueryFilter struct {
	Repo   string
	Mode   string // "manual", "ai-assisted", "ai-generated"
	Tool   string
	Model  string
	Search string // full-text search in commit messages
	Since  string // RFC3339 or date string
	Until  string // RFC3339 or date string
	Limit  int
}

// QueryFiltered returns commits matching all provided filters.
func (j *Journal) QueryFiltered(f QueryFilter) ([]CommitRecord, error) {
	query := `
		SELECT sha, repo, repo_name, branch, message,
			   author, email, files_changed, additions, deletions,
			   ai_tool, ai_model, confidence, commit_mode, committed_at
		FROM commits WHERE 1=1`

	var args []interface{}

	if f.Repo != "" {
		query += " AND repo_name = ?"
		args = append(args, f.Repo)
	}
	if f.Mode != "" {
		query += " AND LOWER(commit_mode) = LOWER(?)"
		args = append(args, f.Mode)
	}
	if f.Tool != "" {
		query += " AND LOWER(ai_tool) = LOWER(?)"
		args = append(args, f.Tool)
	}
	if f.Model != "" {
		query += " AND LOWER(ai_model) = LOWER(?)"
		args = append(args, f.Model)
	}
	if f.Search != "" {
		query += " AND message LIKE ?"
		args = append(args, "%"+f.Search+"%")
	}
	if f.Since != "" {
		query += " AND committed_at >= ?"
		args = append(args, f.Since)
	}
	if f.Until != "" {
		query += " AND committed_at <= ?"
		args = append(args, f.Until)
	}

	query += " ORDER BY committed_at DESC"

	if f.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, f.Limit)
	}

	rows, err := j.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query commits: %w", err)
	}
	defer rows.Close()

	return scanCommits(rows)
}

// --- Internal helpers ---

// scanCommits reads rows into CommitRecord slices.
func scanCommits(rows *sql.Rows) ([]CommitRecord, error) {
	var records []CommitRecord
	for rows.Next() {
		var r CommitRecord
		var committedAt string
		err := rows.Scan(
			&r.SHA, &r.Repo, &r.RepoName, &r.Branch, &r.Message,
			&r.Author, &r.Email, &r.FilesChanged, &r.Additions, &r.Deletions,
			&r.AITool, &r.AIModel, &r.Confidence, &r.CommitMode, &committedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan commit row: %w", err)
		}
		r.CommittedAt, _ = time.Parse(time.RFC3339, committedAt)
		records = append(records, r)
	}
	return records, rows.Err()
}

// --- Migrations ---

// migrate brings the database schema up to the current version.
// If there are pending migrations and the database already exists,
// a timestamped backup is created first.
// Each migration runs inside a transaction. New migrations are
// appended to the migrations slice — never modify existing ones.
func migrate(db *sql.DB, dbPath string) error {
	// Create version table if needed
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_version (
			version INTEGER NOT NULL DEFAULT 0
		)`)
	if err != nil {
		return err
	}

	// Get current version
	var current int
	row := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version")
	if err := row.Scan(&current); err != nil {
		return err
	}

	// All migrations in order. Each is a SQL string.
	// RULE: never edit existing migrations — only append new ones.
	// RULE: update schemaVersion constant when adding a migration.
	migrations := []string{
		// Migration 1: initial schema
		`CREATE TABLE IF NOT EXISTS commits (
			sha           TEXT PRIMARY KEY,
			repo          TEXT NOT NULL,
			repo_name     TEXT NOT NULL,
			branch        TEXT NOT NULL DEFAULT '',
			message       TEXT NOT NULL DEFAULT '',
			author        TEXT NOT NULL DEFAULT '',
			email         TEXT NOT NULL DEFAULT '',
			files_changed INTEGER NOT NULL DEFAULT 0,
			additions     INTEGER NOT NULL DEFAULT 0,
			deletions     INTEGER NOT NULL DEFAULT 0,
			ai_tool       TEXT NOT NULL DEFAULT '',
			ai_model      TEXT NOT NULL DEFAULT '',
			confidence    REAL NOT NULL DEFAULT 0.0,
			committed_at  TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_commits_repo ON commits(repo_name);
		CREATE INDEX IF NOT EXISTS idx_commits_date ON commits(committed_at);
		CREATE INDEX IF NOT EXISTS idx_commits_tool ON commits(ai_tool);`,

		// Migration 2: add commit_mode column for Manual/AI Assisted/AI Generated tracking
		`ALTER TABLE commits ADD COLUMN commit_mode TEXT NOT NULL DEFAULT '';
		CREATE INDEX IF NOT EXISTS idx_commits_mode ON commits(commit_mode);`,
	}

	// Safety check: schemaVersion constant must match migrations count
	if len(migrations) != schemaVersion {
		return fmt.Errorf("bug: schemaVersion is %d but there are %d migrations", schemaVersion, len(migrations))
	}

	// Run pending migrations
	if current < len(migrations) {
		// Auto-backup before schema changes (only if DB already has data)
		if current > 0 {
			backupPath := dbPath + ".pre-migration.bak"
			if src, err := os.Open(dbPath); err == nil {
				if dst, err := os.Create(backupPath); err == nil {
					io.Copy(dst, src)
					dst.Close()
				}
				src.Close()
			}
		}
	}

	for i := current; i < len(migrations); i++ {
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("migration %d: failed to begin transaction: %w", i+1, err)
		}

		if _, err := tx.Exec(migrations[i]); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d failed: %w", i+1, err)
		}

		// Update version
		if current == 0 && i == 0 {
			_, err = tx.Exec("INSERT INTO schema_version (version) VALUES (?)", i+1)
		} else {
			_, err = tx.Exec("UPDATE schema_version SET version = ?", i+1)
		}
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: failed to update version: %w", i+1, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("migration %d: failed to commit: %w", i+1, err)
		}
	}

	return nil
}
