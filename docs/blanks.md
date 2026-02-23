# Blanks

Blanks are the source files of the Ailloy compiler. They are Markdown instruction templates that live in mold directories, define commands and skills for your AI coding tool, and are compiled with flux variables via `ailloy forge` (dry-run) or `ailloy cast` (install).

## Blank Types

Ailloy supports three types of blanks, each serving a different purpose:

### Commands

Command blanks define commands that users invoke explicitly in their AI coding tool. They live in your mold's `commands/` directory and are installed to the destination configured in your `flux.yaml` output mapping (e.g., `.claude/commands/` for Claude Code).

```
my-mold/
└── commands/
    ├── brainstorm.md
    ├── create-issue.md
    └── open-pr.md
```

After `ailloy cast`, each file becomes available as a command in your AI coding tool (e.g., `/brainstorm`, `/create-issue` in Claude Code).

### Skills

Skill blanks define proactive workflows that your AI coding tool uses automatically based on context, without requiring explicit command invocation. They live in your mold's `skills/` directory and are installed to the destination configured in your output mapping (e.g., `.claude/skills/` for Claude Code).

```
my-mold/
└── skills/
    └── code-review-style.md
```

Skills are ideal for instructions that should always be available — coding standards, review guidelines, or domain-specific knowledge.

### Workflows

Workflow blanks are GitHub Actions YAML files. They live in your mold's `workflows/` directory and are installed to `.github/workflows/`. Because workflow files contain raw YAML syntax that conflicts with Go template delimiters, they are typically configured with `process: false` in the output mapping:

```yaml
output:
  workflows:
    dest: .github/workflows
    process: false
```

Workflow blanks are only installed when using `ailloy cast --with-workflows`.

## Creating Your First Blank

### 1. Set up a mold directory

```bash
mkdir my-mold && cd my-mold
```

### 2. Write `mold.yaml`

```yaml
apiVersion: v1
kind: mold
name: my-team-mold
version: 1.0.0
description: "My team's AI workflow blanks"
author:
  name: My Team
  url: https://github.com/my-org
```

### 3. Write `flux.yaml` with output mapping

The `output:` key maps source directories in your mold to destination paths in the target project:

```yaml
output:
  commands: .claude/commands
  skills: .claude/skills

project:
  organization: my-org

scm:
  provider: GitHub
  cli: gh
```

### 4. Create a command blank

```bash
mkdir -p commands
```

Write `commands/deploy-checklist.md`:

```markdown
# Deploy Checklist

Generate a deployment checklist for {{ project.organization }}.

## Steps

1. Use `{{ scm.cli }}` to check for open PRs targeting the release branch
2. Verify all CI checks are passing
3. List recent commits since last deploy
4. Generate a summary of changes
```

### 5. Preview with `forge`

```bash
ailloy forge ./my-mold
```

This renders all blanks with your flux values and prints the output — a dry run that lets you verify templates before installing.

### 6. Install with `cast`

```bash
ailloy cast ./my-mold
```

This compiles and installs the rendered blanks into the directories defined by your `output:` mapping (e.g., `.claude/commands/` and `.claude/skills/` for Claude Code).

## Template Syntax

Blanks use Go's [text/template](https://pkg.go.dev/text/template) engine with a preprocessing step that simplifies variable references.

### Simple variables

Use `{{ variable }}` to reference top-level flux values. The preprocessor automatically adds the Go template dot prefix, so you don't need to write `{{ .variable }}`:

```markdown
Organization: {{ project.organization }}
CLI tool: {{ scm.cli }}
```

Both `{{ variable }}` and `{{ .variable }}` work — use whichever you prefer.

### Dotted path access

Nested flux values are accessed with dotted paths:

```markdown
Provider: {{ scm.provider }}
Board: {{ project.board }}
Status field: {{ ore.status.field_id }}
```

### Conditionals

Use Go template conditionals (these require the dot prefix since they use Go template keywords):

```markdown
{{if .ore.status.enabled}}
## Status Tracking

Update the status field ({{ .ore.status.field_id }}) after each step.
{{end}}
```

### Ranges

Iterate over lists:

```markdown
{{range $key, $value := .items}}
- {{ $key }}: {{ $value }}
{{end}}
```

### Including ingots

Use the `{{ingot "name"}}` function to include reusable template partials:

```markdown
# My Command

## Standard Preamble
{{ingot "team-preamble"}}

## Command-Specific Instructions
...
```

The ingot's content is rendered through the same template engine with the same flux context. See the [Ingots guide](ingots.md) for details on creating and managing ingots.

### Preprocessor rules

The preprocessor converts simple `{{variable}}` references to `{{.variable}}` before Go template parsing. It skips Go template keywords (`if`, `else`, `end`, `range`, `with`, `define`, `block`, `template`, `ingot`, `not`, `and`, `or`, `eq`, `ne`, `lt`, `le`, `gt`, `ge`, `len`, `index`, `print`, `printf`, `println`, `call`, `nil`, `true`, `false`) so they are not dot-prefixed.

### Unresolved variables

Variables that cannot be resolved from the flux context produce a logged warning and resolve to empty strings. The template does not fail — this allows progressive development where not all variables need to be set immediately.

## Flux Variables in Blanks

Blanks reference values defined in `flux.yaml` (or overridden via `-f` files and `--set` flags). See the [Flux guide](flux.md) for full details on defining and layering values.

### Value precedence

When blanks are rendered, flux values are resolved in this order (lowest to highest priority):

1. `mold.yaml` `flux:` schema defaults
2. `flux.yaml` defaults
3. `-f, --values` flux overrides (left to right)
4. `--set` flags

### Multiline values

Use YAML block syntax for multiline flux values:

```yaml
api:
  post_review: |-
    gh api repos/:owner/:repo/pulls/<pr-number>/reviews \
      --method POST \
      --field body="<summary>"
```

Then reference in blanks: `{{ api.post_review }}`

## Blank Discovery

Blanks are automatically discovered from the mold's `output:` mapping in `flux.yaml`. The `ResolveFiles` function walks each mapped directory and collects all files:

- **Map output** — each key maps a source directory to a destination
- **String output** — all top-level directories go under the specified parent
- **No output key** — files are placed at their source paths (identity mapping)

The `ingots/` directory and hidden directories (starting with `.`) are always excluded from auto-discovery.

## Testing and Previewing

### Dry-run render

Preview what `cast` will produce without writing any files:

```bash
ailloy forge ./my-mold
ailloy forge ./my-mold --set project.organization=my-org
ailloy forge ./my-mold -o /tmp/preview  # write to directory
```

### Validation

Check your mold's structure, manifests, and template syntax:

```bash
ailloy temper ./my-mold
```

This catches template syntax errors, missing manifest fields, and broken file references before you distribute your mold. See the [Validation guide](temper.md) for details.

## Getting Started with Examples

The [official mold](https://github.com/nimble-giant/nimble-mold) provides a complete reference implementation with command, skill, and workflow blanks. It's a good starting point for understanding blank structure and conventions. For a step-by-step guide to creating a full mold from scratch, see the [Packaging Molds guide](smelt.md).

## Targeting Different AI Tools

The `output:` mapping in `flux.yaml` determines where blanks are installed, making the same mold portable across AI coding tools. Change the output paths to target your tool of choice:

### Claude Code

```yaml
output:
  commands: .claude/commands
  skills: .claude/skills
```

### Cursor

```yaml
output:
  rules: .cursor/rules
```

### Windsurf

```yaml
output:
  rules: .windsurf/rules
```

### Generic (agents.md compatible)

The [agents.md](https://agents.md) format is supported by many AI coding tools. Place instructions at the project root:

```yaml
output:
  agents: .
```

### Multi-tool

You can target multiple tools from the same mold by mapping source directories to multiple destinations:

```yaml
output:
  commands: .claude/commands
  skills: .claude/skills
  cursor-rules: .cursor/rules
```

Since `output:` lives in flux, consumers can override destination paths at install time using `-f` value files or `--set` flags. This means a single mold can serve teams using different AI coding tools.
