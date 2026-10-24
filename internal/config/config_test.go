package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// setTestHome sets the home directory for tests across all platforms.
// Windows uses USERPROFILE, Unix uses HOME.
func setTestHome(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("HOME", dir)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", dir)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	if !cfg.General.Enabled {
		t.Error("General.Enabled should default to true")
	}
	if cfg.Template.Format != "conventional" {
		t.Errorf("Template.Format should be 'conventional', got '%s'", cfg.Template.Format)
	}
	if cfg.Template.Trailer.Format != "Assisted-by" {
		t.Errorf("Trailer.Format should be 'Assisted-by', got '%s'", cfg.Template.Trailer.Format)
	}
	if !cfg.Template.Trailer.IncludeModel {
		t.Error("Trailer.IncludeModel should default to true")
	}
	if !cfg.AI.Enabled {
		t.Error("AI.Enabled should default to true")
	}
	if cfg.AI.Mode != "auto" {
		t.Errorf("AI.Mode should be 'auto', got '%s'", cfg.AI.Mode)
	}
	if cfg.AI.ConfidenceThreshold != 0.7 {
		t.Errorf("AI.ConfidenceThreshold should be 0.7, got %f", cfg.AI.ConfidenceThreshold)
	}
	if !cfg.Journal.Enabled {
		t.Error("Journal.Enabled should default to true")
	}
	if cfg.Journal.DBPath != "~/.gitx/commits.db" {
		t.Errorf("Journal.DBPath should be '~/.gitx/commits.db', got '%s'", cfg.Journal.DBPath)
	}
}

func TestDefaultConfigReturnsNewInstance(t *testing.T) {
	cfg1 := DefaultConfig()
	cfg2 := DefaultConfig()

	cfg1.General.Enabled = false
	cfg1.Template.Format = "angular"

	if !cfg2.General.Enabled {
		t.Error("modifying cfg1 should not affect cfg2")
	}
	if cfg2.Template.Format != "conventional" {
		t.Error("modifying cfg1 should not affect cfg2")
	}
}

func TestDefaultConfigZeroValueSafety(t *testing.T) {
	cfg := DefaultConfig()

	boolChecks := []struct {
		name string
		ok   bool
	}{
		{"General.Enabled", cfg.General.Enabled},
		{"AI.Enabled", cfg.AI.Enabled},
		{"Journal.Enabled", cfg.Journal.Enabled},
		{"Trailer.IncludeModel", cfg.Template.Trailer.IncludeModel},
	}
	for _, c := range boolChecks {
		if !c.ok {
			t.Errorf("%s is false — likely a missing default", c.name)
		}
	}

	stringChecks := []struct {
		name  string
		value string
	}{
		{"Template.Format", cfg.Template.Format},
		{"Trailer.Format", cfg.Template.Trailer.Format},
		{"AI.Mode", cfg.AI.Mode},
		{"Journal.DBPath", cfg.Journal.DBPath},
	}
	for _, c := range stringChecks {
		if c.value == "" {
			t.Errorf("%s is empty — likely a missing default", c.name)
		}
	}

	if cfg.AI.ConfidenceThreshold == 0.0 {
		t.Error("AI.ConfidenceThreshold is 0.0 — likely a missing default")
	}
}

func TestLoadReturnsDefaultsWhenNoConfigFile(t *testing.T) {
	origDir, _ := os.Getwd()
	os.Chdir(t.TempDir())
	defer os.Chdir(origDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if cfg.Template.Format != "conventional" {
		t.Errorf("should fall back to defaults, got format '%s'", cfg.Template.Format)
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmpHome := t.TempDir()
	setTestHome(t, tmpHome)

	cfg := DefaultConfig()
	cfg.Template.Format = "angular"
	cfg.AI.ConfidenceThreshold = 0.5

	err := Save(cfg)
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Verify file exists
	path := filepath.Join(tmpHome, ".gitx", "commit-config.toml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if loaded.Template.Format != "angular" {
		t.Errorf("expected format 'angular', got '%s'", loaded.Template.Format)
	}
	if loaded.AI.ConfidenceThreshold != 0.5 {
		t.Errorf("expected threshold 0.5, got %f", loaded.AI.ConfidenceThreshold)
	}
}

func TestConfigDir(t *testing.T) {
	tmpHome := t.TempDir()
	setTestHome(t, tmpHome)

	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() failed: %v", err)
	}

	expected := filepath.Join(tmpHome, ".gitx")
	if dir != expected {
		t.Errorf("expected '%s', got '%s'", expected, dir)
	}

	if _, err := os.Stat(dir); err != nil {
		t.Error("ConfigDir should create the directory")
	}
}

func TestGlobalConfigPath(t *testing.T) {
	tmpHome := t.TempDir()
	setTestHome(t, tmpHome)

	path, err := GlobalConfigPath()
	if err != nil {
		t.Fatalf("GlobalConfigPath() failed: %v", err)
	}

	expected := filepath.Join(tmpHome, ".gitx", "commit-config.toml")
	if path != expected {
		t.Errorf("expected '%s', got '%s'", expected, path)
	}
}

func TestRepoConfigPath(t *testing.T) {
	path := RepoConfigPath()
	if path != ".gitx-commit.toml" {
		t.Errorf("expected '.gitx-commit.toml', got '%s'", path)
	}
}

func TestJournalDBPathDefault(t *testing.T) {
	cfg := DefaultConfig()
	path := JournalDBPath(cfg)
	if path != "~/.gitx/commits.db" {
		t.Errorf("expected default path, got '%s'", path)
	}
}

func TestJournalDBPathOverride(t *testing.T) {
	t.Setenv("GITX_JOURNAL_PATH", "/tmp/test.db")
	cfg := DefaultConfig()
	path := JournalDBPath(cfg)
	if path != "/tmp/test.db" {
		t.Errorf("expected override path, got '%s'", path)
	}
}

func TestLoadRepoOverridesGlobal(t *testing.T) {
	tmpHome := t.TempDir()
	setTestHome(t, tmpHome)

	cfg := DefaultConfig()
	cfg.Template.Format = "angular"
	Save(cfg)

	tmpRepo := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpRepo)
	defer os.Chdir(origDir)

	repoConfig := []byte("[template]\nformat = 'custom'\n")
	os.WriteFile(".gitx-commit.toml", repoConfig, 0644)

	loaded, _ := Load()
	if loaded.Template.Format != "custom" {
		t.Errorf("repo config should override global, got '%s'", loaded.Template.Format)
	}
}
