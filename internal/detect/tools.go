package detect

// ToolDef describes how to detect a specific AI coding tool.
// Each field is a detection vector — the Detect() function
// checks them in order of cost (env vars → repo files → home files).
type ToolDef struct {
	Name       string   // display name shown in trailers
	EnvVars    []string // env vars that prove the tool is actively running
	RepoFiles  []string // files/dirs in the repo root indicating tool usage
	HomeFiles  []string // files/dirs in ~ indicating the tool is installed
	VersionEnv string   // env var that holds the tool's version (optional)

	// Model detection — how to find which LLM the tool is using
	ModelEnv  string // env var holding the model name (e.g., GEMINI_MODEL)
	ModelFile string // config file path relative to ~ (e.g., ".kiro/settings/cli.json")
	ModelKey  string // JSON key to extract from ModelFile (e.g., "chat.defaultModel")
}

// knownTools is the registry of all AI tools we can detect.
// Ordered roughly by detection reliability (highest first).
//
// To add a new tool: append a ToolDef here. No other code changes needed —
// Detect() iterates this list automatically.
var knownTools = []ToolDef{
	{
		Name:       "Kiro",
		EnvVars:    []string{"KIRO_SESSION_ID", "KIRO_VERSION"},
		RepoFiles:  []string{".kiro"},
		HomeFiles:  []string{".kiro"},
		VersionEnv: "KIRO_VERSION",
		ModelFile:  ".kiro/settings/cli.json",
		ModelKey:   "chat.defaultModel",
	},
	{
		Name:       "Claude Code",
		EnvVars:    []string{"CLAUDECODE"},
		RepoFiles:  []string{"CLAUDE.md", "AGENTS.md", ".claude"},
		HomeFiles:  []string{".claude"},
		VersionEnv: "",
		// Claude Code doesn't store model in an easily parseable location
		// Falls back to GITX_AI_MODEL env var or config
	},
	{
		Name:       "Cursor",
		EnvVars:    []string{"CURSOR_AGENT", "CURSOR_TRACE_ID"},
		RepoFiles:  []string{".cursor", ".cursorrules"},
		HomeFiles:  []string{".cursor"},
		VersionEnv: "",
		// Cursor stores model in internal DB, not easily readable
	},
	{
		Name:      "Gemini CLI",
		EnvVars:   []string{"GEMINI_API_KEY", "GEMINI_MODEL"},
		RepoFiles: []string{"GEMINI.md", ".gemini", ".geminiignore"},
		HomeFiles: []string{".gemini"},
		ModelEnv:  "GEMINI_MODEL",
	},
	{
		Name:      "Aider",
		EnvVars:   []string{},
		RepoFiles: []string{".aider.chat.history.md", ".aider.conf.yml", ".aider.input.history"},
		HomeFiles: []string{},
		// Aider stores model in .aider.conf.yml — could parse YAML but
		// keeping it simple for v1. Falls back to GITX_AI_MODEL.
	},
	{
		Name:      "GitHub Copilot",
		EnvVars:   []string{},
		RepoFiles: []string{".github/copilot-instructions.md"},
		HomeFiles: []string{".config/github-copilot"},
		// Model is server-side, not detectable locally
	},
	{
		Name:      "Windsurf",
		EnvVars:   []string{},
		RepoFiles: []string{".windsurf"},
		HomeFiles: []string{".windsurf", ".codeium"},
	},
	{
		Name:      "Continue",
		EnvVars:   []string{},
		RepoFiles: []string{".continue"},
		HomeFiles: []string{".continue"},
	},
	{
		Name:      "Tabnine",
		EnvVars:   []string{},
		RepoFiles: []string{},
		HomeFiles: []string{".tabnine"},
	},
}
