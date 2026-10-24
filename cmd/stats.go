package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Malay-dev/gitx-commit/internal/config"
	"github.com/Malay-dev/gitx-commit/internal/journal"
	"github.com/Malay-dev/gitx-commit/internal/ui"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show AI usage analytics from the commit journal",
	Long: `Reads the local commit journal and displays statistics about
your commits across all repositories — total counts, AI vs manual
breakdown, most-used tools and models.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return showStats()
	},
}

func showStats() error {
	cfg := loadConfig()

	j, err := journal.Open(config.JournalDBPath(cfg))
	if err != nil {
		return fmt.Errorf("failed to open journal: %w", err)
	}
	defer j.Close()

	total, err := j.Count()
	if err != nil {
		return fmt.Errorf("failed to count commits: %w", err)
	}

	if total == 0 {
		fmt.Println()
		ui.Info("No commits logged yet. Enable gitx-commit and start committing!")
		fmt.Println()
		return nil
	}

	manual, assisted, err := j.CountByMode()
	if err != nil {
		return fmt.Errorf("failed to count by mode: %w", err)
	}

	repoCount, err := j.CountRepos()
	if err != nil {
		return fmt.Errorf("failed to count repos: %w", err)
	}

	dateRange, err := j.GetDateRange()
	if err != nil {
		return fmt.Errorf("failed to get date range: %w", err)
	}

	fmt.Println()
	fmt.Println("  Commit Journal Stats:")
	fmt.Println()

	manualPct := float64(manual) / float64(total) * 100
	assistedPct := float64(assisted) / float64(total) * 100

	ui.Field("Total commits", fmt.Sprintf("%d", total))
	ui.Field("Manual", fmt.Sprintf("%d (%.1f%%)", manual, manualPct))
	ui.Field("AI Assisted", fmt.Sprintf("%d (%.1f%%)", assisted, assistedPct))

	toolStats, err := j.CountByTool()
	if err == nil && len(toolStats) > 0 {
		fmt.Println()
		fmt.Println("  Top AI Tools:")
		for _, ts := range toolStats {
			fmt.Printf("    %-20s %d commits\n", ts.Tool, ts.Count)
		}
	}

	modelStats, err := j.CountByModel()
	if err == nil && len(modelStats) > 0 {
		fmt.Println()
		fmt.Println("  Top Models:")
		for _, ms := range modelStats {
			fmt.Printf("    %-20s %d commits\n", ms.Model, ms.Count)
		}
	}

	fmt.Println()
	ui.Field("Repositories", fmt.Sprintf("%d", repoCount))
	if dateRange != nil {
		ui.Field("Date range", fmt.Sprintf("%s — %s",
			dateRange.Earliest.Format("2006-01-02"),
			dateRange.Latest.Format("2006-01-02")))
	}
	fmt.Println()

	return nil
}

func init() {
	rootCmd.AddCommand(statsCmd)
}
