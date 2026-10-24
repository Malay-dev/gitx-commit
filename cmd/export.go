package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Malay-dev/gitx-commit/internal/config"
	"github.com/Malay-dev/gitx-commit/internal/journal"
)

var (
	exportFormat string
	exportOutput string
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export the commit journal for reporting and analysis",
	Long: `Exports all commit journal entries to a file. Supported formats:
  json  — structured data, ideal for LLM consumption and year-end reviews
  csv   — spreadsheet-compatible
  md    — human-readable markdown table`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runExport()
	},
}

func runExport() error {
	cfg := loadConfig()

	j, err := journal.Open(config.JournalDBPath(cfg))
	if err != nil {
		return fmt.Errorf("failed to open journal: %w", err)
	}
	defer j.Close()

	records, err := j.AllCommits()
	if err != nil {
		return fmt.Errorf("failed to read commits: %w", err)
	}

	if len(records) == 0 {
		fmt.Println("No commits to export.")
		return nil
	}

	// Default output filename based on format
	if exportOutput == "" {
		exportOutput = "gitx-commits." + exportFormat
	}

	var exportErr error
	switch exportFormat {
	case "json":
		exportErr = exportJSON(records, exportOutput)
	case "csv":
		exportErr = exportCSV(records, exportOutput)
	case "md":
		exportErr = exportMarkdown(records, exportOutput)
	default:
		return fmt.Errorf("unsupported format: %s (use json, csv, or md)", exportFormat)
	}

	if exportErr != nil {
		return exportErr
	}

	fmt.Printf("✓ Exported %d commits to %s\n", len(records), exportOutput)
	return nil
}

func exportJSON(records []journal.CommitRecord, path string) error {
	// Build a clean JSON structure
	type jsonRecord struct {
		SHA          string  `json:"sha"`
		Repo         string  `json:"repo"`
		RepoName     string  `json:"repo_name"`
		Branch       string  `json:"branch"`
		Message      string  `json:"message"`
		Author       string  `json:"author"`
		Email        string  `json:"email"`
		FilesChanged int     `json:"files_changed"`
		Additions    int     `json:"additions"`
		Deletions    int     `json:"deletions"`
		AITool       string  `json:"ai_tool,omitempty"`
		AIModel      string  `json:"ai_model,omitempty"`
		Confidence   float64 `json:"confidence,omitempty"`
		CommitMode   string  `json:"commit_mode"`
		CommittedAt  string  `json:"committed_at"`
	}

	var out []jsonRecord
	for _, r := range records {
		out = append(out, jsonRecord{
			SHA:          r.SHA,
			Repo:         r.Repo,
			RepoName:     r.RepoName,
			Branch:       r.Branch,
			Message:      r.Message,
			Author:       r.Author,
			Email:        r.Email,
			FilesChanged: r.FilesChanged,
			Additions:    r.Additions,
			Deletions:    r.Deletions,
			AITool:       r.AITool,
			AIModel:      r.AIModel,
			Confidence:   r.Confidence,
			CommitMode:   r.CommitMode,
			CommittedAt:  r.CommittedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

func exportCSV(records []journal.CommitRecord, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Header
	w.Write([]string{
		"sha", "repo_name", "branch", "message", "author", "email",
		"files_changed", "additions", "deletions",
		"ai_tool", "ai_model", "confidence", "commit_mode", "committed_at",
	})

	for _, r := range records {
		w.Write([]string{
			r.SHA, r.RepoName, r.Branch, r.Message, r.Author, r.Email,
			fmt.Sprintf("%d", r.FilesChanged),
			fmt.Sprintf("%d", r.Additions),
			fmt.Sprintf("%d", r.Deletions),
			r.AITool, r.AIModel,
			fmt.Sprintf("%.2f", r.Confidence),
			r.CommitMode,
			r.CommittedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	return nil
}

func exportMarkdown(records []journal.CommitRecord, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	fmt.Fprintf(f, "# Commit Journal Export\n\n")
	fmt.Fprintf(f, "Total commits: %d\n\n", len(records))
	fmt.Fprintf(f, "| SHA | Repo | Branch | Message | Mode | AI Tool | AI Model | Date |\n")
	fmt.Fprintf(f, "|-----|------|--------|---------|------|---------|----------|------|\n")

	for _, r := range records {
		shortSHA := r.SHA
		if len(shortSHA) > 7 {
			shortSHA = shortSHA[:7]
		}

		mode := r.CommitMode
		if mode == "" {
			mode = "—"
		}
		aiTool := r.AITool
		if aiTool == "" {
			aiTool = "—"
		}
		aiModel := r.AIModel
		if aiModel == "" {
			aiModel = "—"
		}

		fmt.Fprintf(f, "| %s | %s | %s | %s | %s | %s | %s | %s |\n",
			shortSHA, r.RepoName, r.Branch, r.Message,
			mode, aiTool, aiModel,
			r.CommittedAt.Format("2006-01-02"),
		)
	}

	return nil
}

func init() {
	exportCmd.Flags().StringVarP(&exportFormat, "format", "f", "json", "export format: json, csv, or md")
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "output file path (defaults to gitx-commits.<format>)")
	rootCmd.AddCommand(exportCmd)
}
