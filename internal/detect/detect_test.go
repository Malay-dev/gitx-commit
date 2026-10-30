package detect

import (
	"os"
	"path/filepath"
	"testing"
)

// clearAllToolEnvVars neutralizes env vars from all known tools
// so tests only see what they explicitly set.
func clearAllToolEnvVars(t *testing.T) {
	t.Helper()
	for _, tool := range knownTools {
		for _, env := range tool.EnvVars {
			t.Setenv(env, "")
		}
		if tool.VersionEnv != "" {
			t.Setenv(tool.VersionEnv, "")
		}
	}
	t.Setenv("GITX_AI_TOOL", "")
}

func TestDetectNoSignals(t *testing.T) {
	clearAllToolEnvVars(t)
	repoPath := t.TempDir()

	result := Detect(repoPath, nil)

	// On a machine with AI tools installed in home dir, this may still
	// detect something via home_file. On a clean CI machine, result is nil.
	if result != nil {
		t.Logf("detected %s via %s (confidence %.2f) — likely from home dir artifacts",
			result.Tool, result.Method, result.Confidence)
	}
}

func TestDetectExplicitOverride(t *testing.T) {
	repoPath := t.TempDir()

	// Override takes highest precedence — no need to clear other vars
	t.Setenv("GITX_AI_TOOL", "MyCustomTool")

	result := Detect(repoPath, nil)

	if result == nil {
		t.Fatal("expected detection result with override")
	}
	if result.Tool != "MyCustomTool" {
		t.Errorf("expected tool 'MyCustomTool', got '%s'", result.Tool)
	}
	if result.Confidence != 1.0 {
		t.Errorf("override should have confidence 1.0, got %.2f", result.Confidence)
	}
	if result.Method != "override" {
		t.Errorf("expected method 'override', got '%s'", result.Method)
	}
}

func TestDetectOverrideNone(t *testing.T) {
	repoPath := t.TempDir()
	t.Setenv("GITX_AI_TOOL", "none")

	result := Detect(repoPath, nil)

	if result != nil {
		t.Errorf("GITX_AI_TOOL=none should return nil, got %+v", result)
	}
}

func TestDetectKiroEnvVar(t *testing.T) {
	clearAllToolEnvVars(t)
	repoPath := t.TempDir()

	// Simulate Kiro being active
	t.Setenv("KIRO_SESSION_ID", "test-session-123")
	t.Setenv("KIRO_VERSION", "2.32.0")

	result := Detect(repoPath, nil)

	if result == nil {
		t.Fatal("expected Kiro detection")
	}
	if result.Tool != "Kiro" {
		t.Errorf("expected tool 'Kiro', got '%s'", result.Tool)
	}
	if result.Confidence != 0.95 {
		t.Errorf("env detection should have confidence 0.95, got %.2f", result.Confidence)
	}
	if result.Version != "2.32.0" {
		t.Errorf("expected version '2.32.0', got '%s'", result.Version)
	}
	if result.Method != "env" {
		t.Errorf("expected method 'env', got '%s'", result.Method)
	}
}

func TestDetectClaudeCodeEnv(t *testing.T) {
	clearAllToolEnvVars(t)
	repoPath := t.TempDir()

	t.Setenv("CLAUDECODE", "1")

	result := Detect(repoPath, nil)

	if result == nil {
		t.Fatal("expected Claude Code detection")
	}
	if result.Tool != "Claude Code" {
		t.Errorf("expected 'Claude Code', got '%s'", result.Tool)
	}
}

func TestDetectCursorEnv(t *testing.T) {
	clearAllToolEnvVars(t)
	repoPath := t.TempDir()

	t.Setenv("CURSOR_AGENT", "1")

	result := Detect(repoPath, nil)

	if result == nil {
		t.Fatal("expected Cursor detection")
	}
	if result.Tool != "Cursor" {
		t.Errorf("expected 'Cursor', got '%s'", result.Tool)
	}
}

func TestDetectRepoFile(t *testing.T) {
	clearAllToolEnvVars(t)
	repoPath := t.TempDir()

	// Create a CLAUDE.md file in the repo
	os.WriteFile(filepath.Join(repoPath, "CLAUDE.md"), []byte("# Claude context"), 0644)

	result := Detect(repoPath, nil)

	if result == nil {
		t.Fatal("expected detection from CLAUDE.md")
	}
	if result.Tool != "Claude Code" {
		t.Errorf("expected 'Claude Code', got '%s'", result.Tool)
	}
	if result.Confidence != 0.50 {
		t.Errorf("repo file detection should have confidence 0.50, got %.2f", result.Confidence)
	}
	if result.Method != "repo_file" {
		t.Errorf("expected method 'repo_file', got '%s'", result.Method)
	}
}

func TestDetectRepoDir(t *testing.T) {
	clearAllToolEnvVars(t)
	repoPath := t.TempDir()

	os.MkdirAll(filepath.Join(repoPath, ".cursor"), 0755)

	result := Detect(repoPath, nil)

	if result == nil {
		t.Fatal("expected detection from .cursor dir")
	}
	if result.Tool != "Cursor" {
		t.Errorf("expected 'Cursor', got '%s'", result.Tool)
	}
}

func TestDetectAiderFiles(t *testing.T) {
	clearAllToolEnvVars(t)
	repoPath := t.TempDir()

	os.WriteFile(filepath.Join(repoPath, ".aider.chat.history.md"), []byte("chat log"), 0644)

	result := Detect(repoPath, nil)

	if result == nil {
		t.Fatal("expected detection from aider files")
	}
	if result.Tool != "Aider" {
		t.Errorf("expected 'Aider', got '%s'", result.Tool)
	}
}

func TestDetectGeminiFiles(t *testing.T) {
	clearAllToolEnvVars(t)
	repoPath := t.TempDir()

	os.WriteFile(filepath.Join(repoPath, "GEMINI.md"), []byte("# Gemini context"), 0644)

	result := Detect(repoPath, nil)

	if result == nil {
		t.Fatal("expected detection from GEMINI.md")
	}
	if result.Tool != "Gemini CLI" {
		t.Errorf("expected 'Gemini CLI', got '%s'", result.Tool)
	}
}

func TestDetectEnvWinsOverRepoFile(t *testing.T) {
	clearAllToolEnvVars(t)
	repoPath := t.TempDir()

	// Kiro env var active + Claude repo file present
	t.Setenv("KIRO_SESSION_ID", "test-session")
	os.WriteFile(filepath.Join(repoPath, "CLAUDE.md"), []byte("context"), 0644)

	result := Detect(repoPath, nil)

	if result == nil {
		t.Fatal("expected detection")
	}
	// Env (0.95) should beat repo file (0.50)
	if result.Tool != "Kiro" {
		t.Errorf("env detection should win over repo file, got '%s'", result.Tool)
	}
	if result.Confidence != 0.95 {
		t.Errorf("expected confidence 0.95, got %.2f", result.Confidence)
	}
}

func TestDetectAll(t *testing.T) {
	clearAllToolEnvVars(t)
	repoPath := t.TempDir()

	// Set up multiple signals
	t.Setenv("KIRO_SESSION_ID", "test")
	os.WriteFile(filepath.Join(repoPath, "CLAUDE.md"), []byte("context"), 0644)
	os.MkdirAll(filepath.Join(repoPath, ".cursor"), 0755)

	results := DetectAll(repoPath)

	// Should find at least Kiro (env) + Claude (repo) + Cursor (repo)
	if len(results) < 3 {
		t.Errorf("expected at least 3 detections, got %d", len(results))
	}
	for _, r := range results {
		t.Logf("  found: %s (%.2f via %s)", r.Tool, r.Confidence, r.Method)
	}
}

func TestDetectAllEmpty(t *testing.T) {
	clearAllToolEnvVars(t)
	repoPath := t.TempDir()

	results := DetectAll(repoPath)

	// May still find tools via home files — just log what we see
	for _, r := range results {
		t.Logf("  found: %s (%.2f via %s)", r.Tool, r.Confidence, r.Method)
	}
}

func TestDetectOverrideTakesPrecedenceOverEverything(t *testing.T) {
	repoPath := t.TempDir()

	// Active Kiro env + Claude repo file + explicit override
	t.Setenv("KIRO_SESSION_ID", "test")
	os.WriteFile(filepath.Join(repoPath, "CLAUDE.md"), []byte("context"), 0644)
	t.Setenv("GITX_AI_TOOL", "CustomAgent")

	result := Detect(repoPath, nil)

	if result == nil {
		t.Fatal("expected detection")
	}
	if result.Tool != "CustomAgent" {
		t.Errorf("override should beat everything, got '%s'", result.Tool)
	}
	if result.Confidence != 1.0 {
		t.Errorf("override confidence should be 1.0, got %.2f", result.Confidence)
	}
}

// --- DetectOptions tests ---

func TestDetectWithConfigOverrideTool(t *testing.T) {
	clearAllToolEnvVars(t)
	repoPath := t.TempDir()

	opts := &DetectOptions{
		OverrideTool:  "CustomTool",
		OverrideModel: "custom-model-v1",
	}

	result := Detect(repoPath, opts)
	if result == nil {
		t.Fatal("expected detection with config override")
	}
	if result.Tool != "CustomTool" {
		t.Errorf("expected 'CustomTool', got '%s'", result.Tool)
	}
	if result.Model != "custom-model-v1" {
		t.Errorf("expected 'custom-model-v1', got '%s'", result.Model)
	}
	if result.Confidence != 1.0 {
		t.Errorf("config override should have confidence 1.0, got %.2f", result.Confidence)
	}
	if result.Method != "override" {
		t.Errorf("expected method 'override', got '%s'", result.Method)
	}
}

func TestDetectEnvOverrideBbeatsConfigOverride(t *testing.T) {
	clearAllToolEnvVars(t)
	repoPath := t.TempDir()

	// Env override set
	t.Setenv("GITX_AI_TOOL", "EnvTool")
	t.Setenv("GITX_AI_MODEL", "env-model")

	// Config override also set
	opts := &DetectOptions{
		OverrideTool:  "ConfigTool",
		OverrideModel: "config-model",
	}

	result := Detect(repoPath, opts)
	if result == nil {
		t.Fatal("expected detection")
	}
	// Env should win over config
	if result.Tool != "EnvTool" {
		t.Errorf("env should beat config override, got '%s'", result.Tool)
	}
	if result.Model != "env-model" {
		t.Errorf("env model should win, got '%s'", result.Model)
	}
}

func TestDetectConfigOverrideToolWithEnvModel(t *testing.T) {
	clearAllToolEnvVars(t)
	repoPath := t.TempDir()

	// Only GITX_AI_MODEL env set, tool comes from config
	t.Setenv("GITX_AI_MODEL", "env-model")

	opts := &DetectOptions{
		OverrideTool: "ConfigTool",
	}

	result := Detect(repoPath, opts)
	if result == nil {
		t.Fatal("expected detection")
	}
	if result.Tool != "ConfigTool" {
		t.Errorf("expected 'ConfigTool', got '%s'", result.Tool)
	}
	if result.Model != "env-model" {
		t.Errorf("env model should be picked up, got '%s'", result.Model)
	}
}

func TestDetectNilOpts(t *testing.T) {
	clearAllToolEnvVars(t)
	repoPath := t.TempDir()

	// nil opts should work fine — just auto-detection
	result := Detect(repoPath, nil)
	// May or may not detect something from home files
	if result != nil {
		t.Logf("detected %s via %s", result.Tool, result.Method)
	}
}
