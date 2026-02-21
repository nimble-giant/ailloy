# Packaging Molds with `ailloy smelt`

The `smelt` command packages a mold directory into a distributable `.tar.gz` archive. It follows the same pattern as Helm's chart packaging: lean metadata in `mold.yaml`, default values in `flux.yaml`, and optional validation in `flux.schema.yaml`.

## Directory Structure

A mold directory should look like this:

```
my-mold/
├── mold.yaml                # Required - metadata
├── flux.yaml                # Optional - default variable values
├── flux.schema.yaml         # Optional - validation rules
├── .claude/
│   ├── commands/
│   │   └── my-command.md    # Command templates
│   └── skills/
│       └── my-skill.md      # Skill templates
├── .github/
│   └── workflows/
│       └── ci.yml           # Workflow templates
└── ingots/                  # Optional - template partials
    └── my-ingot/
        ├── ingot.yaml
        └── partial.md
```

## Step 1: Write `mold.yaml`

This is lean metadata listing what your mold contains:

```yaml
apiVersion: v1
kind: mold
name: my-team-mold
version: 1.0.0
description: "Our team's Claude Code templates"
author:
  name: My Team
  url: https://github.com/my-org
requires:
  ailloy: ">=0.2.0"
commands:
  - my-command.md
skills:
  - my-skill.md
workflows:
  - ci.yml
```

## Step 2: Write `flux.yaml` (optional)

Default values for template variables, like Helm's `values.yaml`. Flat key-value pairs:

```yaml
organization: my-org
default_board: Engineering
scm_provider: GitHub
scm_cli: gh
scm_base_url: https://github.com
```

Templates reference these as `{{ .organization }}`, `{{ .default_board }}`, etc.

Multiline values use YAML block syntax:

```yaml
my_command: |-
  gh api
    --method POST
    repos/{owner}/{repo}/issues
```

If you omit `flux.yaml`, smelt will generate one from any `flux:` declarations in `mold.yaml` (backwards compatibility).

## Step 3: Write `flux.schema.yaml` (optional)

Only declare variables that need validation. You don't need to list every variable from `flux.yaml`:

```yaml
- name: organization
  type: string
  required: true
  description: "GitHub org name"
- name: default_board
  type: string
```

Supported types: `string`, `bool`, `int`, `list`.

When present, `flux.schema.yaml` is used for validation during `forge` and `cast`. If absent, ailloy falls back to any `flux:` declarations in `mold.yaml`. If neither exists, no validation is performed.

## Step 4: Create your templates

Add command templates to `.claude/commands/`, skill templates to `.claude/skills/`, and workflow files to `.github/workflows/`. Reference flux variables with Go template syntax:

```markdown
# My Command

Use `{{ .scm_cli }}` to interact with {{ .scm_provider }}.

Organization: {{ .organization }}
```

## Step 5: Package it

```bash
ailloy smelt ./my-mold
```

Output:

```
Smelting mold...
Smelted: my-team-mold-1.0.0.tar.gz (4.2 KB)
```

To write the tarball to a specific directory:

```bash
ailloy smelt ./my-mold --output ./dist
```

If you omit the path, smelt defaults to the current directory:

```bash
cd my-mold/
ailloy smelt
```

The alias `ailloy package` also works.

## What goes in the tarball

The archive includes all files referenced by `mold.yaml`:

- `mold.yaml`
- `flux.yaml` (source file if present, otherwise generated from `flux:` declarations)
- `flux.schema.yaml` (if present)
- All command templates listed under `commands:`
- All skill templates listed under `skills:`
- All workflow files listed under `workflows:`
- Everything in the `ingots/` directory (if present)

The tarball is named `{name}-{version}.tar.gz` and entries are prefixed with `{name}-{version}/`.

## CLI Reference

```
ailloy smelt [mold-dir] [flags]
```

| Argument | Default | Description |
|----------|---------|-------------|
| `mold-dir` | `.` (current directory) | Path to the mold directory |

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output` | | `.` (current directory) | Output directory for the archive |
| `--output-format` | `-o` | `tar` | Output format (`tar` or `binary`) |

## Using a Mold

After packaging (or directly from source), use `forge` to preview and `cast` to install:

```bash
# Preview rendered output (dry run, like helm template)
ailloy forge ./my-mold

# Write rendered output to a specific directory
ailloy forge ./my-mold -o /tmp/preview

# Install templates into the current project
ailloy cast ./my-mold

# Override flux values at install time
ailloy forge ./my-mold --set organization=my-org --set scm_provider=GitLab
```

## Value Precedence

When a mold is installed with `forge` or `cast`, flux values are resolved in this order (lowest to highest priority):

1. `mold.yaml` `flux:` schema defaults (backwards compatibility)
2. `flux.yaml` defaults
3. Project config (`ailloy.yaml` / `.ailloyrc`)
4. Global config (`~/.ailloy/`)
5. `--set` flags
