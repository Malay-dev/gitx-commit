package detect

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// DetectionResult holds the outcome of an AI tool detection.
// Confidence ranges from 0.0 (no signal) to 1.0 (certain).
type DetectionResult struct {
	Tool       string  // display name: "Kiro", "Claude Code", etc.
	Version    string  // tool version if detectable, empty otherwise
	Model      string  // LLM model name: "claude-opus-4.6", "gemini-3-pro", etc.
	Confidence float64 // 0.0–1.0 how sure we are
	Method     string  // how we detected it: "env", "repo_file", "home_file", "override"
}

// Detect runs the layered detection algorithm and returns the
// highest-confidence match, or nil if no AI tool is detected.
//
// Detection order (cheapest → most expensive):
//  1. Explicit override via GITX_AI_TOOL env var
//  2. Environment variables (proves active agent in this shell)
//  3. Repo config files (proves tool used in this repo)
//  4. Home directory files (proves tool is installed — weakest signal)
//
// After detection, the model is resolved from:
//  1. GITX_AI_MODEL env var (explicit override)
//  2. Tool-specific model env var (e.g., GEMINI_MODEL)
//  3. Tool-specific config file (e.g., ~/.kiro/settings/cli.json)
func Detect(repoPath string) *DetectionResult {
	// Layer 0: explicit override — highest precedence
	if override := os.Getenv("GITX_AI_TOOL"); override != "" {
		if override == "none" {
			return nil
		}
		result := &DetectionResult{
			Tool:       override,
			Confidence: 1.0,
			Method:     "override",
		}
		// Check for explicit model override
		result.Model = os.Getenv("GITX_AI_MODEL")
		return result
	}

	var best *DetectionResult

	for _, tool := range knownTools {
		// Layer 1: env vars — strongest automatic signal
		if result := checkEnvVars(tool); result != nil {
			resolveModel(result, tool)
			if best == nil || result.Confidence > best.Confidence {
				best = result
			}
			continue
		}

		// Layer 2: repo config files
		if result := checkRepoFiles(tool, repoPath); result != nil {
			resolveModel(result, tool)
			if best == nil || result.Confidence > best.Confidence {
				best = result
			}
			continue
		}

		// Layer 3: home directory files (weakest signal)
		if result := checkHomeFiles(tool); result != nil {
			resolveModel(result, tool)
			if best == nil || result.Confidence > best.Confidence {
				best = result
			}
		}
	}

	return best
}

// DetectAll returns all detected tools, not just the best one.
// Useful for the "git xcommit detect" command to show everything found.
func DetectAll(repoPath string) []DetectionResult {
	var results []DetectionResult

	for _, tool := range knownTools {
		if result := checkEnvVars(tool); result != nil {
			resolveModel(result, tool)
			results = append(results, *result)
			continue
		}
		if result := checkRepoFiles(tool, repoPath); result != nil {
			resolveModel(result, tool)
			results = append(results, *result)
			continue
		}
		if result := checkHomeFiles(tool); result != nil {
			resolveModel(result, tool)
			results = append(results, *result)
		}
	}

	return results
}

// resolveModel tries to find which LLM model the tool is using.
// Priority: GITX_AI_MODEL env → tool-specific env → tool config file
func resolveModel(result *DetectionResult, tool ToolDef) {
	// Priority 1: explicit override via env var
	if model := os.Getenv("GITX_AI_MODEL"); model != "" {
		result.Model = model
		return
	}

	// Priority 2: tool-specific model env var (e.g., GEMINI_MODEL)
	if tool.ModelEnv != "" {
		if model := os.Getenv(tool.ModelEnv); model != "" {
			result.Model = model
			return
		}
	}

	// Priority 3: read from tool's config file
	if tool.ModelFile != "" && tool.ModelKey != "" {
		if model := readModelFromJSON(tool.ModelFile, tool.ModelKey); model != "" {
			result.Model = model
			return
		}
	}
}

// readModelFromJSON reads a top-level string key from a JSON config file.
// The path is relative to the user's home directory.
func readModelFromJSON(relPath, key string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	fullPath := filepath.Join(home, relPath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return ""
	}

	// Parse as a flat map — works for simple top-level keys
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return ""
	}

	if val, ok := config[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}

	return ""
}

// checkEnvVars looks for tool-specific environment variables.
// If ANY of the tool's env vars are set, the tool is considered active.
// Confidence: 0.95 — env vars prove this shell session is that agent.
func checkEnvVars(tool ToolDef) *DetectionResult {
	for _, envVar := range tool.EnvVars {
		if value := os.Getenv(envVar); value != "" {
			result := &DetectionResult{
				Tool:       tool.Name,
				Confidence: 0.95,
				Method:     "env",
			}
			if tool.VersionEnv != "" {
				result.Version = os.Getenv(tool.VersionEnv)
			}
			return result
		}
	}
	return nil
}

// checkRepoFiles looks for tool-specific files/directories in the repo.
// Confidence: 0.50 — proves the tool was used in this repo,
// but not necessarily for this commit.
func checkRepoFiles(tool ToolDef, repoPath string) *DetectionResult {
	for _, file := range tool.RepoFiles {
		fullPath := filepath.Join(repoPath, file)
		if _, err := os.Stat(fullPath); err == nil {
			return &DetectionResult{
				Tool:       tool.Name,
				Confidence: 0.50,
				Method:     "repo_file",
			}
		}
	}
	return nil
}

// checkHomeFiles looks for tool artifacts in the user's home directory.
// Confidence: 0.30 — proves the tool is installed, but says nothing
// about whether it was used for this repo or commit.
func checkHomeFiles(tool ToolDef) *DetectionResult {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	for _, file := range tool.HomeFiles {
		fullPath := filepath.Join(home, file)
		if _, err := os.Stat(fullPath); err == nil {
			return &DetectionResult{
				Tool:       tool.Name,
				Confidence: 0.30,
				Method:     "home_file",
			}
		}
	}
	return nil
}
