package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Malay-dev/gitx-commit/internal/config"
	"github.com/Malay-dev/gitx-commit/internal/hook"
	"github.com/Malay-dev/gitx-commit/internal/ui"
)

var initLocal bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive setup wizard for gitx-commit",
	Long: `Walks you through configuring gitx-commit.

Without flags, sets up the global config (~/.gitx/commit-config.toml).
With --local, creates a repo-level override (.gitx-commit.toml) that
inherits from global — you only change what's different for this repo.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInit()
	},
}

func runInit() error {
	reader := bufio.NewReader(os.Stdin)

	if initLocal {
		return runInitLocal(reader)
	}
	return runInitGlobal(reader)
}

func runInitGlobal(reader *bufio.Reader) error {
	fmt.Println()
	fmt.Println("  gitx-commit — global setup")
	fmt.Println("  ──────────────────────────")
	fmt.Println()

	cfg := config.DefaultConfig()

	fmt.Printf("  Commit template format (conventional/angular/none) [%s]: ", cfg.Template.Format)
	if answer := readLine(reader); answer != "" {
		cfg.Template.Format = answer
	}

	fmt.Printf("  Trailer key (Assisted-by/Co-authored-by/Drafted-With) [%s]: ", cfg.Template.Trailer.Format)
	if answer := readLine(reader); answer != "" {
		cfg.Template.Trailer.Format = answer
	}

	fmt.Printf("  Include AI model in trailer? (y/n) [y]: ")
	if answer := readLine(reader); answer == "n" || answer == "N" {
		cfg.Template.Trailer.IncludeModel = false
	}

	fmt.Printf("  AI detection mode (auto/manual/off) [%s]: ", cfg.AI.Mode)
	if answer := readLine(reader); answer != "" {
		cfg.AI.Mode = answer
	}

	fmt.Printf("  Detection confidence threshold (0.0-1.0) [%.1f]: ", cfg.AI.ConfidenceThreshold)
	if answer := readLine(reader); answer != "" {
		var threshold float64
		if _, err := fmt.Sscanf(answer, "%f", &threshold); err == nil {
			cfg.AI.ConfidenceThreshold = threshold
		}
	}

	fmt.Printf("  Enable commit journal? (y/n) [y]: ")
	if answer := readLine(reader); answer == "n" || answer == "N" {
		cfg.Journal.Enabled = false
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	path, _ := config.GlobalConfigPath()
	ui.Success("Config saved to %s", path)

	// Enable hooks if needed
	installHooksIfNeeded(reader)

	// Ensure .gitx-commit.toml is globally gitignored
	ensureGlobalGitignore()

	fmt.Printf("\n  Setup complete! Run 'git xcommit status' to verify.\n\n")
	return nil
}

func runInitLocal(reader *bufio.Reader) error {
	// Check we're in a git repo
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return fmt.Errorf("not a git repository — --local requires a repo")
	}
	repoPath := strings.TrimSpace(string(out))

	// Load global config as the baseline
	globalCfg := loadConfig()

	fmt.Println()
	fmt.Println("  gitx-commit — repo-level config")
	fmt.Println("  ───────────────────────────────")
	fmt.Printf("  Repository: %s\n", repoPath)
	fmt.Println()
	fmt.Println("  Current global values shown in brackets.")
	fmt.Println("  Press Enter to keep, or type a new value to override.")
	fmt.Println()

	// We build a partial config — only changed fields get written
	repoCfg := config.DefaultConfig()
	changed := false

	// Template format
	fmt.Printf("  Commit template format [%s]: ", globalCfg.Template.Format)
	if answer := readLine(reader); answer != "" && answer != globalCfg.Template.Format {
		repoCfg.Template.Format = answer
		changed = true
	} else {
		repoCfg.Template.Format = globalCfg.Template.Format
	}

	// Trailer format
	fmt.Printf("  Trailer key [%s]: ", globalCfg.Template.Trailer.Format)
	if answer := readLine(reader); answer != "" && answer != globalCfg.Template.Trailer.Format {
		repoCfg.Template.Trailer.Format = answer
		changed = true
	} else {
		repoCfg.Template.Trailer.Format = globalCfg.Template.Trailer.Format
	}

	// Include model
	modelDefault := "y"
	if !globalCfg.Template.Trailer.IncludeModel {
		modelDefault = "n"
	}
	fmt.Printf("  Include AI model in trailer? (y/n) [%s]: ", modelDefault)
	if answer := readLine(reader); answer != "" {
		newVal := answer != "n" && answer != "N"
		if newVal != globalCfg.Template.Trailer.IncludeModel {
			repoCfg.Template.Trailer.IncludeModel = newVal
			changed = true
		} else {
			repoCfg.Template.Trailer.IncludeModel = globalCfg.Template.Trailer.IncludeModel
		}
	} else {
		repoCfg.Template.Trailer.IncludeModel = globalCfg.Template.Trailer.IncludeModel
	}

	// AI mode
	fmt.Printf("  AI detection mode (auto/manual/off) [%s]: ", globalCfg.AI.Mode)
	if answer := readLine(reader); answer != "" && answer != globalCfg.AI.Mode {
		repoCfg.AI.Mode = answer
		changed = true
	} else {
		repoCfg.AI.Mode = globalCfg.AI.Mode
	}

	// Confidence
	fmt.Printf("  Detection confidence threshold [%.1f]: ", globalCfg.AI.ConfidenceThreshold)
	if answer := readLine(reader); answer != "" {
		var threshold float64
		if _, err := fmt.Sscanf(answer, "%f", &threshold); err == nil && threshold != globalCfg.AI.ConfidenceThreshold {
			repoCfg.AI.ConfidenceThreshold = threshold
			changed = true
		} else {
			repoCfg.AI.ConfidenceThreshold = globalCfg.AI.ConfidenceThreshold
		}
	} else {
		repoCfg.AI.ConfidenceThreshold = globalCfg.AI.ConfidenceThreshold
	}

	// Override tool — always attribute commits to a specific AI tool
	currentOverride := globalCfg.AI.OverrideTool
	if currentOverride == "" {
		currentOverride = "none"
	}
	fmt.Printf("  Always attribute to AI tool (tool name, or 'none') [%s]: ", currentOverride)
	if answer := readLine(reader); answer != "" {
		if answer == "none" {
			repoCfg.AI.OverrideTool = ""
		} else if answer != globalCfg.AI.OverrideTool {
			repoCfg.AI.OverrideTool = answer
			changed = true
		}
	}

	// Journal
	journalDefault := "y"
	if !globalCfg.Journal.Enabled {
		journalDefault = "n"
	}
	fmt.Printf("  Enable commit journal? (y/n) [%s]: ", journalDefault)
	if answer := readLine(reader); answer != "" {
		newVal := answer != "n" && answer != "N"
		if newVal != globalCfg.Journal.Enabled {
			repoCfg.Journal.Enabled = newVal
			changed = true
		} else {
			repoCfg.Journal.Enabled = globalCfg.Journal.Enabled
		}
	} else {
		repoCfg.Journal.Enabled = globalCfg.Journal.Enabled
	}

	if !changed {
		fmt.Println()
		ui.Info("No changes — repo config not created (global settings apply)")
	} else {
		if err := config.SaveLocal(repoCfg); err != nil {
			return fmt.Errorf("failed to save repo config: %w", err)
		}
		ui.Success("Repo config saved to %s", config.RepoConfigPath())
	}

	// Enable hooks if needed
	installHooksIfNeeded(reader)

	// Ensure .gitx-commit.toml is globally gitignored
	ensureGlobalGitignore()

	fmt.Printf("\n  Setup complete! Run 'git xcommit config show' to verify.\n\n")
	return nil
}

func installHooksIfNeeded(reader *bufio.Reader) {
	repoPath := ""
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err == nil {
		repoPath = strings.TrimSpace(string(out))
	}

	if repoPath != "" && !hook.IsInstalled(repoPath) {
		fmt.Printf("\n  Enable gitx-commit hooks in this repo? (y/n) [y]: ")
		if answer := readLine(reader); answer != "n" && answer != "N" {
			if err := hook.Install(repoPath); err != nil {
				fmt.Printf("  ✗ Failed to install hooks: %v\n", err)
			} else {
				ui.Success("Hooks installed")
			}
		}
	}
}

// ensureGlobalGitignore adds .gitx-commit.toml to the global gitignore
// so repo-level configs are never accidentally committed.
func ensureGlobalGitignore() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	// Get or set the global excludes file
	globalIgnorePath := ""
	out, err := exec.Command("git", "config", "--global", "core.excludesFile").Output()
	if err == nil {
		globalIgnorePath = strings.TrimSpace(string(out))
	}

	if globalIgnorePath == "" {
		globalIgnorePath = filepath.Join(home, ".gitignore")
		exec.Command("git", "config", "--global", "core.excludesFile", globalIgnorePath).Run()
	}

	// Expand ~ if present
	if strings.HasPrefix(globalIgnorePath, "~") {
		globalIgnorePath = filepath.Join(home, globalIgnorePath[1:])
	}

	// Check if .gitx-commit.toml is already in the file
	existing, _ := os.ReadFile(globalIgnorePath)
	if strings.Contains(string(existing), ".gitx-commit.toml") {
		return // already there
	}

	// Append it
	f, err := os.OpenFile(globalIgnorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	// Add newline if file doesn't end with one
	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		f.WriteString("\n")
	}

	f.WriteString("# gitx-commit repo config (personal, not committed)\n")
	f.WriteString(".gitx-commit.toml\n")

	ui.Success(".gitx-commit.toml added to global gitignore (%s)", globalIgnorePath)
}

// readLine reads a line from stdin, trimming whitespace.
func readLine(reader *bufio.Reader) string {
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func init() {
	initCmd.Flags().BoolVar(&initLocal, "local", false, "create repo-level config override (.gitx-commit.toml)")
	rootCmd.AddCommand(initCmd)
}
