# Ailloy

[![CI](https://github.com/nimble-giant/ailloy/actions/workflows/ci.yml/badge.svg)](https://github.com/nimble-giant/ailloy/actions/workflows/ci.yml)
[![Security](https://github.com/nimble-giant/ailloy/actions/workflows/security.yml/badge.svg)](https://github.com/nimble-giant/ailloy/actions/workflows/security.yml)
[![Release](https://github.com/nimble-giant/ailloy/actions/workflows/release.yml/badge.svg)](https://github.com/nimble-giant/ailloy/actions/workflows/release.yml)

![Ailloy Mascot](.assets/Ailloy%20the%20Blacksmith%20Innovator.png)

**Ailloy** is a CLI tool for initializing projects with agentic AI patterns to assist with software development lifecycle (SDLC) tasks. Currently focused on Claude Code integration, Ailloy helps developers set up structured AI workflows using blanks and configuration files.

Like in metallurgy—where combining two elements yields a stronger alloy—Ailloy represents the fusion of traditional development practices with AI assistance to create more efficient engineering workflows.

![Ailloy with Claude Integration](.assets/Friendly%20Ailloy%20with%20Glowing%20Orb.png)

## What is Ailloy?

Ailloy helps you:

- **Initialize AI-ready projects**: Set up command blank structure for AI-assisted workflows
- **Customize blanks**: Configure team-specific defaults for consistent workflows
- **Manage AI blanks**: Access pre-built blanks for common development tasks
- **Standardize AI workflows**: Use consistent patterns for GitHub issues, PRs, and development tasks

Currently focused on **Claude Code** because it offers the level of customization and blank support needed for sophisticated AI-assisted development workflows.

## Quick Start

Get started in minutes with the CLI:

![Ailloy Ready to Help](.assets/Ailloy%20Winking%20with%20Orb%20and%20Hammer.png)

### Installation

#### Quick Install (Recommended)

Download the latest pre-built binary for your platform:

```bash
curl -fsSL https://raw.githubusercontent.com/nimble-giant/ailloy/main/install.sh | bash
```

#### Go Install

If you have Go installed:

```bash
go install github.com/nimble-giant/ailloy/cmd/ailloy@latest
```

#### Build from Source

```bash
git clone https://github.com/nimble-giant/ailloy
cd ailloy
make build
```

The binary will be available at `./bin/ailloy`.

### Cast a Mold into a Project

```bash
# Install blanks from the official mold
ailloy cast ./nimble-mold

# Include GitHub Actions workflow blanks
ailloy cast ./nimble-mold --with-workflows

# Override flux values at install time
ailloy cast ./nimble-mold --set project.organization=mycompany
```

### Configure Flux Variables

```bash
# Interactive wizard to configure flux variables
ailloy anneal -o flux-overrides.yaml

# Scripted mode
ailloy anneal --set project.organization=mycompany --set project.board=Engineering -o flux-overrides.yaml

# Use overrides with cast
ailloy cast ./nimble-mold -f flux-overrides.yaml
```

### Working with Blanks

```bash
# List available blanks
ailloy mold list

# View a specific blank
ailloy mold show create-issue
```

## Available Commands

### `ailloy cast [mold-dir]`

Install rendered blanks from a mold into the current project (alias: `install`):

- `--with-workflows`: Include GitHub Actions workflow blanks
- `--set key=value`: Override flux variables (can be repeated)
- `-f, --values file`: Layer additional flux value files (can be repeated)

### `ailloy forge [mold-dir]`

Dry-run render of mold blanks (aliases: `blank`, `template`):

- `-o, --output dir`: Write rendered files to a directory instead of stdout
- `--set key=value`: Set flux values (can be repeated)
- `-f, --values file`: Layer additional flux value files (can be repeated)

### `ailloy mold`

Manage AI command blanks:

- `list`: Show all available blanks
- `show <blank-name>`: Display blank content

### `ailloy anneal`

Interactive wizard to configure flux variables (alias: `configure`):

- `-s, --set key=value`: Set flux variable in scripted mode (can be repeated)
- `-o, --output file`: Write flux YAML to file (default: stdout)

### `ailloy smelt [mold-dir]`

Package a mold into a distributable format (alias: `package`):

- `-o, --output-format`: Output format: `tar` (default) or `binary`
- `--output dir`: Output directory (default: current directory)

### `ailloy temper [path]`

Validate and lint a mold or ingot package (alias: `lint`):

- Checks structural integrity, manifest fields, file references, template syntax, and flux schema consistency
- Reports errors (blocking) and warnings (informational)

### `ailloy plugin`

Generate and manage Claude Code plugins:

- `generate`: Generate Claude Code plugin from blanks (`--mold`, `--output`, `--watch`, `--force`)
- `update [path]`: Update existing Claude Code plugin (`--mold`, `--force`)
- `validate [path]`: Validate Claude Code plugin structure

## Blanks

Ailloy includes pre-built blanks for common SDLC tasks, optimized for Claude Code:

### Available Blanks

- **`brainstorm`**: Analyze an idea for feasibility, scope, and value using structured brainstorming techniques
- **`create-issue`**: Generate well-formatted GitHub issues with proper structure
- **`start-issue`**: Fetch GitHub issue details and begin implementation
- **`open-pr`**: Create pull requests with structured descriptions
- **`pr-description`**: Generate comprehensive PR descriptions
- **`pr-comments`**: Respond to PR review comments efficiently
- **`pr-review`**: Conduct comprehensive code reviews with silent/interactive modes
- **`preflight`**: Pre-flight checks and setup
- **`update-pr`**: Update existing pull requests

### Available Skills

Skills are proactive workflows that Claude Code can use automatically based on context, without requiring explicit slash command invocation:

- **`brainstorm`**: Structured brainstorming methodology for evaluating ideas using freewriting, cubing, and journalistic techniques
- **`add-ailloy-blank`**: Guided workflow for creating new Ailloy blanks with proper mold structure

### Workflow Blanks

Ailloy also includes GitHub Actions workflow blanks in the official mold (`nimble-mold/workflows/`). These are installed into your project's `.github/workflows/` when using `ailloy cast --with-workflows`:

- **`claude-code`**: GitHub Actions workflow for the [Claude Code agent](https://github.com/anthropics/claude-code-action). Responds to `@claude` mentions in issues, PR comments, and PR reviews. Requires an `ANTHROPIC_API_KEY` secret in your repository.
- **`claude-code-review`**: GitHub Actions workflow for automated PR reviews with the [Claude Code agent](https://github.com/anthropics/claude-code-action). Features brevity-focused formatting, collapsible sections for detailed analysis, and intelligent comment management (updates summary comments, creates reply comments). Requires an `ANTHROPIC_API_KEY` secret in your repository.

### Using Blanks

Blanks are Markdown files containing instructions for Claude Code. You can:

1. View blanks: `ailloy mold show create-issue`
2. Copy blank content into Claude Code conversations
3. Customize blanks for your project's specific needs

### Blank Structure

Each blank includes:

- Clear instructions for Claude Code
- Context requirements
- Expected output format
- Integration with GitHub CLI commands

### Blank Customization

Blanks use Go's [text/template](https://pkg.go.dev/text/template) engine with dotted flux variable paths. Common variables include:

- `{{ project.board }}`: Default GitHub project board name
- `{{ project.organization }}`: GitHub organization name
- `{{ scm.provider }}`: Source control provider (e.g., GitHub)
- `{{ scm.cli }}`: CLI tool for SCM operations (e.g., gh)

Blanks also support **conditional rendering** based on your flux configuration:

```markdown
{{if .ore.status.enabled}}
Status Field: {{ .ore.status.field_id }}
{{end}}
```

Variables and conditionals are processed when blanks are rendered during `ailloy cast` or `ailloy forge`. See the [Packaging Molds Guide](docs/smelt.md) for full details on flux values, mold structure, and template syntax.

## Configuration

Ailloy uses **flux** — YAML variable files that configure how blanks are rendered. Each mold ships with a `flux.yaml` containing defaults, and you can override values at multiple layers.

### Flux Value Precedence

When blanks are rendered with `forge` or `cast`, flux values are resolved in this order (lowest to highest priority):

1. `mold.yaml` `flux:` schema defaults
2. `flux.yaml` defaults (shipped with the mold)
3. `-f` value files (left to right, later files override earlier)
4. `--set` flags (highest priority)

### Configuring Flux Values

```bash
# Interactive wizard — generates a flux YAML file
ailloy anneal -o my-overrides.yaml

# Scripted mode
ailloy anneal --set project.organization=mycompany -o my-overrides.yaml

# Use overrides when casting
ailloy cast ./nimble-mold -f my-overrides.yaml

# Or override inline
ailloy cast ./nimble-mold --set project.organization=mycompany
```

## Project Structure

```text
/cmd/ailloy          # CLI tool entry point
/internal            # Private Go packages
  /commands          # CLI command implementations (cast, forge, smelt, etc.)
/pkg
  /blanks            # MoldReader abstraction (reads mold directories)
  /github            # GitHub ProjectV2 discovery via gh API GraphQL
  /mold              # Template engine, flux loading, ingot resolution
  /plugin            # Plugin generation pipeline
  /safepath          # Safe path utilities
  /smelt             # Mold packaging (tarball/binary)
  /styles            # Terminal UI styles (lipgloss)
/nimble-mold         # Official mold (commands, skills, workflows, flux)
/docs                # Documentation
```

## Current Status

**Alpha Stage**: Ailloy is an early-stage tool focused on Claude Code integration. The CLI provides:

- ✅ Mold casting and forging with flux variable rendering
- ✅ Blank management and viewing
- ✅ Flux-based configuration with Helm-style value precedence
- ✅ Conditional blank rendering with ore model-aware context
- ✅ Mold packaging (tarball and self-contained binary)
- ✅ Mold/ingot validation and linting
- ✅ Claude Code plugin generation from blanks
- ✅ Claude Code-optimized workflow blanks
- ✅ Automatic GitHub Project field discovery via GraphQL
- ✅ Interactive wizard with charmbracelet/huh for guided configuration
- 🔄 Additional AI provider support (planned)
- 🔄 Advanced workflow automation (planned)

## Contributing

This is an evolving framework—community input welcome! As AI development practices mature, Ailloy aims to capture and standardize the most effective patterns for human-AI collaboration.

### Prerequisites

**Option 1: Flox (Recommended)**

[Flox](https://flox.dev/) provides a reproducible development environment with all dependencies:

```bash
# Install Flox (one-time)
curl -fsSL https://flox.dev/install | bash

# Activate the development environment
flox activate

# Verify dependencies
make check-deps
```

**Option 2: Manual Installation**

If you prefer to manage dependencies manually:

| Dependency | Purpose | Installation |
|------------|---------|--------------|
| [Go 1.24+](https://go.dev/dl/) | Build the CLI | `brew install go` |
| [golangci-lint](https://golangci-lint.run/) | Linting | `brew install golangci-lint` |
| [lefthook](https://github.com/evilmartians/lefthook) | Git hooks | `brew install lefthook` |
| [Docker](https://www.docker.com/) | Required for local CI | [Docker Desktop](https://www.docker.com/products/docker-desktop/) |
| [act](https://github.com/nektos/act) | Run GitHub Actions locally | `brew install act` |
| [gh](https://cli.github.com/) | GitHub CLI | `brew install gh` |

### Development Setup

```bash
# Clone the repository
git clone https://github.com/nimble-giant/ailloy
cd ailloy

# Activate Flox environment (or install deps manually)
flox activate

# Install git hooks (runs checks automatically on commit/push)
make hooks

# Build the project
make build

# Run tests
make test

# Run linter
make lint

# Run CI locally (requires Docker)
make ci
```

### To Contribute

1. Fork the repository
2. Create a feature branch
3. Add blanks or improve CLI functionality
4. Ensure tests pass: `make test`
5. Ensure linting passes: `make lint`
6. Submit a pull request

## Contributors

<a href="https://github.com/nimble-giant/ailloy/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=nimble-giant/ailloy" />
</a>

## License

Apache License 2.0 - see [LICENSE.md](LICENSE.md) for details.
