// cmd/hook.go — Internal commands called by git hooks.
//
// These are NOT user-facing commands. They're called by the shell scripts
// that Install() writes to .git/hooks/:
//
//	git-xcommit hook prepare-commit-msg <file> <source> [sha]
//	git-xcommit hook post-commit
//
// The "hook" command is hidden from --help output.
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Malay-dev/gitx-commit/internal/config"
	"github.com/Malay-dev/gitx-commit/internal/detect"
	"github.com/Malay-dev/gitx-commit/internal/journal"
	"github.com/Malay-dev/gitx-commit/internal/trailer"
)

// hookCmd is the parent command. Not runnable itself — just a namespace.
var hookCmd = &cobra.Command{
	Use:    "hook",
	Short:  "Internal commands called by git hooks",
	Hidden: true,
}

// prepareCommitMsgCmd is called by the prepare-commit-msg hook.
var prepareCommitMsgCmd = &cobra.Command{
	Use:   "prepare-commit-msg <file> [source] [sha]",
	Short: "Process commit message before commit",
	Args:  cobra.RangeArgs(1, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		msgFile := args[0]
		source := ""
		if len(args) >= 2 {
			source = args[1]
		}
		return runPrepareCommitMsg(msgFile, source)
	},
}

// postCommitCmd is called by the post-commit hook.
var postCommitCmd = &cobra.Command{
	Use:   "post-commit",
	Short: "Log commit to journal after commit",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPostCommit()
	},
}

func init() {
	hookCmd.AddCommand(prepareCommitMsgCmd)
	hookCmd.AddCommand(postCommitCmd)
	rootCmd.AddCommand(hookCmd)
}

// runPrepareCommitMsg is the core runtime path. It runs on EVERY commit.
// Must be fast — anything slow here annoys the user.
//
// Flow:
//  1. Read the commit message file
//  2. Detect AI tool
//  3. Add Assisted-by trailer (if AI detected above threshold)
//  4. Add Commit-Mode trailer (always — Manual or AI Assisted)
//  5. Write modified message back
func runPrepareCommitMsg(msgFile, source string) error {
	cfg := loadConfig()

	if !cfg.General.Enabled {
		return nil
	}

	// Skip trailer injection on merge and squash commits — they're automated
	if source == "merge" || source == "squash" {
		return nil
	}

	// Read the commit message
	msgBytes, err := os.ReadFile(msgFile)
	if err != nil {
		return fmt.Errorf("failed to read commit message: %w", err)
	}
	message := string(msgBytes)

	// Find repo root for detection
	repoPath := getRepoPath()

	if cfg.AI.Enabled {
		// Build detection options from config overrides
		opts := &detect.DetectOptions{
			OverrideTool:  cfg.AI.OverrideTool,
			OverrideModel: cfg.AI.OverrideModel,
		}

		// Detect AI tool
		result := detect.Detect(repoPath, opts)

		if result != nil && result.Confidence >= cfg.AI.ConfidenceThreshold {
			// Add Assisted-by trailer only
			key := cfg.Template.Trailer.Format
			value := trailer.BuildTrailerValue(
				result.Tool,
				result.Model,
				cfg.Template.Trailer.IncludeModel,
			)
			message = trailer.AddTrailer(message, key, value)
		}
	}

	// Only write the message if we added a trailer
	if message != string(msgBytes) {
		if err := os.WriteFile(msgFile, []byte(message), 0644); err != nil {
			return fmt.Errorf("failed to write commit message: %w", err)
		}
	}

	return nil
}

// runPostCommit logs the commit to the SQLite journal.
func runPostCommit() error {
	cfg := loadConfig()

	if !cfg.General.Enabled || !cfg.Journal.Enabled {
		return nil
	}

	j, err := journal.Open(config.JournalDBPath(cfg))
	if err != nil {
		fmt.Fprintf(os.Stderr, "gitx-commit: journal error: %v\n", err)
		return nil
	}
	defer j.Close()

	record, err := buildCommitRecord()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gitx-commit: failed to read commit data: %v\n", err)
		return nil
	}

	// Add AI detection info
	repoPath := getRepoPath()

	// Config-level override
	opts := &detect.DetectOptions{
		OverrideTool:  cfg.AI.OverrideTool,
		OverrideModel: cfg.AI.OverrideModel,
	}

	record.CommitMode = "Manual"
	if modeOverride := os.Getenv("GITX_COMMIT_MODE"); modeOverride != "" {
		record.CommitMode = modeOverride
	} else if result := detect.Detect(repoPath, opts); result != nil && result.Confidence >= cfg.AI.ConfidenceThreshold {
		record.AITool = result.Tool
		record.AIModel = result.Model
		record.Confidence = result.Confidence
		record.CommitMode = "AI Assisted"
	}

	if err := j.LogCommit(record); err != nil {
		fmt.Fprintf(os.Stderr, "gitx-commit: failed to log commit: %v\n", err)
	}

	return nil
}

// buildCommitRecord extracts metadata from the most recent commit using git log.
func buildCommitRecord() (journal.CommitRecord, error) {
	var record journal.CommitRecord

	out, err := exec.Command("git", "log", "-1",
		"--format=%H%n%s%n%an%n%ae%n%aI",
	).Output()
	if err != nil {
		return record, fmt.Errorf("git log failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 5 {
		return record, fmt.Errorf("unexpected git log output: %s", out)
	}

	record.SHA = lines[0]
	record.Message = lines[1]
	record.Author = lines[2]
	record.Email = lines[3]

	if t, err := time.Parse(time.RFC3339, lines[4]); err == nil {
		record.CommittedAt = t
	} else {
		record.CommittedAt = time.Now()
	}

	if out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output(); err == nil {
		record.Repo = strings.TrimSpace(string(out))
		record.RepoName = filepath.Base(record.Repo)
	}

	if out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output(); err == nil {
		record.Branch = strings.TrimSpace(string(out))
	}

	if out, err := exec.Command("git", "diff", "--shortstat", "HEAD~1", "HEAD").Output(); err == nil {
		parseShortStat(strings.TrimSpace(string(out)), &record)
	}

	return record, nil
}

// parseShortStat parses "3 files changed, 50 insertions(+), 10 deletions(-)"
func parseShortStat(stat string, record *journal.CommitRecord) {
	fmt.Sscanf(stat, "%d file", &record.FilesChanged)

	if idx := strings.Index(stat, "insertion"); idx > 0 {
		part := stat[:idx]
		fields := strings.Fields(part)
		if len(fields) > 0 {
			fmt.Sscanf(fields[len(fields)-1], "%d", &record.Additions)
		}
	}

	if idx := strings.Index(stat, "deletion"); idx > 0 {
		part := stat[:idx]
		fields := strings.Fields(part)
		if len(fields) > 0 {
			fmt.Sscanf(fields[len(fields)-1], "%d", &record.Deletions)
		}
	}
}
