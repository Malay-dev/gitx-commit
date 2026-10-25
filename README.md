<p align="center">
  <img src="gitx-commit.png" alt="gitx-commit"/>
</p>

<h1 align="center">gitx-commit</h1>

<p align="center">
  <a href="https://github.com/Malay-dev/gitx-commit/actions/workflows/ci.yml"><img src="https://github.com/Malay-dev/gitx-commit/actions/workflows/ci.yml/badge.svg" alt="CI" /></a>
  <a href="https://goreportcard.com/report/github.com/Malay-dev/gitx-commit"><img src="https://goreportcard.com/badge/github.com/Malay-dev/gitx-commit" alt="Go Report Card" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License: MIT" /></a>
</p>

<p align="center"><strong>Intelligent commit companion — AI provenance, templates, and commit journal for Git.</strong></p>

```
$ git commit -m "feat: add OAuth login"

feat: add OAuth login

Assisted-by: Kiro (claude-opus-4.6)
Commit-Mode: AI Assisted
```

gitx-commit augments `git commit` with automatic AI tool detection, commit message trailers, and a local commit journal across all your repositories. It hooks into your normal Git workflow transparently — no new commands to learn.

Part of the [gitx ecosystem](https://github.com/Malay-dev).

## What it does

Every time you commit, gitx-commit automatically:

1. **Detects which AI tool you're using** — Kiro, Claude Code, Cursor, Copilot, Gemini CLI, Aider, and more
2. **Adds an `Assisted-by` trailer** to your commit message with the tool and model name
3. **Tags the commit mode** — Manual, AI Assisted, or AI Generated
4. **Logs the commit** to a local SQLite journal for analytics and year-end reviews

```
$ git commit -m "feat: add OAuth login"

# Your commit message becomes:
feat: add OAuth login

Assisted-by: Kiro (claude-opus-4.6)
Commit-Mode: AI Assisted
```

Manual commits (no AI tool detected) get tagged too:

```
fix: typo in README

Commit-Mode: Manual
```

No cloud APIs. No code leaves your machine. Everything runs locally.

## Install

### From source (requires Go 1.23+)

```bash
go install github.com/Malay-dev/gitx-commit@latest
```

The binary is named `gitx-commit`. For Git subcommand integration, symlink it:

```bash
ln -sf $(go env GOPATH)/bin/gitx-commit ~/.local/bin/git-xcommit
```

### From release binaries

Download from [GitHub Releases](https://github.com/Malay-dev/gitx-commit/releases) and place `git-xcommit` on your PATH.

## Quick start

```bash
# Interactive setup (recommended for first time)
git xcommit init

# Or manually enable in a repo
git xcommit enable

# Verify
git xcommit status

# Make a commit — trailer is added automatically
git commit -m "feat: add login page"

# See what AI tools are detected
git xcommit detect
```

## Commands

| Command | Description |
|---------|-------------|
| `git xcommit init` | Interactive setup wizard |
| `git xcommit enable` | Install hooks in the current repository |
| `git xcommit disable` | Remove hooks from the current repository |
| `git xcommit status` | Show configuration, hooks, and detected tools |
| `git xcommit detect` | Show all detected AI coding tools |
| `git xcommit stats` | Show AI usage analytics from the journal |
| `git xcommit journal` | View recent commits from the journal |
| `git xcommit export` | Export journal to JSON, CSV, or Markdown |

### Enable options

```bash
git xcommit enable              # current repository only
git xcommit enable --global     # all repositories (via core.hooksPath)
git xcommit enable --force      # overwrite existing hooks from other tools
```

### Journal options

```bash
git xcommit journal                    # last 10 commits
git xcommit journal -n 20             # last 20 commits
git xcommit journal -r my-project     # filter by repo
git xcommit journal --since "last month"  # time filter
git xcommit journal --since "2026-01-01" --until "2026-06-30"  # date range
git xcommit journal --mode manual     # only manual commits
git xcommit journal --mode ai-assisted # only AI-assisted commits
git xcommit journal --tool Kiro       # filter by AI tool
git xcommit journal --search "auth"   # search in commit messages
git xcommit journal --backup          # create timestamped backup
git xcommit journal --purge-repo test # delete entries for a repo
```

### Export options

```bash
git xcommit export                     # export as JSON (default)
git xcommit export -f csv             # export as CSV
git xcommit export -f md              # export as Markdown
git xcommit export -o report.json     # custom output path
```

## AI tool detection

gitx-commit detects AI tools through a layered confidence system:

| Layer | Signal | Confidence | Speed |
|-------|--------|-----------|-------|
| Env vars | `KIRO_SESSION_ID`, `CLAUDECODE`, `CURSOR_AGENT`, `GEMINI_MODEL` | 0.95 | ~0ms |
| Repo files | `.kiro/`, `CLAUDE.md`, `.cursor/`, `GEMINI.md`, `.aider*` | 0.50 | ~1ms |
| Home files | `~/.kiro/`, `~/.claude/`, `~/.gemini/` | 0.30 | ~1ms |

Trailers are only added when confidence ≥ 0.70 (configurable).

### Supported tools

Kiro, Claude Code, Cursor, GitHub Copilot, Gemini CLI, Aider, Windsurf/Codeium, Continue.dev, Tabnine.

### Model detection

For tools that store model info locally, gitx-commit reads it automatically:
- **Kiro** → reads from `~/.kiro/settings/cli.json`
- **Gemini CLI** → reads `GEMINI_MODEL` env var

For other tools, set it manually:
```bash
export GITX_AI_MODEL="gpt-4"
```

### Override detection

```bash
export GITX_AI_TOOL="MyCustomTool"    # force a specific tool name
export GITX_AI_MODEL="my-model"       # force a specific model
export GITX_AI_TOOL="none"            # suppress all AI metadata
export GITX_COMMIT_MODE="AI Generated" # override commit mode
```

## Commit modes

Every commit gets a `Commit-Mode` trailer:

| Mode | Meaning | When assigned |
|------|---------|---------------|
| `Manual` | No AI involvement | No AI tool detected |
| `AI Assisted` | Human-directed, AI helped | AI tool detected above confidence threshold |
| `AI Generated` | AI wrote most of the code | User sets via `GITX_COMMIT_MODE` env var |

## Configuration

Config is loaded from two locations (repo overrides global):

| File | Scope |
|------|-------|
| `~/.gitx/commit-config.toml` | Global (all repos) |
| `.gitx-commit.toml` | Per-repository |

### Example config

```toml
[general]
enabled = true

[template]
format = "conventional"

[template.trailer]
format = "Assisted-by"
include_model = true

[ai]
enabled = true
mode = "auto"
confidence_threshold = 0.7

[journal]
enabled = true
db_path = "~/.gitx/commits.db"
```

Run `git xcommit init` for an interactive config wizard.

## Commit journal

Every commit is logged to `~/.gitx/commits.db` (SQLite). This enables:

- **Cross-repo analytics** — "What percentage of my code is AI-assisted?"
- **Year-end reviews** — Export all commits with `git xcommit export`
- **AI usage tracking** — Which tools and models you use most
- **Activity timeline** — "What did I work on last month?"

```bash
# Analytics
git xcommit stats

# Output:
# Total commits:       47
# Manual:              12 (25.5%)
# AI Assisted:         35 (74.5%)
#
# Top AI Tools:
#   Kiro                 28 commits
#   Claude Code           5 commits
```

## Environment variables

| Variable | Description |
|----------|-------------|
| `GITX_AI_TOOL` | Override detected tool name (or `none` to suppress) |
| `GITX_AI_MODEL` | Override detected model name |
| `GITX_COMMIT_MODE` | Override commit mode (Manual, AI Assisted, AI Generated) |
| `GITX_JOURNAL_PATH` | Override journal database path |

## How it works

gitx-commit uses Git's standard hook mechanism:

```
git commit -m "feat: add login"
    │
    ├─ prepare-commit-msg hook
    │   ├─ Detect AI tool → Add Assisted-by trailer
    │   └─ Classify mode → Add Commit-Mode trailer
    │
    ├─ Git creates the commit
    │
    └─ post-commit hook
        └─ Log commit metadata to SQLite journal
```

No magic — just Git hooks, file I/O, and SQLite.

## Development

```bash
# Build
make build

# Test
make test

# Coverage
make cover-html

# Lint
make vet

# Vulnerability scan
make vuln

# Everything
make all
```

## License

MIT — see [LICENSE](LICENSE).

## Part of the gitx ecosystem

- [gitx-profile](https://github.com/Malay-dev/gitx-profile) — Manage multiple Git identities
- **gitx-commit** — AI provenance, templates, and commit journal
