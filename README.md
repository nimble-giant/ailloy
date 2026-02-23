# Ailloy

[![CI](https://github.com/nimble-giant/ailloy/actions/workflows/ci.yml/badge.svg)](https://github.com/nimble-giant/ailloy/actions/workflows/ci.yml)
[![Security](https://github.com/nimble-giant/ailloy/actions/workflows/security.yml/badge.svg)](https://github.com/nimble-giant/ailloy/actions/workflows/security.yml)
[![Release](https://github.com/nimble-giant/ailloy/actions/workflows/release.yml/badge.svg)](https://github.com/nimble-giant/ailloy/actions/workflows/release.yml)

![Ailloy Mascot](.assets/Ailloy%20the%20Blacksmith%20Innovator.png)

**Ailloy is the package manager for AI instructions.** It helps you find, create, and share reusable AI workflow packages — the same way [Helm](https://helm.sh/) manages Kubernetes applications. Molds are to Ailloy what charts are to Helm: versioned, configurable packages that can be installed into any project or workload.

Ailloy gives teams a reproducible pipeline for authoring, packaging, and distributing AI-assisted development workflows.

Like in metallurgy — where combining two elements yields a stronger alloy — Ailloy represents the fusion of human development practices with AI assistance.

![Ailloy with Claude Integration](.assets/Friendly%20Ailloy%20with%20Glowing%20Orb.png)

## What is Ailloy?

Ailloy is the best way to find, create, and share AI instruction packages. Like Helm for Kubernetes, it provides:

- **Manage Complexity**: Molds describe complete AI workflows — commands, skills, GitHub Actions — with a single `mold.yaml` manifest and configurable flux variables
- **Easy Updates**: Override defaults at install time with `--set` flags or layered value files, following Helm-style precedence
- **Simple Sharing**: Package molds into distributable tarballs or self-contained binaries with `ailloy smelt`, then share them across teams and projects

### The Ailloy Pipeline

| Step | Command | Helm Equivalent | What It Does |
|------|---------|-----------------|--------------|
| Author | — | — | Write instruction templates (blanks) with Go `text/template` syntax |
| Configure | `ailloy anneal` | — | Interactive wizard to set flux variables |
| Preview | `ailloy forge` | `helm template` | Dry-run render of blanks with flux values |
| Install | `ailloy cast` | `helm install` | Compile and install blanks into a project |
| Package | `ailloy smelt` | `helm package` | Bundle a mold into a tarball or binary |
| Validate | `ailloy temper` | `helm lint` | Lint mold structure, manifests, and templates |

Currently focused on **Claude Code** because it offers the level of customization and instruction support needed for sophisticated AI-assisted development workflows.

## Quick Start

Get started in minutes:

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

Molds can be resolved directly from git repositories — no local clone required:

```bash
# Install blanks from the official mold (resolves latest tag from GitHub)
ailloy cast github.com/nimble-giant/nimble-mold

# Pin to a specific version
ailloy cast github.com/nimble-giant/nimble-mold@v0.1.10

# Semver constraints (caret, tilde, range)
ailloy cast github.com/nimble-giant/nimble-mold@^0.1.0

# Include GitHub Actions workflow blanks
ailloy cast github.com/nimble-giant/nimble-mold --with-workflows

# Override flux values at install time
ailloy cast github.com/nimble-giant/nimble-mold --set project.organization=mycompany

# Local mold directories still work
ailloy cast ./my-local-mold
```

### Configure Flux Variables

```bash
# Interactive wizard — reads flux.schema.yaml from a mold
ailloy anneal github.com/nimble-giant/nimble-mold -o flux-overrides.yaml

# Scripted mode
ailloy anneal --set project.organization=mycompany --set project.board=Engineering -o flux-overrides.yaml

# Use overrides with cast
ailloy cast github.com/nimble-giant/nimble-mold -f flux-overrides.yaml
```

### Working with Blanks

```bash
# List available blanks
ailloy mold list

# View a specific blank
ailloy mold show create-issue
```

## Available Commands

### `ailloy cast [mold-ref]`

Install rendered blanks from a mold into the current project (alias: `install`). Accepts a local directory path or a remote git reference (`host/owner/repo[@version][//subpath]`):

- `-g, --global`: Install into user home directory (`~/`) instead of current project
- `--with-workflows`: Include GitHub Actions workflow blanks
- `--set key=value`: Override flux variables (can be repeated)
- `-f, --values file`: Layer additional flux value files (can be repeated)

### `ailloy forge [mold-ref]`

Dry-run render of mold blanks (aliases: `blank`, `template`). Accepts a local directory path or a remote git reference:

- `-o, --output dir`: Write rendered files to a directory instead of stdout
- `--set key=value`: Set flux values (can be repeated)
- `-f, --values file`: Layer additional flux value files (can be repeated)

### `ailloy mold`

Manage AI command blanks:

- `list`: Show all available blanks
- `show <blank-name>`: Display blank content
- `get <reference>`: Download a mold to local cache without installing (validates `mold.yaml`, prints cache path)

### `ailloy anneal [mold-ref]`

Dynamic, mold-aware wizard to configure flux variables (alias: `configure`). Reads `flux.schema.yaml` from the mold to generate type-driven prompts with optional discovery commands. Accepts a local directory path or a remote git reference:

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

### `ailloy foundry`

Discover and manage mold registries. See the [Remote Molds guide](docs/foundry.md) for details:

- `search <query>`: Search GitHub for molds tagged with the `ailloy-mold` topic
- `add <url>`: Register a foundry index URL in `~/.ailloy/config.yaml`

### `ailloy ingot`

Manage reusable template components (ingots):

- `get <reference>`: Download an ingot to the local cache without installing
- `add <reference>`: Download and install an ingot into the project's `.ailloy/ingots/` directory

### `ailloy plugin`

Generate and manage Claude Code plugins:

- `generate`: Generate Claude Code plugin from blanks (`--mold`, `--output`, `--watch`, `--force`)
- `update [path]`: Update existing Claude Code plugin (`--mold`, `--force`)
- `validate [path]`: Validate Claude Code plugin structure

### Bidirectional Commands

All compound commands support both noun-verb and verb-noun ordering:

```bash
# These are equivalent
ailloy foundry search blueprint    # noun-verb
ailloy search foundry blueprint    # verb-noun

ailloy mold get github.com/org/repo
ailloy get mold github.com/org/repo

ailloy ingot add github.com/org/repo
ailloy add ingot github.com/org/repo
```

## Blanks

Blanks are Markdown instruction templates that define Claude Code slash commands, skills, and GitHub Actions workflows. Each blank lives in a mold directory and is compiled with flux variables when you run `ailloy cast` or `ailloy forge`.

There are three types of blanks:

- **Commands** (`commands/`) — Slash commands users invoke explicitly (e.g., `/brainstorm`, `/create-issue`)
- **Skills** (`skills/`) — Proactive workflows Claude Code uses automatically based on context
- **Workflows** (`workflows/`) — GitHub Actions YAML files, installed with `--with-workflows`

### Creating a Blank

Add a Markdown file to your mold's `commands/` or `skills/` directory. Reference flux variables with Go [text/template](https://pkg.go.dev/text/template) syntax:

```markdown
# Deploy Checklist

Generate a deployment checklist for {{ project.organization }}.

1. Use `{{ scm.cli }}` to check for open PRs targeting the release branch
2. Verify all CI checks are passing

{{if .ore.status.enabled}}
Update the status field ({{ .ore.status.field_id }}) after each step.
{{end}}
```

### Working with Blanks

```bash
# List installed blanks
ailloy mold list

# View a specific blank
ailloy mold show create-issue

# Preview rendered output (dry run)
ailloy forge ./my-mold

# Install into current project
ailloy cast ./my-mold
```

The [official mold](https://github.com/nimble-giant/nimble-mold) provides pre-built blanks for common SDLC tasks (issue management, PR workflows, code review) and is a good reference for blank structure and conventions.

For the full guide on creating blanks, template syntax, and ingot includes, see the [Blanks guide](docs/blanks.md). For packaging and distributing molds, see the [Packaging Molds guide](docs/smelt.md).

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
# Interactive wizard — reads schema from mold, generates a flux YAML file
ailloy anneal github.com/nimble-giant/nimble-mold -o my-overrides.yaml

# Scripted mode
ailloy anneal --set project.organization=mycompany -o my-overrides.yaml

# Use overrides when casting
ailloy cast github.com/nimble-giant/nimble-mold -f my-overrides.yaml

# Or override inline
ailloy cast github.com/nimble-giant/nimble-mold --set project.organization=mycompany
```

For the full guide on flux variables, schemas, and value layering, see the [Flux Variables guide](docs/flux.md). For the interactive wizard, see the [Anneal guide](docs/anneal.md).

## Project Structure

```text
/cmd/ailloy          # CLI tool entry point
/internal            # Private Go packages
  /commands          # CLI command implementations (cast, forge, smelt, etc.)
/pkg
  /blanks            # MoldReader abstraction (reads mold directories)
  /foundry           # SCM-native mold resolution, caching, and version management
  /github            # GitHub ProjectV2 discovery via gh API GraphQL
  /mold              # Template engine, flux loading, ingot resolution
  /plugin            # Plugin generation pipeline
  /safepath          # Safe path utilities
  /smelt             # Mold packaging (tarball/binary)
  /styles            # Terminal UI styles (lipgloss)
/docs                # Documentation
```

## Current Status

**Alpha Stage**: Ailloy is an early-stage package manager for AI instructions. The toolchain provides:

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
- ✅ SCM-native mold resolution from git repos with semver constraints and local caching
- ✅ Foundry search and discovery via GitHub topic-based registry
- ✅ Ingot package management (get, add) with bidirectional CLI commands
- 🔄 Additional AI provider support (planned)
- 🔄 Advanced workflow automation (planned)

## Contributing

This is an evolving project — community input welcome! As AI development practices mature, Ailloy aims to be the standard package manager for AI instructions, the way Helm became the standard for Kubernetes.

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
