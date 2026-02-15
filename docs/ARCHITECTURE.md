# Ailloy — Architecture & Onboarding Guide

## What Is Ailloy?

Ailloy is a CLI tool that initializes projects with AI-assisted development workflows — specifically optimized for Claude Code. Think of it as a scaffolding tool: you run `ailloy init` in a repo, and it sets up structured templates that Claude Code can use as slash commands (`/open-pr`, `/brainstorm`, `/create-issue`, etc.).

The name is a metallurgy metaphor — combining traditional dev practices with AI to produce something stronger than either alone.

**Current stage:** Alpha. Core features work, but the provider abstraction (for future OpenAI support) and plugin marketplace are still planned.

---

## High-Level Architecture

```
cmd/ailloy/main.go          <- Entry point, injects build metadata via ldflags
    |
    v
internal/commands/           <- Cobra CLI command handlers (init, template, customize, plugin)
    |
    |-- internal/providers/  <- AI provider abstraction (registry pattern, currently Claude-only)
    |
    v
pkg/                         <- Public, reusable packages
    |-- config/              <- YAML config loading/saving, template variable substitution
    |-- templates/           <- Embedded template filesystem (go:embed)
    |-- plugin/              <- Plugin generation, transformation, validation, updates
    |-- styles/              <- Terminal styling (lipgloss/glamour), ASCII fox mascots
    +-- safepath/            <- Directory traversal prevention (CWE-22 mitigation)
```

The key insight: **templates are compiled into the binary** via Go's `//go:embed`. There are no runtime file dependencies. When a user runs `ailloy init`, templates are read from the embedded filesystem, variable-substituted, and written to the project's `.claude/` directory.

---

## Core Data Flow

### The Init Flow (most important to understand)

```
ailloy init
  -> Check for .git (warn if absent)
  -> Create .claude/commands/ and .claude/skills/
  -> Load config: project (.ailloy/ailloy.yaml) merged with global (~/.ailloy/ailloy.yaml)
  -> For each embedded template:
       Read from embed.FS
       Replace {{variable}} placeholders with config values
       Write to .claude/commands/ or .claude/skills/
  -> Display success banner
```

This is defined in `internal/commands/init.go`. The template reading happens through `pkg/templates/templates.go` and variable substitution through `pkg/config/config.go` (`ProcessTemplate()`).

### Configuration Hierarchy

```
Project config (.ailloy/ailloy.yaml)    <- Highest priority
         | falls back to
Global config (~/.ailloy/ailloy.yaml)   <- User-level defaults
         | falls back to
Built-in defaults                        <- Hardcoded in code
```

The config struct in `pkg/config/config.go`:

```go
type Config struct {
    Project   ProjectConfig    // name, description, ai_providers
    Templates TemplateConfig   // variables, repositories, auto_update
    Workflows WorkflowConfig   // workflow definitions
    User      UserConfig       // name, email
    Providers ProvidersConfig  // claude & gpt API key env vars
}
```

---

## CLI Command Structure

```
ailloy
|-- init                     # Scaffold .claude/ templates into a project
|   +-- --global             # Init user-level config instead
|-- template
|   |-- list                 # Show all available templates
|   +-- show <name>          # Display a specific template's content
|-- customize                # Manage template variables
|   +-- --set, --list, --delete, --global
+-- plugin
    |-- generate             # Create a Claude Code plugin from templates
    |-- update [path]        # Update an existing plugin
    +-- validate [path]      # Check plugin structure integrity
```

Commands are wired via Cobra's `init()` pattern — each file in `internal/commands/` registers its commands with `rootCmd.AddCommand()`.

---

## Key Design Patterns

### 1. Embedded Filesystem for Templates

In `pkg/templates/templates.go`:

```go
//go:embed all:claude
var embeddedTemplates embed.FS
```

This compiles the `pkg/templates/claude/` directory into the binary. Templates are accessed via `GetTemplate(name)` and `GetSkill(name)` — no disk I/O needed.

### 2. Provider Registry (Extensibility Point)

In `internal/providers/provider.go`:

```go
type Provider interface {
    Name() string
    ExecuteTemplate(ctx, template, context) (*Response, error)
    ValidateConfig() error
    IsEnabled() bool
}
```

A `Registry` manages providers by name. Currently only `internal/providers/claude.go` exists (and it's mostly a placeholder for future API integration). This is where you'd add OpenAI support.

### 3. Safe Path Handling

`pkg/safepath/safepath.go` prevents directory traversal attacks (CWE-22). All user-provided paths go through `safepath.Clean()`, `ValidateUnder()`, or `Join()` before filesystem operations.

### 4. Plugin Transformation Pipeline

The plugin system converts Ailloy markdown templates into Claude Code plugin format:

```
Embedded Template -> Transformer -> Claude Command Format -> Plugin Directory
```

`pkg/plugin/transformer.go` parses markdown sections, extracts descriptions, and reformats. `pkg/plugin/generator.go` orchestrates creating the full plugin directory structure (manifest, commands, scripts, README).

---

## The 9 Command Templates

These live in `pkg/templates/claude/commands/`:

| Template | Purpose |
|----------|---------|
| `brainstorm` | Structured ideation using freewriting, cubing, and journalistic methods |
| `create-issue` | Generate well-formatted GitHub issues |
| `start-issue` | Fetch issue details and begin implementation |
| `open-pr` | Create PRs with structured descriptions |
| `pr-description` | Generate comprehensive PR descriptions |
| `update-pr` | Update existing PRs |
| `pr-review` | Conduct code reviews (silent/interactive modes) |
| `pr-comments` | Respond to PR review comments |
| `preflight` | Pre-release / pre-PR checks |

Plus 1 skill: `brainstorm` (in `pkg/templates/claude/skills/`) — a proactive skill Claude Code can invoke automatically.

---

## Build System

### Makefile Targets You'll Use Daily

```bash
make build          # Build binary to bin/ailloy (with version/commit/date injected)
make test           # go test -v ./...
make lint           # golangci-lint run
make fmt            # Check gofmt compliance
make hooks          # Install lefthook git hooks
```

### Version Injection

The binary gets version info at compile time via ldflags:

```makefile
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"
```

`cmd/ailloy/main.go` passes these to `commands.SetVersionInfo()`, which builds the display string shown in the CLI banner.

---

## Git Hooks (lefthook)

Defined in `lefthook.yml`:

| Hook | What Runs | Blocking? |
|------|-----------|-----------|
| `pre-commit` | `go vet` + `gofmt` (staged files only) | Yes |
| `commit-msg` | `conform enforce` (conventional commits) | Yes |
| `pre-push` | `golangci-lint` + `go build` + `go test -race` | Yes |

Install with `make hooks`. Commit messages must follow [Conventional Commits](https://www.conventionalcommits.org/) — enforced by `.conform.yaml` (header max 89 chars, imperative mood, lowercase, specific type prefixes like `feat:`, `fix:`, `chore:`, etc.).

---

## Testing Patterns

Tests are colocated with their packages. The project uses two styles:

**Table-driven unit tests** (most common):

```go
func TestGetTemplateIcon(t *testing.T) {
    tests := []struct {
        name     string
        expected string
    }{
        {"brainstorm", "💡"},
        {"create-issue", "🎯"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // ...
        })
    }
}
```

**Integration tests** using `t.TempDir()` for filesystem isolation:

```go
func TestIntegration_CopyTemplateFiles(t *testing.T) {
    tmpDir := t.TempDir()
    // Create dirs, run workflow, verify files exist with correct content
}
```

Run all tests: `make test` or `go test -v -race ./...`

---

## CI/CD

Three GitHub Actions workflows in `.github/workflows/`:

1. **ci.yml** — Every push/PR: build + test (Go 1.23 & 1.24 matrix), conventional commit check, lint, coverage upload
2. **release.yml** — Push to main: release-please for automatic versioning + cross-platform binary builds (linux/darwin/windows x amd64/arm64)
3. **security.yml** — Security scanning

---

## Dependencies

Intentionally minimal:

- **`github.com/spf13/cobra`** — CLI framework (industry standard for Go CLIs)
- **`gopkg.in/yaml.v3`** — YAML config parsing
- **`github.com/charmbracelet/lipgloss`** + **`glamour`** — Terminal styling and markdown rendering (transitive, ~40 packages)

No database, no HTTP server, no heavy frameworks.

---

## Quick Mental Model

> Ailloy is a **template compiler for AI workflows**. It takes markdown templates with `{{variables}}`, merges them with user config, and deploys them where Claude Code expects to find them (`.claude/commands/` and `.claude/skills/`). The plugin system is a secondary output format that packages these same templates as a distributable Claude Code plugin.

The codebase is clean, well-separated, and follows standard Go conventions. Start by reading `internal/commands/init.go` and `pkg/config/config.go` — those two files will give you the best intuition for how everything connects.
