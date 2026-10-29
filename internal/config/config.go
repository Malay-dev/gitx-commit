package config

import (
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// Config is the top-level configuration for gitx-commit.
// Maps to the TOML config file structure.
type Config struct {
	General  GeneralConfig  `toml:"general"`
	Template TemplateConfig `toml:"template"`
	AI       AIConfig       `toml:"ai"`
	Journal  JournalConfig  `toml:"journal"`
}

// GeneralConfig controls whether gitx-commit is active.
type GeneralConfig struct {
	Enabled bool `toml:"enabled"`
}

// TemplateConfig controls commit message formatting.
type TemplateConfig struct {
	Format         string        `toml:"format"`
	CustomTemplate string        `toml:"custom_template"`
	Trailer        TrailerConfig `toml:"trailer"`
}

// TrailerConfig controls the AI metadata trailers appended to commits.
type TrailerConfig struct {
	Format       string `toml:"format"`
	IncludeModel bool   `toml:"include_model"`
}

// AIConfig controls automatic AI tool detection.
type AIConfig struct {
	Enabled             bool    `toml:"enabled"`
	Mode                string  `toml:"mode"`
	OverrideTool        string  `toml:"override_tool"`
	OverrideModel       string  `toml:"override_model"`
	ConfidenceThreshold float64 `toml:"confidence_threshold"`
}

// JournalConfig controls the local commit journal.
type JournalConfig struct {
	Enabled bool   `toml:"enabled"`
	DBPath  string `toml:"db_path"`
}

// DefaultConfig returns a Config with sensible defaults.
// The tool works out of the box without any config file.
func DefaultConfig() *Config {
	return &Config{
		General: GeneralConfig{
			Enabled: true,
		},
		Template: TemplateConfig{
			Format:         "conventional",
			CustomTemplate: "",
			Trailer: TrailerConfig{
				Format:       "Assisted-by",
				IncludeModel: true,
			},
		},
		AI: AIConfig{
			Enabled:             true,
			Mode:                "auto",
			OverrideTool:        "",
			ConfidenceThreshold: 0.7,
		},
		Journal: JournalConfig{
			Enabled: true,
			DBPath:  "~/.gitx/commits.db",
		},
	}
}

// JournalDBPath returns the journal database path, respecting
// the GITX_JOURNAL_PATH env var override. This allows tests and
// CI to use a separate database without touching the user's journal.
func JournalDBPath(cfg *Config) string {
	if override := os.Getenv("GITX_JOURNAL_PATH"); override != "" {
		return override
	}
	return cfg.Journal.DBPath
}

// ConfigDir returns the gitx config directory (~/.gitx/).
// Creates it if it doesn't exist.
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".gitx")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

// GlobalConfigPath returns the path to the global config file.
func GlobalConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "commit-config.toml"), nil
}

// RepoConfigPath returns the path to the repo-level config file
// in the current directory. Does not check if it exists.
func RepoConfigPath() string {
	return ".gitx-commit.toml"
}

// Load reads config from TOML files with layered precedence:
//  1. Start with defaults
//  2. Override with global config (~/.gitx/commit-config.toml)
//  3. Override with repo config (.gitx-commit.toml in current dir)
//
// Missing files are silently skipped — the tool works without any config.
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Layer 1: global config
	globalPath, err := GlobalConfigPath()
	if err == nil {
		loadFromFile(globalPath, cfg)
	}

	// Layer 2: repo-level config (overrides global)
	loadFromFile(RepoConfigPath(), cfg)

	return cfg, nil
}

// Save writes the config to the global config file.
func Save(cfg *Config) error {
	path, err := GlobalConfigPath()
	if err != nil {
		return err
	}

	data, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// SaveLocal writes the config to the repo-level config file (.gitx-commit.toml).
func SaveLocal(cfg *Config) error {
	data, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(RepoConfigPath(), data, 0644)
}

// loadFromFile reads a TOML file and unmarshals it into the config.
// Silently returns if the file doesn't exist — this is expected.
func loadFromFile(path string, cfg *Config) {
	data, err := os.ReadFile(path)
	if err != nil {
		return // file doesn't exist — that's fine
	}
	toml.Unmarshal(data, cfg)
}
