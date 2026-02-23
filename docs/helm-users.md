# Ailloy for Helm Users

If you know Helm, you already know most of Ailloy. The core workflow — author a package of templates, configure them with values, render or install — is the same. Ailloy adapts this pattern for AI instruction files instead of Kubernetes manifests, and adds a few concepts of its own.

This guide maps what you already know to Ailloy terminology, shows every command written both ways (Ailloy-native and Helm-coded alias), and highlights what's new.

## Concept Map

| Helm | Ailloy | Notes |
|------|--------|-------|
| Chart | Mold | A versioned, configurable package of templates |
| `Chart.yaml` | `mold.yaml` | Package metadata and version constraints |
| Template | Blank | Markdown instruction files with Go template syntax |
| `values.yaml` | `flux.yaml` | Default configuration values shipped with the package |
| Values | Flux | The configuration layer — variables that drive templates |
| `--set` / `-f` | `--set` / `-f` | Same flags, same layering semantics |
| Repository | Foundry | Where packages are discovered and resolved from |
| Subchart / Dependency | Ingot | Reusable template partials included via `{{ingot "name"}}` |
| `Chart.lock` | `ailloy.lock` | Pins dependencies to exact versions and commits |
| — | Output mapping | Maps source directories to destination paths (no Helm equivalent) |
| — | Anneal | Interactive configuration wizard (no Helm equivalent) |

## Command Equivalences

Ailloy provides Helm-coded aliases for every core command, so you can use whichever name feels natural. Both forms are identical — same flags, same behavior.

| What You Want to Do | Helm | Ailloy (native) | Ailloy (Helm alias) |
|---------------------|------|------------------|---------------------|
| Install a package | `helm install` | `ailloy cast` | `ailloy install` |
| Dry-run / render templates | `helm template` | `ailloy forge` | `ailloy template` |
| Lint / validate | `helm lint` | `ailloy temper` | `ailloy lint` |
| Package for distribution | `helm package` | `ailloy smelt` | `ailloy package` |
| Upgrade dependencies | `helm upgrade` | `ailloy recast` | `ailloy upgrade` |
| Lock dependency versions | — | `ailloy quench` | `ailloy lock` |
| Interactive configuration | — | `ailloy anneal` | `ailloy configure` |
| Scaffold a new package | `helm create` | `ailloy mold new` | `ailloy mold create` |

## Side-by-Side Examples

Every example below shows the Ailloy-native command first, then the Helm-coded alias. They do exactly the same thing.

### Install a mold into your project

```bash
# Ailloy-native
ailloy cast ./my-mold

# Helm alias
ailloy install ./my-mold
```

### Install from a remote repository

```bash
# Ailloy-native
ailloy cast github.com/my-org/my-mold@v1.0.0

# Helm alias
ailloy install github.com/my-org/my-mold@v1.0.0
```

### Render templates without installing (dry run)

```bash
# Ailloy-native
ailloy forge ./my-mold

# Helm alias
ailloy template ./my-mold
```

### Override values at install time

The `-f` and `--set` flags work exactly like Helm:

```bash
# Ailloy-native
ailloy cast ./my-mold -f team-values.yaml --set project.organization=acme

# Helm alias
ailloy install ./my-mold -f team-values.yaml --set project.organization=acme
```

### Lint a package

```bash
# Ailloy-native
ailloy temper ./my-mold

# Helm alias
ailloy lint ./my-mold
```

### Package a mold for distribution

```bash
# Ailloy-native
ailloy smelt ./my-mold

# Helm alias
ailloy package ./my-mold
```

### Upgrade locked dependencies

```bash
# Ailloy-native
ailloy recast

# Helm alias
ailloy upgrade
```

### Scaffold a new mold

```bash
# Ailloy-native
ailloy mold new my-team-mold

# Helm alias
ailloy mold create my-team-mold
```

## Value Precedence (Same as Helm)

Ailloy resolves flux values in the same order Helm resolves chart values:

1. **Schema defaults** — `mold.yaml` `flux:` declarations (like `Chart.yaml` defaults)
2. **`flux.yaml` defaults** — shipped with the mold (like `values.yaml`)
3. **`-f` override files** — left to right, later files win (identical to Helm)
4. **`--set` flags** — highest priority (identical to Helm)

```bash
# Layer overrides just like Helm
ailloy cast ./my-mold -f base.yaml -f env-overrides.yaml --set project.organization=acme
```

## Template Syntax (Same Engine, Simpler Defaults)

Both tools use Go's `text/template`. Ailloy adds a preprocessor that auto-prefixes the dot, so you can skip it for simple variables:

```markdown
<!-- Helm style (works in Ailloy too) -->
Organization: {{ .project.organization }}

<!-- Ailloy shorthand (preprocessor adds the dot for you) -->
Organization: {{ project.organization }}
```

Conditionals, ranges, and other Go template keywords work the same way:

```markdown
{{if .feature.enabled}}
## Feature Section
This content is included when feature.enabled is true.
{{end}}

{{range $name, $url := .endpoints}}
- {{ $name }}: {{ $url }}
{{end}}
```

## Package Structure Comparison

The file layout will feel familiar:

```
Helm Chart                          Ailloy Mold
─────────                           ───────────
my-chart/                           my-mold/
├── Chart.yaml          →           ├── mold.yaml
├── Chart.lock          →           ├── ailloy.lock
├── values.yaml         →           ├── flux.yaml
├── values.schema.json  →           ├── flux.schema.yaml
├── templates/          →           ├── commands/
│   ├── deployment.yaml →           │   └── deploy-checklist.md
│   └── _helpers.tpl    →           ├── skills/
├── charts/             →           │   └── code-review-style.md
│   └── subchart/       →           ├── ingots/
└── README.md                       │   └── team-preamble/
                                    └── README.md
```

Key differences:
- **Multiple template directories**: Ailloy organizes blanks by type (`commands/`, `skills/`, `workflows/`) instead of a single `templates/` directory
- **YAML schema**: `flux.schema.yaml` uses YAML instead of JSON Schema
- **Output mapping**: `flux.yaml` includes an `output:` key that maps source directories to destination paths — this replaces Helm's hardcoded output to `templates/`

## What's Familiar

### Manifests

Helm's `Chart.yaml` maps directly to `mold.yaml`:

```yaml
# Helm Chart.yaml                    # Ailloy mold.yaml
apiVersion: v2                        apiVersion: v1
name: my-chart                        kind: mold
version: 1.0.0                        name: my-mold
description: "My chart"               version: 1.0.0
                                      description: "My mold"
```

### Values layering

The `-f` and `--set` flags use the same semantics. Dotted paths set nested values:

```bash
# Helm
helm install my-release ./my-chart --set service.port=8080

# Ailloy
ailloy cast ./my-mold --set project.organization=acme
```

### Packaging and linting

Same two-step validation-then-package workflow:

```bash
# Helm                              # Ailloy (native)       # Ailloy (alias)
helm lint ./my-chart                ailloy temper ./my-mold  ailloy lint ./my-mold
helm package ./my-chart             ailloy smelt ./my-mold   ailloy package ./my-mold
```

### Dependencies and lock files

Both tools pin dependency versions in a lock file. Ailloy's `ailloy.lock` captures the exact commit SHA alongside the version:

```yaml
# ailloy.lock
apiVersion: v1
molds:
  - name: nimble-mold
    source: github.com/nimble-giant/nimble-mold
    version: v0.1.10
    commit: 2347a626798553252668a15dc98dd020ab9a9c0c
    timestamp: 2026-02-21T19:30:00Z
```

## What's New

These features have no direct Helm equivalent.

### Foundries: Your SCM Is the Registry

Helm charts live in dedicated chart repositories (OCI registries, `index.yaml` repos). Ailloy skips the registry entirely — any git repository with a `mold.yaml` is a valid foundry. Versions are git tags:

```bash
# Install directly from a git repo — no helm repo add step
ailloy cast github.com/my-org/my-mold@v1.0.0

# Semver constraints work like you'd expect
ailloy cast github.com/my-org/my-mold@^1.0.0

# Monorepo subpaths
ailloy cast github.com/my-org/mono-repo@v1.0.0//molds/frontend

# Private repos work if git auth is configured
ailloy cast github.com/my-org/private-mold@v1.0.0
```

No `helm repo add`, no `helm repo update`, no managing repository indexes. If you can `git clone` it, you can `ailloy cast` it. See the [Foundry guide](foundry.md) for details.

### Anneal: Interactive Configuration Wizard

Helm has no built-in way to interactively configure values. Ailloy's `anneal` command reads your mold's schema and generates a terminal wizard with type-driven prompts:

```bash
# Launch the wizard — generates a values override file
ailloy anneal github.com/my-org/my-mold -o team-values.yaml

# Or use the alias
ailloy configure github.com/my-org/my-mold -o team-values.yaml

# Then install with those values
ailloy cast github.com/my-org/my-mold -f team-values.yaml
```

The wizard supports dynamic discovery — flux variables can declare shell commands that populate dropdown options at runtime (e.g., listing GitHub project boards via the API). See the [Anneal guide](anneal.md) for details.

### Output Mapping

Helm always renders templates into a fixed output. Ailloy's `output:` key in `flux.yaml` lets mold authors (and consumers) control exactly where files land:

```yaml
output:
  commands: .claude/commands
  skills: .claude/skills
  workflows:
    dest: .github/workflows
    process: false           # skip template processing for raw YAML
```

Since `output:` lives in flux, consumers can override it with `-f` to target different AI coding tools from the same mold:

```bash
# Default: install for Claude Code
ailloy cast github.com/my-org/my-mold

# Override output for Cursor
ailloy cast github.com/my-org/my-mold -f cursor-output.yaml
```

### Binary Packaging

Beyond tarballs, `ailloy smelt` can produce a self-contained binary with the mold baked in:

```bash
ailloy smelt -o binary ./my-mold
# → my-team-mold-1.0.0 (executable)

# Recipients don't need ailloy installed
./my-team-mold-1.0.0 cast
./my-team-mold-1.0.0 cast --set project.organization=acme
```

### Ingot Includes

Helm uses `_helpers.tpl` with `{{- include "helper" . -}}` and `{{ template }}`. Ailloy uses ingots — self-contained template partials included with a simpler syntax:

```markdown
{{ingot "team-preamble"}}
```

Ingots resolve from three locations (mold-local → project → global), making them easy to share across molds. See the [Ingots guide](ingots.md) for details.

## Quick Reference Card

```bash
# Scaffold a new mold
ailloy mold new my-mold          # or: ailloy mold create my-mold

# Preview rendered output
ailloy forge ./my-mold            # or: ailloy template ./my-mold

# Install into your project
ailloy cast ./my-mold             # or: ailloy install ./my-mold

# Install from remote with overrides
ailloy cast github.com/org/mold@v1.0.0 -f vals.yaml --set key=value
ailloy install github.com/org/mold@v1.0.0 -f vals.yaml --set key=value

# Validate
ailloy temper ./my-mold           # or: ailloy lint ./my-mold

# Package
ailloy smelt ./my-mold            # or: ailloy package ./my-mold

# Configure interactively
ailloy anneal ./my-mold -o vals.yaml  # or: ailloy configure ./my-mold -o vals.yaml

# Upgrade locked dependencies
ailloy recast                     # or: ailloy upgrade

# Pin dependency versions
ailloy quench                     # or: ailloy lock
```
