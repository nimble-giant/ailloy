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
│   │   └── my-command.md    # Command blanks
│   └── skills/
│       └── my-skill.md      # Skill blanks
├── .github/
│   └── workflows/
│       └── ci.yml           # Workflow blanks
└── ingots/                  # Optional - ingot partials
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
description: "Our team's Claude Code blanks"
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

Default values for flux variables, like Helm's `values.yaml`. Use nested YAML to group related values:

```yaml
project:
  organization: my-org
  board: Engineering

scm:
  provider: GitHub
  cli: gh
  base_url: https://github.com
```

Blanks reference nested values with dotted paths: `{{ scm.provider }}`, `{{ project.board }}`, etc.

Multiline values use YAML block syntax:

```yaml
api:
  post_review: |-
    gh api repos/:owner/:repo/pulls/<pr-number>/reviews \
      --method POST \
      --field body="<summary>"
```

If you omit `flux.yaml`, smelt will generate one from any `flux:` declarations in `mold.yaml` (backwards compatibility).

## Step 3: Write `flux.schema.yaml` (optional)

Only declare variables that need validation. You don't need to list every variable from `flux.yaml`:

```yaml
- name: project.organization
  type: string
  required: true
  description: "GitHub org name"
- name: project.board
  type: string
```

Supported types: `string`, `bool`, `int`, `list`.

When present, `flux.schema.yaml` is used for validation during `forge` and `cast`. If absent, ailloy falls back to any `flux:` declarations in `mold.yaml`. If neither exists, no validation is performed.

## Step 4: Create your blanks

Add command blanks to `.claude/commands/`, skill blanks to `.claude/skills/`, and workflow files to `.github/workflows/`. Reference flux variables with Go template syntax:

```markdown
# My Command

Use `{{ scm.cli }}` to interact with {{ scm.provider }}.

Organization: {{ project.organization }}
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
- All command blanks listed under `commands:`
- All skill blanks listed under `skills:`
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

# Install blanks into the current project
ailloy cast ./my-mold

# Override flux values at install time
ailloy forge ./my-mold --set project.organization=my-org --set scm.provider=GitLab
```

## Value Precedence

When a mold is installed with `forge` or `cast`, flux values are resolved in this order (lowest to highest priority):

1. `mold.yaml` `flux:` schema defaults (backwards compatibility)
2. `flux.yaml` defaults
3. Project config (`ailloy.yaml` / `.ailloyrc`)
4. Global config (`~/.ailloy/`)
5. `--set` flags
