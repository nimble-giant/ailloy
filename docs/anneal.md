# Configuration Wizard (`ailloy anneal`)

The `anneal` command provides an interactive, mold-aware wizard for configuring flux variables. It reads your mold's schema to generate type-driven prompts, optionally running discovery commands to populate options dynamically. The result is a YAML file you pass to `cast` or `forge` via the `-f` flag.

Alias: `configure`

## Quick Start

```bash
# Interactive wizard — generates a flux override file
ailloy anneal ./my-mold -o my-values.yaml

# Use the overrides when casting
ailloy cast ./my-mold -f my-values.yaml
```

## Interactive Mode

When run without `--set` flags, anneal launches a terminal wizard that walks through each flux variable:

```bash
ailloy anneal ./my-mold -o my-values.yaml
```

The wizard generates prompts based on the variable's type:

| Type | Prompt Style |
|------|-------------|
| `string` | Text input |
| `bool` | Toggle (true/false) |
| `int` | Numeric input |
| `select` | Dropdown with static options |
| `list` | Text input (comma-separated) |

At the end, the wizard presents **Save** and **Cancel** options:

- **Save** — Writes the YAML file to the specified output path (or the mold's `flux.yaml` if no `-o` is given)
- **Cancel** — Prints the result to stdout for inspection without writing to disk

### Schema Discovery

Variables with a `discover` block run shell commands to populate options dynamically. For example, a variable that discovers GitHub project boards:

```yaml
# flux.schema.yaml
- name: project.id
  type: string
  description: "Default project board"
  discover:
    command: "gh api graphql -f query='...' -f org='{{.project.organization}}'"
    parse: |
      {{- range .data.organization.projectsV2.nodes -}}
      {{ .title }}|{{ .id }}
      {{ end -}}
    prompt: select
```

Discovery commands run lazily — they execute when the wizard reaches that variable. If a command depends on a variable the user hasn't filled in yet (e.g., `{{.project.organization}}`), the wizard shows a placeholder until the dependency is satisfied.

If a discovery command fails, the wizard falls back to manual input with a warning.

## Scripted Mode

Use `--set` flags to skip the wizard entirely. This is useful for CI/CD or automation:

```bash
# Set values directly
ailloy anneal --set project.organization=my-org --set scm.provider=GitHub -o my-values.yaml

# No mold required in scripted mode
ailloy anneal -s project.organization=my-org -o my-values.yaml
```

## Schema Resolution

Anneal resolves the schema in this order (first match wins):

1. **`flux.schema.yaml`** — Dedicated schema file with type information, descriptions, and discovery specs
2. **`mold.yaml` `flux:` section** — Inline variable declarations in the manifest
3. **Inferred from `flux.yaml`** — When no schema exists, anneal infers types from the current flux values:
   - `true`/`false` values become `bool`
   - Numeric values become `int`
   - Everything else becomes `string`
   - Nested maps are flattened to dotted paths

If none of these are available, anneal reports an error.

## Remote Mold Support

Anneal works with remote mold references just like `cast` and `forge`:

```bash
ailloy anneal github.com/my-org/my-mold@v1.0.0 -o my-values.yaml
```

## CLI Reference

```
ailloy anneal [mold-dir] [flags]
```

| Flag | Short | Description |
|------|-------|-------------|
| `--set key=value` | `-s` | Set flux variable in scripted mode (can be repeated) |
| `--output file` | `-o` | Write flux YAML to file (default: mold's `flux.yaml`) |

## Example Workflow

```bash
# 1. Run the wizard to generate team-specific overrides
ailloy anneal github.com/my-org/my-mold -o team-values.yaml

# 2. Review the generated file
cat team-values.yaml

# 3. Install with overrides
ailloy cast github.com/my-org/my-mold -f team-values.yaml

# 4. Or combine with inline overrides
ailloy cast github.com/my-org/my-mold -f team-values.yaml --set project.board=Platform
```

For more on defining flux variables and schemas, see the [Flux Variables guide](flux.md).
