# Flux Variables

Flux is Ailloy's configuration system for templating blanks with project-specific values. It works like Helm's values — define defaults in your mold, override them at install time, and reference them in blanks with Go template syntax.

## File Layout

A mold can define flux variables in up to three places:

| File | Purpose | Required |
|------|---------|----------|
| `flux.yaml` | Default values and output mappings | Optional |
| `flux.schema.yaml` | Type validation and wizard prompts | Optional |
| `mold.yaml` `flux:` section | Inline variable declarations (fallback) | Optional |

### `flux.yaml`

Default values for your mold's template variables, plus the `output:` mapping that controls where files are installed. This is the primary configuration file:

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

### `flux.schema.yaml`

Type information and validation rules for flux variables. When present, it drives the interactive wizard during `ailloy anneal` and enables validation during `cast` and `forge`:

```yaml
- name: project.organization
  type: string
  required: true
  description: "GitHub org name"
- name: project.board
  type: string
  description: "Default project board"
- name: scm.provider
  type: select
  description: "Source control provider"
  options:
    - label: GitHub
      value: github
    - label: GitLab
      value: gitlab
```

### `mold.yaml` `flux:` section

An alternative to `flux.schema.yaml` — declare variables inline in the manifest. If both exist, `flux.schema.yaml` takes precedence at runtime:

```yaml
apiVersion: v1
kind: mold
name: my-mold
version: 1.0.0
flux:
  - name: project.organization
    type: string
    required: true
```

## Value Precedence

When blanks are rendered with `forge` or `cast`, flux values are resolved in this order (lowest to highest priority):

1. **`mold.yaml` `flux:` schema defaults** — Default values from inline declarations
2. **`flux.yaml` defaults** — Values shipped with the mold
3. **`-f, --values` files** — Override files passed at install time (left to right, later files win)
4. **`--set` flags** — Highest priority, set individual values from the command line

```bash
# Layer 3: -f file overrides
ailloy cast ./my-mold -f team-values.yaml -f env-overrides.yaml

# Layer 4: --set overrides (highest priority)
ailloy cast ./my-mold --set project.organization=my-org --set scm.provider=GitLab

# Combined
ailloy cast ./my-mold -f team-values.yaml --set project.organization=my-org
```

## Nested Values and Dotted Paths

Flux values use standard YAML nesting. In blanks, reference them with dotted paths:

```yaml
# flux.yaml
project:
  organization: my-org
  board: Engineering
scm:
  provider: GitHub
```

```markdown
# In a blank
Organization: {{ project.organization }}
Board: {{ project.board }}
Provider: {{ scm.provider }}
```

The `--set` flag also uses dotted paths to set nested values:

```bash
ailloy cast ./my-mold --set project.organization=my-org
# Creates: project: { organization: my-org }
```

## Schema Types

The `type` field in `flux.schema.yaml` (or `mold.yaml` `flux:`) controls validation and wizard prompts:

| Type | Description | Wizard Prompt | Validation |
|------|-------------|---------------|------------|
| `string` | Any text value | Text input | Any non-empty string |
| `bool` | Boolean flag | Toggle | Must be `true` or `false` |
| `int` | Integer number | Numeric input | Must parse as integer |
| `list` | Comma-separated values | Text input | Non-empty string |
| `select` | Fixed set of choices | Dropdown | Any value (runtime check) |

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

### Schema discovery

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
| `parse` | No | Go template applied to JSON output. Each line should be `label\|value` or just `value`. Extra segments beyond `label\|value` are available to `also_sets`. If omitted, each line of stdout becomes an option. |
| `prompt` | No | `"select"` for a dropdown, `"input"` for freeform text (default). |
| `also_sets` | No | Maps flux variable names to extra segment indices (0-based). A single selection can populate multiple variables. |

Discovery commands run lazily during `ailloy anneal` when the user reaches the relevant wizard section. If a command's template dependencies (e.g., `{{.project.organization}}`) are not yet populated, the wizard shows a waiting placeholder until the user fills them in. If a command fails, the wizard falls back to manual input with a warning.

## Output Mapping

The `output:` key in `flux.yaml` defines where each source directory in your mold maps to in the target project. It supports three forms:

### Map output (recommended)

Each key maps a source directory to a destination path:

```yaml
output:
  commands: .claude/commands
  skills: .claude/skills
```

### Expanded map

Per-directory options like disabling template processing:

```yaml
output:
  commands: .claude/commands
  workflows:
    dest: .github/workflows
    process: false          # skip Go template processing
```

### String output

All top-level directories go under a single parent:

```yaml
output: .claude
# commands/ → .claude/commands/
# skills/ → .claude/skills/
```

### No output key

Files are placed at their source paths (identity mapping):

```yaml
# omitting output: means commands/my-cmd.md → commands/my-cmd.md
```

Since `output:` lives in flux, consumers can override destination paths using `-f` value files or `--set` flags. This makes the same mold portable across different AI coding tools:

```yaml
# team-claude.yaml — for Claude Code users
output:
  commands: .claude/commands
  skills: .claude/skills

# team-cursor.yaml — for Cursor users
output:
  rules: .cursor/rules

# team-windsurf.yaml — for Windsurf users
output:
  rules: .windsurf/rules
```

```bash
# Install for Claude Code (default)
ailloy cast ./my-mold

# Install for Cursor
ailloy cast ./my-mold -f team-cursor.yaml
```

For more multi-tool targeting patterns, see [Targeting Different AI Tools](blanks.md#targeting-different-ai-tools).

The `ingots/` directory and hidden directories (starting with `.`) are always excluded from output resolution.

## Validation

When a `flux.schema.yaml` (or `mold.yaml` `flux:` section) is present, Ailloy validates flux values during `cast` and `forge`:

- **Required fields** — Variables marked `required: true` must be present and non-empty
- **Type checking** — Values must match their declared type (e.g., bools must be `true`/`false`, ints must parse as integers)
- **Select type** — Select variables must have `options` or a `discover` block
- **Discovery** — Discovery blocks must have a `command` field; `prompt` must be `"select"` or `"input"`

Validation errors are logged as warnings during rendering. Use `ailloy temper` to run full validation before distributing your mold. See the [Validation guide](temper.md) for details.

## Examples

### Minimal flux.yaml

```yaml
output:
  commands: .claude/commands
```

### Full flux.yaml

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
  number: "42"

scm:
  provider: GitHub
  cli: gh
  base_url: https://github.com

api:
  post_review: |-
    gh api repos/:owner/:repo/pulls/<pr-number>/reviews \
      --method POST \
      --field body="<summary>"
```

### Override file (team-values.yaml)

```yaml
project:
  organization: acme-corp
  board: Platform Team

scm:
  provider: GitLab
  cli: glab
  base_url: https://gitlab.com
```

```bash
ailloy cast ./my-mold -f team-values.yaml
```
