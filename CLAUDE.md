# CLAUDE.md — Ailloy

Ailloy is a Go CLI tool that initializes projects with agentic AI workflow patterns for the software development lifecycle. It generates CLAUDE.md files, command templates, and Claude Code plugins to streamline GitHub issue management, pull request workflows, and development task automation.

## Quick Reference

```bash
make build          # Build the CLI binary to bin/ailloy
make test           # Run tests (go test -v ./...)
make lint           # Run golangci-lint
make install        # Install to GOPATH/bin
make clean          # Remove build artifacts
make ci             # Run full CI locally with act
```

## Project Structure

```
cmd/ailloy/             # CLI entry point (main.go)
internal/
  commands/             # CLI command implementations (Cobra)
    root.go             # Root command, welcome banner
    init.go             # `ailloy init` — project initialization, CLAUDE.md creation
    customize.go        # `ailloy customize` — template variable configuration
    template.go         # `ailloy template list|show` — template management
    plugin.go           # `ailloy plugin generate|validate|update`
  providers/            # AI provider abstraction
    provider.go         # Provider interface and registry
    claude.go           # Claude provider implementation
pkg/
  config/               # YAML configuration system (ailloy.yaml)
  plugin/               # Plugin generation, validation, updating, transformation
  safepath/             # Path traversal prevention (CWE-22 mitigation)
  styles/               # Terminal styling (Charmbracelet lipgloss), fox ASCII art
  templates/            # Embedded template filesystem
    claude/commands/    # 8 command templates (create-issue, start-issue, open-pr, etc.)
ailloy/                 # Claude Code plugin directory
  .claude-plugin/       # Plugin manifest (plugin.json)
  commands/             # Plugin command implementations
  agents/               # Complex workflow agents
  hooks/                # Event-based automation hooks
  scripts/              # Installation scripts
templates/              # Source template documentation
docs/                   # Project documentation
examples/               # Example configurations
.github/workflows/      # CI/CD (ci.yml, release.yml, security.yml)
```

## Language and Dependencies

- **Go 1.24.1** (module: `github.com/kriscoleman/ailloy`)
- **Cobra v1.9.1** — CLI framework
- **gopkg.in/yaml.v3** — configuration parsing
- **Charmbracelet** (lipgloss, glamour, bubbles) — terminal UI and styling

## Build and Development

### Prerequisites

Go 1.24+, golangci-lint, act (local CI), gh (GitHub CLI). Use `make check-deps` to verify or `flox activate` for a reproducible dev environment via Flox.

### Building

```bash
make build    # Outputs bin/ailloy with version from git describe
```

Version is injected via ldflags: `-X main.version=$(VERSION)`.

### Testing

```bash
make test     # go test -v ./...
```

CI runs tests on Go 1.23 and 1.24 with `-race` and coverage profiling.

### Linting

```bash
make lint     # golangci-lint run
```

### Local CI

```bash
make ci           # Full CI with act (requires Docker)
make ci-build     # Build job only
make ci-lint      # Lint job only
make ci-security  # Security scanning
```

## CI/CD

- **ci.yml**: DCO sign-off check (PRs), build/test on Go 1.23+1.24, golangci-lint
- **release.yml**: release-please for semantic versioning, multi-platform binary builds (linux/darwin/windows, amd64/arm64)
- **security.yml**: gosec and govulncheck, runs weekly and on push/PR

## Commit Conventions

This project uses [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>
```

**Types**: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`
**Scopes**: `cli`, `config`, `templates`, `plugin`, `docs`

All PR commits require a DCO sign-off (`Signed-off-by: Name <email>`).

## Code Style

- Follow [Effective Go](https://go.dev/doc/effective_go) guidelines
- Format with `gofmt`
- Run `golangci-lint` before committing
- Document exported functions, types, and packages
- Handle errors explicitly — don't ignore them
- Wrap errors with context: `fmt.Errorf("reading template %s: %w", path, err)`
- File permissions: 0600 for config files, 0644 for public/template files, 0750 for directories

## Key Patterns

- **Embedded templates**: `pkg/templates/` uses `go:embed` to bundle command templates into the binary
- **Safe path handling**: `pkg/safepath/` validates all file paths to prevent directory traversal attacks
- **Provider abstraction**: `internal/providers/` defines a provider interface for extensible AI provider support
- **Plugin system**: `pkg/plugin/` handles generation, validation, updating, and template-to-command transformation

## Command Templates

Eight templates in `pkg/templates/claude/commands/`:

| Template | Purpose |
|---|---|
| `create-issue.md` | Generate GitHub issues |
| `start-issue.md` | Fetch issue details and begin work |
| `open-pr.md` | Create pull requests |
| `pr-description.md` | Generate PR descriptions |
| `pr-comments.md` | Respond to PR review comments |
| `pr-review.md` | Conduct code reviews |
| `update-pr.md` | Update existing PRs |
| `preflight.md` | Pre-deployment verification |

Templates support variable substitution with `{{variable}}` syntax (e.g., `{{default_board}}`, `{{organization}}`, `{{project_id}}`).

## Plugin Management

```bash
make plugin-generate    # Generate Claude plugin from templates
make plugin-validate    # Validate plugin structure
make plugin-update      # Update existing plugin
make plugin-rebuild     # Clean and regenerate
make plugin-diff        # Compare manual vs generated plugins
```
