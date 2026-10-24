package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Malay-dev/gitx-commit/internal/config"
	"github.com/Malay-dev/gitx-commit/internal/hook"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive setup wizard for gitx-commit",
	Long: `Walks you through configuring gitx-commit for the first time.
Sets up the global config file, installs hooks in the current
repository, and shows detected AI tools.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInit()
	},
}

func runInit() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Println("  gitx-commit setup")
	fmt.Println("  ─────────────────")
	fmt.Println()

	cfg := config.DefaultConfig()

	// 1. Template format
	fmt.Printf("  Commit template format (conventional/angular/none) [conventional]: ")
	if answer := readLine(reader); answer != "" {
		cfg.Template.Format = answer
	}

	// 2. Trailer format
	fmt.Printf("  Trailer key (Assisted-by/Co-authored-by/Drafted-With) [Assisted-by]: ")
	if answer := readLine(reader); answer != "" {
		cfg.Template.Trailer.Format = answer
	}

	// 3. Include model in trailer
	fmt.Printf("  Include AI model in trailer? (y/n) [y]: ")
	if answer := readLine(reader); answer == "n" || answer == "N" {
		cfg.Template.Trailer.IncludeModel = false
	}

	// 4. AI detection mode
	fmt.Printf("  AI detection mode (auto/manual/off) [auto]: ")
	if answer := readLine(reader); answer != "" {
		cfg.AI.Mode = answer
	}

	// 5. Confidence threshold
	fmt.Printf("  Detection confidence threshold (0.0-1.0) [0.7]: ")
	if answer := readLine(reader); answer != "" {
		var threshold float64
		if _, err := fmt.Sscanf(answer, "%f", &threshold); err == nil {
			cfg.AI.ConfidenceThreshold = threshold
		}
	}

	// 6. Journal
	fmt.Printf("  Enable commit journal? (y/n) [y]: ")
	if answer := readLine(reader); answer == "n" || answer == "N" {
		cfg.Journal.Enabled = false
	}

	// Save config
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	path, _ := config.GlobalConfigPath()
	fmt.Printf("\n  ✓ Config saved to %s\n", path)

	// 7. Enable in current repo?
	repoPath := ""
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err == nil {
		repoPath = strings.TrimSpace(string(out))
	}

	if repoPath != "" && !hook.IsInstalled(repoPath) {
		fmt.Printf("\n  Enable gitx-commit in %s? (y/n) [y]: ", repoPath)
		if answer := readLine(reader); answer != "n" && answer != "N" {
			if err := hook.Install(repoPath); err != nil {
				fmt.Printf("  ✗ Failed to install hooks: %v\n", err)
			} else {
				fmt.Printf("  ✓ Hooks installed\n")
			}
		}
	}

	fmt.Printf("\n  Setup complete! Run 'git xcommit status' to verify.\n\n")
	return nil
}

// readLine reads a line from stdin, trimming whitespace.
func readLine(reader *bufio.Reader) string {
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func init() {
	rootCmd.AddCommand(initCmd)
}
