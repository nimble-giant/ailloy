# Packaging Molds with `ailloy smelt`

The `smelt` command packages a mold directory into a distributable format. It follows the same pattern as Helm's chart packaging: lean metadata in `mold.yaml`, default values in `flux.yaml`, and optional validation in `flux.schema.yaml`.

Two output formats are available:

- **`tar`** (default): A `.tar.gz` archive that can be extracted and used with `ailloy cast ./extracted-mold`
- **`binary`**: A self-contained executable with the mold baked in — run `./my-mold cast` directly

## Directory Structure

A mold directory uses clean top-level directories for source files. The `output:` field in `flux.yaml` defines where each directory maps to in the target project:

```
my-mold/
├── mold.yaml                # Required - metadata
├── flux.yaml                # Optional - default values + output mappings
├── flux.schema.yaml         # Optional - validation rules
├── commands/
│   └── my-command.md        # Command blanks
├── skills/
│   └── my-skill.md          # Skill blanks
├── workflows/
│   └── ci.yml               # Workflow files
└── ingots/                  # Optional - ingot partials
    └── my-ingot/
        ├── ingot.yaml
        └── partial.md
```

## Step 1: Write `mold.yaml`

This is lean metadata — the mold's identity and version constraints:

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
```

## Step 2: Write `flux.yaml` (optional)

Default values for flux variables, like Helm's `values.yaml`. The `output:` key maps source directories in your mold to destination paths in the target project. Use nested YAML to group related values:

```yaml
output:
  commands: .claude/commands
  skills: .claude/skills
  workflows:
    dest: .github/workflows
    process: false

project:
  organization: my-org
  board: Engineering

scm:
  provider: GitHub
  cli: gh
  base_url: https://github.com
```

### Output mapping forms

**Simple map** — directory-to-directory mappings:

```yaml
output:
  commands: .claude/commands
  skills: .claude/skills
```

**Expanded map** — per-directory options like disabling template processing:

```yaml
output:
  workflows:
    dest: .github/workflows
    process: false          # skip Go template processing
```

**No output key** — files are placed at their source paths (identity mapping):

```yaml
# omitting output: means commands/my-cmd.md → commands/my-cmd.md
```

Since output lives in flux, consumers can override destination paths using the standard flux layering (`-f` value files or `--set` flags).

Blanks reference nested values with dotted paths: `{{ scm.provider }}`, `{{ project.board }}`, etc.

Multiline values use YAML block syntax:

```yaml
api:
  post_review: |-
    gh api repos/:owner/:repo/pulls/<pr-number>/reviews \
      --method POST \
      --field body="<summary>"
```

If you omit `flux.yaml`, smelt will generate one from any `flux:` declarations in `mold.yaml`.

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

Supported types: `string`, `bool`, `int`, `list`, `select`.

When present, `flux.schema.yaml` is used for validation during `forge` and `cast`, and drives the interactive wizard during `ailloy anneal`. If absent, ailloy falls back to any `flux:` declarations in `mold.yaml`. If neither exists, no validation is performed.

### Select type

Use `type: select` with static options for variables that have a fixed set of choices:

```yaml
- name: scm.provider
  type: select
  description: "Source control provider"
  options:
    - label: GitHub
      value: github
    - label: GitLab
      value: gitlab
    - label: Bitbucket
      value: bitbucket
```

### Schema discovery (optional)

Flux variables can declare a `discover` block to dynamically populate options from external commands during `ailloy anneal`:

```yaml
- name: project.id
  type: string
  description: "Default project board"
  discover:
    command: "gh api graphql -f query='...' -f org='{{.project.organization}}'"
    parse: |
      {{- range .data.organization.projectsV2.nodes -}}
      {{ .title }}|{{ .id }}|{{ .title }}|{{ .number }}
      {{ end -}}
    also_sets:
      project.board: 0
      project.number: 1
    prompt: select
```

| Field | Required | Description |
|-------|----------|-------------|
| `command` | Yes | Shell command to execute. Supports `{{.variable}}` template expansion against current flux values. |
| `parse` | No | Go template applied to JSON output. Each line should be `label\|value` or just `value`. Extra pipe-delimited segments beyond `label\|value` are available to `also_sets`. If omitted, each line of stdout becomes an option. |
| `prompt` | No | `"select"` for a dropdown, `"input"` for freeform text (default). |
| `also_sets` | No | Maps additional flux variable names to extra segment indices (0-based into segments after `label\|value`). Allows a single selection to populate multiple variables. |

Discovery commands run lazily during `ailloy anneal` when the user reaches the relevant wizard section. If a discovery command's template dependencies (e.g. `{{.project.organization}}`) are not yet populated, the wizard shows a waiting placeholder until the user fills them in. If a command fails, the wizard falls back to manual input with a warning.

## Step 4: Create your blanks

Add command blanks to `commands/`, skill blanks to `skills/`, and workflow files to `workflows/`. The `output:` mapping in `flux.yaml` determines where they end up in the target project. Reference flux variables with Go template syntax:

```markdown
# My Command

Use `{{ scm.cli }}` to interact with {{ scm.provider }}.

Organization: {{ project.organization }}
```

## Step 5: Validate (optional)

Before packaging, validate your mold's structure, manifests, and template syntax:

```bash
ailloy temper ./my-mold
```

This catches errors early — missing manifest fields, broken file references, and template syntax issues. See the [Validation guide](temper.md) for details.

## Step 6: Package it

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

The archive includes all files discovered from the mold directory:

- `mold.yaml`
- `flux.yaml` (source file if present, otherwise generated from `flux:` declarations)
- `flux.schema.yaml` (if present)
- All files in directories referenced by `output:` in `flux.yaml` (or all top-level directories if `output:` is omitted)
- Everything in the `ingots/` directory (if present)

The tarball is named `{name}-{version}.tar.gz` and entries are prefixed with `{name}-{version}/`.

## Binary Output

The binary format creates a self-contained executable by embedding the mold files into a copy of the ailloy binary using [stuffbin](https://github.com/knadh/stuffbin).

### Creating a binary

```bash
ailloy smelt -o binary ./my-mold
```

Output:

```
Smelting mold...
Smelted: my-team-mold-1.0.0 (12.3 MB)
```

To write to a specific directory:

```bash
ailloy smelt -o binary ./my-mold --output ./dist
```

### Using a binary

The output binary is a portable ailloy with a baked-in mold. When `cast` or `forge` is run without a mold-dir argument, the embedded mold is used automatically:

```bash
# Cast the ailloy into a project
./my-team-mold-1.0.0 cast

# Preview the rendered output
./my-team-mold-1.0.0 forge

# Override flux values
./my-team-mold-1.0.0 cast --set project.organization=my-org

# Layer additional flux files
./my-team-mold-1.0.0 cast -f production.yaml
```

You can still pass an explicit mold-dir to override the embedded mold:

```bash
./my-team-mold-1.0.0 cast ./other-mold
```

### What goes in the binary

The binary includes the same files as the tarball (see above). The output is named `{name}-{version}` (no extension) and is made executable.

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

1. `mold.yaml` `flux:` schema defaults
2. `flux.yaml` defaults
3. `-f, --values` flux overrides
4. `--set` flags
