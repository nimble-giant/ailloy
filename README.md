<div align="center">

<img src=".assets/Ailloy%20the%20Blacksmith%20Innovator.png" alt="Ailloy" width="280" />

# Ailloy

**The package manager for AI instructions.**

[![CI](https://github.com/nimble-giant/ailloy/actions/workflows/ci.yml/badge.svg)](https://github.com/nimble-giant/ailloy/actions/workflows/ci.yml)
[![Security](https://github.com/nimble-giant/ailloy/actions/workflows/security.yml/badge.svg)](https://github.com/nimble-giant/ailloy/actions/workflows/security.yml)
[![Release](https://github.com/nimble-giant/ailloy/actions/workflows/release.yml/badge.svg)](https://github.com/nimble-giant/ailloy/actions/workflows/release.yml)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE.md)

Find, create, and share reusable AI workflow packages — the way [Helm](https://helm.sh/) manages Kubernetes applications.

[**Quick Start**](#quick-start) · [**Documentation**](docs/) · [**Official Mold**](https://github.com/nimble-giant/nimble-mold) · [**Contributing**](#contributing)

</div>

---

## Why Ailloy

- **Reproducible** — Versioned, configurable mold packages with Helm-style value precedence. Same mold, same flux values, same output every time.
- **Tool-agnostic** — Works with any AI coding tool that reads file-based instructions: Claude Code, Cursor, Windsurf, GitHub Copilot, and [more](https://agents.md). The `output:` mapping in `flux.yaml` decides where blanks land.
- **Helm-style ergonomics** — `cast`, `forge`, `smelt`, `temper`. If you know `helm install` and `helm template`, you already know the shape.

## What is Ailloy?

Ailloy compiles, packages, and distributes AI-assisted development workflows. **Molds** are to Ailloy what charts are to Helm: versioned, configurable packages that can be installed into any project.

Like in metallurgy — combining elements yields a stronger alloy — Ailloy fuses human development practices with AI assistance.

| Helm | Ailloy | What it does |
|------|--------|--------------|
| `helm template` | `ailloy forge` | Dry-run render of blanks with flux values |
| `helm install` | `ailloy cast` | Compile and install blanks into a project |
| `helm package` | `ailloy smelt` | Bundle a mold into a tarball or binary |
| `helm lint` | `ailloy temper` | Validate mold structure and templates |
| — | `ailloy anneal` | Interactive wizard to set flux variables |
| — | `ailloy assay` | Lint rendered AI instruction files |

## How It Works

<img src=".assets/Friendly%20Ailloy%20with%20Glowing%20Orb.png" alt="Ailloy mascot" width="180" align="right" />

The Ailloy pipeline is a small set of composable steps. Author once, configure per project, render and install anywhere.

| Step | Command | Description |
|------|---------|-------------|
| Author | — | Write instruction templates (blanks) with Go `text/template` syntax |
| Configure | `ailloy anneal` | Interactive wizard to set flux variables |
| Preview | `ailloy forge` | Dry-run render of blanks with flux values |
| Install | `ailloy cast` | Compile and install blanks into a project |
| Package | `ailloy smelt` | Bundle a mold into a tarball or binary |
| Validate | `ailloy temper` | Validate mold structure, manifests, and templates |
| Lint | `ailloy assay` | Lint rendered AI instruction files against best practices |

<br clear="right" />

## Quick Start

<img src=".assets/Ailloy%20Winking%20with%20Orb%20and%20Hammer.png" alt="Ailloy ready to help" width="200" align="right" />

### 1. Install

<details>
<summary><strong>Quick install (recommended)</strong></summary>

```bash
curl -fsSL https://raw.githubusercontent.com/nimble-giant/ailloy/main/install.sh | bash
```

</details>

<details>
<summary><strong>Go install</strong></summary>

```bash
go install github.com/nimble-giant/ailloy/cmd/ailloy@latest
```

</details>

<details>
<summary><strong>Build from source</strong></summary>

```bash
git clone https://github.com/nimble-giant/ailloy
cd ailloy
make build   # binary at ./bin/ailloy
```

</details>

### 2. Cast a mold

Molds resolve directly from git — no clone required:

```bash
# Install blanks from the official mold (latest tag)
ailloy cast github.com/nimble-giant/nimble-mold

# Pin to a version, or use a semver range
ailloy cast github.com/nimble-giant/nimble-mold@v0.1.10
ailloy cast github.com/nimble-giant/nimble-mold@^0.1.0

# Include GitHub Actions workflow blanks
ailloy cast github.com/nimble-giant/nimble-mold --with-workflows

# Override flux at install time
ailloy cast github.com/nimble-giant/nimble-mold --set project.organization=mycompany
```

### 3. Configure flux variables

```bash
# Interactive wizard
ailloy anneal github.com/nimble-giant/nimble-mold -o flux-overrides.yaml

# Use the overrides
ailloy cast github.com/nimble-giant/nimble-mold -f flux-overrides.yaml
```

<br clear="right" />

## Commands

<details>
<summary><strong><code>cast</code> · <code>forge</code></strong> — install and preview molds</summary>

**`ailloy cast [mold-ref]`** (alias: `install`) — Render and install blanks. Accepts a local path or `host/owner/repo[@version][//subpath]`.

- `-g, --global` — Install into `~/` instead of the current project
- `--with-workflows` — Include GitHub Actions workflow blanks
- `--set key=value` — Override flux variables (repeatable)
- `-f, --values file` — Layer flux value files (repeatable)

**`ailloy forge [mold-ref]`** (aliases: `blank`, `template`) — Dry-run render of mold blanks.

- `-o, --output dir` — Write to a directory instead of stdout
- `--set`, `-f` — Same as `cast`

</details>

<details>
<summary><strong><code>mold</code> · <code>ingot</code></strong> — manage blanks and template components</summary>

**`ailloy mold`** — Manage AI command blanks.

- `list` — Show all available blanks
- `show <blank-name>` — Display blank content
- `get <reference>` — Download a mold to local cache without installing

**`ailloy ingot`** — Reusable template components.

- `get <reference>` — Download to local cache
- `add <reference>` — Install into the project's `.ailloy/ingots/`

</details>

<details>
<summary><strong><code>anneal</code></strong> — configure flux variables</summary>

**`ailloy anneal [mold-ref]`** (alias: `configure`) — Mold-aware wizard. Reads `flux.schema.yaml` to generate type-driven prompts with optional discovery commands.

- `-s, --set key=value` — Set in scripted mode (repeatable)
- `-o, --output file` — Write flux YAML to file (default: stdout)

</details>

<details>
<summary><strong><code>smelt</code></strong> — package molds for distribution</summary>

**`ailloy smelt [mold-dir]`** (alias: `package`) — Package a mold into a distributable format.

- `-o, --output-format` — `tar` (default) or `binary`
- `--output dir` — Output directory

</details>

<details>
<summary><strong><code>assay</code> · <code>temper</code></strong> — lint and validate</summary>

**`ailloy assay [path]`** (alias: `lint`) — Lint rendered AI instruction files.

- Auto-detects CLAUDE.md, AGENTS.md, Cursor rules, Copilot instructions, and more
- `--format json|markdown` for CI · `--fail-on warning|suggestion` for exit control
- Configure via `.ailloyrc.yaml` (`--init` for a starter)

**`ailloy temper [path]`** (alias: `validate`) — Validate a mold or ingot package.

- Checks structural integrity, manifests, file references, template syntax, flux schema
- `--lint` — Render and run assay on output before casting
- `--set`, `-f`, `--format`, `--fail-on`, `--max-lines`

</details>

<details>
<summary><strong><code>foundry</code></strong> — discover and manage mold registries</summary>

See the [Remote Molds guide](docs/foundry.md).

- `search <query>` — Search registered indexes and GitHub Topics
- `add <url>` — Register a foundry index (git repo or static YAML URL)
- `list` — List registered indexes and their status
- `remove <name|url>` — Remove a registered index
- `update` — Refresh all cached indexes

</details>

<details>
<summary><strong><code>plugin</code></strong> — Claude Code plugin generation</summary>

Currently Claude Code specific; the core pipeline is tool-agnostic.

- `generate` — Generate plugin from blanks (`--mold`, `--output`, `--watch`, `--force`)
- `update [path]` — Update existing plugin
- `validate [path]` — Validate plugin structure

</details>

<details>
<summary><strong>Bidirectional commands</strong> — noun-verb or verb-noun</summary>

```bash
ailloy foundry search blueprint    # noun-verb
ailloy search foundry blueprint    # verb-noun

ailloy mold get github.com/org/repo
ailloy get mold github.com/org/repo

ailloy ingot add github.com/org/repo
ailloy add ingot github.com/org/repo
```

</details>

## Blanks

Blanks are Markdown instruction templates — commands, skills, or workflows — that compile with flux variables when you `cast` or `forge` a mold.

- **Commands** (`commands/`) — Invoked explicitly (e.g., `/brainstorm`, `/create-issue`)
- **Skills** (`skills/`) — Proactive workflows the AI tool uses based on context
- **Workflows** (`workflows/`) — GitHub Actions YAML, installed with `--with-workflows`

```markdown
# Deploy Checklist

Generate a deployment checklist for {{ project.organization }}.

1. Use `{{ scm.cli }}` to check for open PRs targeting the release branch
2. Verify all CI checks are passing

{{if .ore.status.enabled}}
Update the status field ({{ .ore.status.field_id }}) after each step.
{{end}}
```

The [official mold](https://github.com/nimble-giant/nimble-mold) ships pre-built blanks for SDLC tasks (issue management, PR workflows, code review) and is a good reference. For the full guide, see [docs/blanks.md](docs/blanks.md). For packaging, see [docs/smelt.md](docs/smelt.md).

## Configuration

Ailloy uses **flux** — YAML variable files that configure how blanks render.

**Precedence** (lowest → highest):

1. `mold.yaml` `flux:` schema defaults
2. `flux.yaml` defaults shipped with the mold
3. `-f` value files (left to right)
4. `--set` flags

For the full guide, see [docs/flux.md](docs/flux.md). For the wizard, see [docs/anneal.md](docs/anneal.md).

## Status

> **Alpha** — Ailloy is an early-stage package manager for AI instructions. The core toolchain is functional and used in production by the maintainers, but APIs and on-disk formats may change before 1.0.

<details>
<summary><strong>What's shipped</strong></summary>

- Mold casting and forging with flux variable rendering
- Blank management and viewing
- Flux-based configuration with Helm-style value precedence
- Conditional blank rendering with ore model-aware context
- Mold packaging (tarball and self-contained binary)
- Mold/ingot validation and linting
- Claude Code plugin generation from blanks
- Workflow blanks for GitHub Actions
- Automatic GitHub Project field discovery via GraphQL
- Interactive wizard with `charmbracelet/huh` for guided configuration
- SCM-native mold resolution from git repos with semver constraints and local caching
- Foundry search and discovery via GitHub Topics and SCM-agnostic foundry indexes
- Ingot package management with bidirectional CLI commands

</details>

<details>
<summary><strong>What's planned</strong></summary>

- Additional AI provider support
- Advanced workflow automation

</details>

<details>
<summary><strong>Project structure</strong></summary>

```text
/cmd/ailloy          # CLI tool entry point
/internal            # Private Go packages
  /commands          # CLI command implementations (cast, forge, smelt, etc.)
/pkg
  /blanks            # MoldReader abstraction (reads mold directories)
  /foundry           # SCM-native mold resolution, caching, version management
  /github            # GitHub ProjectV2 discovery via gh API GraphQL
  /mold              # Template engine, flux loading, ingot resolution
  /plugin            # Plugin generation pipeline
  /safepath          # Safe path utilities
  /smelt             # Mold packaging (tarball/binary)
  /styles            # Terminal UI styles (lipgloss)
/docs                # Documentation
```

</details>

## Contributing

Community input welcome. As AI development practices mature, Ailloy aims to be the standard package manager for AI instructions — the way Helm became the standard for Kubernetes.

### Prerequisites

**Recommended:** [Flox](https://flox.dev/) provides a reproducible dev environment with all dependencies.

```bash
curl -fsSL https://flox.dev/install | bash    # one-time
flox activate
make check-deps
```

<details>
<summary><strong>Manual installation</strong></summary>

| Dependency | Purpose | Install |
|------------|---------|---------|
| [Go 1.24+](https://go.dev/dl/) | Build the CLI | `brew install go` |
| [golangci-lint](https://golangci-lint.run/) | Linting | `brew install golangci-lint` |
| [lefthook](https://github.com/evilmartians/lefthook) | Git hooks | `brew install lefthook` |
| [Docker](https://www.docker.com/) | Local CI | [Docker Desktop](https://www.docker.com/products/docker-desktop/) |
| [act](https://github.com/nektos/act) | Run GitHub Actions locally | `brew install act` |
| [gh](https://cli.github.com/) | GitHub CLI | `brew install gh` |

</details>

### Development setup

```bash
git clone https://github.com/nimble-giant/ailloy
cd ailloy
flox activate          # or install deps manually
make hooks             # install git hooks
make build
make test
make lint
make ci                # full CI run locally (requires Docker)
```

### Submitting changes

1. Fork the repository
2. Create a feature branch
3. Add blanks or improve CLI functionality
4. Ensure `make test` and `make lint` pass
5. Submit a pull request

## Contributors

<a href="https://github.com/nimble-giant/ailloy/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=nimble-giant/ailloy" alt="Contributors" />
</a>

## License

Apache License 2.0 — see [LICENSE.md](LICENSE.md).
