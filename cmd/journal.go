package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/Malay-dev/gitx-commit/internal/config"
	"github.com/Malay-dev/gitx-commit/internal/journal"
	"github.com/Malay-dev/gitx-commit/internal/ui"
)

var (
	journalLimit  int
	journalRepo   string
	journalMode   string
	journalTool   string
	journalSearch string
	journalSince  string
	journalUntil  string
	journalPurge  string
	journalBackup bool
)

var journalCmd = &cobra.Command{
	Use:   "journal",
	Short: "Show recent commits from the journal",
	Long: `Displays commit entries logged by gitx-commit across all
repositories. Supports filtering by time, repo, AI tool, commit mode,
and full-text search.

Use --backup to create a timestamped backup of the journal database.
Use --purge-repo to delete all entries for a specific repository.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if journalBackup {
			return backupJournal()
		}
		if journalPurge != "" {
			return purgeRepo(journalPurge)
		}
		return showJournal()
	},
}

func showJournal() error {
	cfg := loadConfig()

	j, err := journal.Open(config.JournalDBPath(cfg))
	if err != nil {
		return fmt.Errorf("failed to open journal: %w", err)
	}
	defer j.Close()

	// Resolve natural language dates
	since := resolveDate(journalSince)
	until := resolveDate(journalUntil)

	// Build filter
	filter := journal.QueryFilter{
		Repo:   journalRepo,
		Mode:   journalMode,
		Tool:   journalTool,
		Search: journalSearch,
		Since:  since,
		Until:  until,
		Limit:  journalLimit,
	}

	records, err := j.QueryFiltered(filter)
	if err != nil {
		return fmt.Errorf("failed to query journal: %w", err)
	}

	if len(records) == 0 {
		fmt.Println("\nNo commits found.")
		fmt.Println()
		return nil
	}

	fmt.Printf("\nRecent Commits (%d):\n\n", len(records))

	for _, r := range records {
		shortSHA := r.SHA
		if len(shortSHA) > 7 {
			shortSHA = shortSHA[:7]
		}

		modeStr := r.CommitMode
		if modeStr == "" {
			modeStr = "unknown"
		}

		aiStr := ""
		if r.AITool != "" {
			aiStr = r.AITool
			if r.AIModel != "" {
				aiStr += " (" + r.AIModel + ")"
			}
		}

		// Line 1: SHA, repo, message
		fmt.Printf("  %s  %-16s  %s\n", shortSHA, r.RepoName, r.Message)

		// Line 2: branch, mode, AI tool, date
		detail := fmt.Sprintf("%-16s  [%s]", r.Branch, modeStr)
		if aiStr != "" {
			detail += "  " + aiStr
		}
		detail += "  " + r.CommittedAt.Format("2006-01-02 15:04")

		fmt.Printf("           %s\n", detail)
		fmt.Println()
	}

	return nil
}

// resolveDate converts natural language like "last week", "last month"
// to RFC3339 date strings. Returns empty string if input is empty or
// already a date.
func resolveDate(input string) string {
	if input == "" {
		return ""
	}

	now := time.Now()

	switch input {
	case "today":
		return now.Truncate(24 * time.Hour).Format(time.RFC3339)
	case "yesterday":
		return now.AddDate(0, 0, -1).Truncate(24 * time.Hour).Format(time.RFC3339)
	case "last week":
		return now.AddDate(0, 0, -7).Format(time.RFC3339)
	case "last month":
		return now.AddDate(0, -1, 0).Format(time.RFC3339)
	case "last year":
		return now.AddDate(-1, 0, 0).Format(time.RFC3339)
	}

	// Try parsing as a date (YYYY-MM-DD)
	if t, err := time.Parse("2006-01-02", input); err == nil {
		return t.Format(time.RFC3339)
	}

	// Try parsing as full RFC3339
	if _, err := time.Parse(time.RFC3339, input); err == nil {
		return input
	}

	return input // pass through as-is
}

func backupJournal() error {
	cfg := loadConfig()

	j, err := journal.Open(config.JournalDBPath(cfg))
	if err != nil {
		return fmt.Errorf("failed to open journal: %w", err)
	}
	defer j.Close()

	path, err := j.Backup()
	if err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	ui.Success("Journal backed up to %s", path)
	return nil
}

func purgeRepo(repoName string) error {
	cfg := loadConfig()

	j, err := journal.Open(config.JournalDBPath(cfg))
	if err != nil {
		return fmt.Errorf("failed to open journal: %w", err)
	}
	defer j.Close()

	deleted, err := j.PurgeRepo(repoName)
	if err != nil {
		return fmt.Errorf("failed to purge: %w", err)
	}

	if deleted == 0 {
		fmt.Printf("No entries found for repo '%s'\n", repoName)
	} else {
		ui.Success("Purged %d entries for repo '%s'", deleted, repoName)
	}
	return nil
}

func init() {
	journalCmd.Flags().IntVarP(&journalLimit, "limit", "n", 10, "number of commits to show")
	journalCmd.Flags().StringVarP(&journalRepo, "repo", "r", "", "filter by repository name")
	journalCmd.Flags().StringVar(&journalMode, "mode", "", "filter by commit mode (manual, ai-assisted)")
	journalCmd.Flags().StringVar(&journalTool, "tool", "", "filter by AI tool name")
	journalCmd.Flags().StringVar(&journalSearch, "search", "", "search in commit messages")
	journalCmd.Flags().StringVar(&journalSince, "since", "", "show commits after date (YYYY-MM-DD, 'last week', 'last month')")
	journalCmd.Flags().StringVar(&journalUntil, "until", "", "show commits before date (YYYY-MM-DD)")
	journalCmd.Flags().BoolVar(&journalBackup, "backup", false, "create a timestamped backup of the journal")
	journalCmd.Flags().StringVar(&journalPurge, "purge-repo", "", "delete all entries for a repository")
	rootCmd.AddCommand(journalCmd)
}
