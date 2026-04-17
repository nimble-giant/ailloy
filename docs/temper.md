# Validation (`ailloy temper`)

The `temper` command validates mold and ingot packages. It checks manifest fields, file references, template syntax, and flux schema consistency — catching errors before you distribute your package.

Alias: `validate`

> **Note:** To lint rendered AI instruction files (CLAUDE.md, AGENTS.md, Cursor rules, etc.) in an already-cast project, use [`ailloy assay`](assay.md). To lint a mold's output *before* casting, use `ailloy temper --assay`.

## Quick Start

```bash
# Validate a mold
ailloy temper ./my-mold

# Validate an ingot
ailloy temper ./my-ingot

# Validate and lint rendered output in one step
ailloy temper --assay ./my-mold

# Using the alias
ailloy validate ./my-mold
```

If no path is provided, the current directory is used:

```bash
cd my-mold/
ailloy temper
```

## What Gets Checked

### Manifest detection

Temper auto-detects whether the target is a mold or ingot by looking for `mold.yaml` or `ingot.yaml` at the root. If neither is found, it reports an error.

### Mold validation

For molds (`mold.yaml` present), temper checks:

| Check | Severity | Description |
|-------|----------|-------------|
| Manifest parsing | Error | `mold.yaml` must be valid YAML |
| Required fields | Error | `apiVersion`, `kind`, `name`, `version` must be present |
| Kind value | Error | Must be `"mold"` |
| Version format | Error | Must be valid semver (e.g., `1.0.0`) |
| Requires constraint | Error | `requires.ailloy` must be a valid version constraint if set |
| Flux variable types | Error | Each `flux[].type` must be `string`, `bool`, `int`, `list`, or `select` |
| Select options | Error | `select` type requires `options` or `discover` |
| Discovery command | Error | `discover.command` is required when `discover` is present |
| Discovery prompt | Error | `discover.prompt` must be `"select"` or `"input"` if set |
| Dependency format | Error | `dependencies[].ingot` and `dependencies[].version` must be present |
| Output sources | Error | All directories in the `output:` mapping must exist in the mold |
| Template syntax | Error | All `.md` files must have valid Go template syntax |
| Schema consistency | Warning | Warns if flux vars are defined in both `mold.yaml` and `flux.schema.yaml` |

### Ingot validation

For ingots (`ingot.yaml` present), temper checks:

| Check | Severity | Description |
|-------|----------|-------------|
| Manifest parsing | Error | `ingot.yaml` must be valid YAML |
| Required fields | Error | `apiVersion`, `kind`, `name`, `version` must be present |
| Kind value | Error | Must be `"ingot"` |
| Version format | Error | Must be valid semver |
| Requires constraint | Error | `requires.ailloy` must be valid if set |
| File references | Error | All files listed in `files:` must exist |
| Template syntax | Error | All `.md` files must have valid Go template syntax |

## Errors vs Warnings

- **Errors** are blocking — `temper` exits with a non-zero exit code when any error is found
- **Warnings** are informational — they are printed but do not cause failure

Currently the only warning is when flux variables are defined in both `mold.yaml` and `flux.schema.yaml` (the schema file takes precedence at runtime).

## Assaying Rendered Output (`--assay`)

The `--assay` flag renders the mold's blanks into a temporary directory and runs [`ailloy assay`](assay.md) against the output. This catches content-level issues (line count, structure, cross-references, naming conventions) before casting — without needing a separate `cast` + `assay` step.

> **Alias:** `--lint` is accepted as an alias for `--assay`.

```bash
# Validate structure and assay rendered output
ailloy temper --assay ./my-mold

# Provide flux values for rendering (same flags as forge/cast)
ailloy temper --assay -f my-values.yaml ./my-mold
ailloy temper --assay --set project.organization=acme ./my-mold

# Control assay output format and failure threshold
ailloy temper --assay --format json ./my-mold
ailloy temper --assay --fail-on warning ./my-mold

# Override assay line-count threshold
ailloy temper --assay --max-lines 200 ./my-mold
```

When `--assay` is used, temper first runs its normal structural validation. If that passes, it renders blanks using the mold's flux defaults (plus any `-f` / `--set` overrides) and runs assay on the result. Both sets of findings are reported.

> **Note:** `--assay` is only supported for molds, not ingots.

## CI Integration

Run `ailloy temper` in your CI pipeline to catch issues before packaging or releasing:

```yaml
# GitHub Actions example
- name: Validate and lint mold
  run: ailloy temper --assay ./my-mold
```

A recommended workflow:

```bash
# Validate structure and assay rendered output in one step
ailloy temper --assay ./my-mold

# Package for distribution
ailloy smelt ./my-mold
```

## Template Syntax Validation

Temper parses all `.md` files through Go's `text/template` engine to catch syntax errors. The preprocessor runs first (converting `{{variable}}` to `{{.variable}}`), so template validation matches the actual rendering behavior.

Common template errors caught:

- Unclosed `{{if}}` or `{{range}}` blocks
- Mismatched `{{end}}` tags
- Invalid template function calls
- Malformed template expressions

Note: Temper checks syntax only, not whether variables resolve to values. Use `ailloy temper --assay` or `ailloy forge` with your actual flux values to verify that all variables are populated.

## Flags

| Flag | Description |
|------|-------------|
| `--assay` | Render blanks and run assay on the output |
| `--lint` | Alias for `--assay` |
| `--set key=value` | Set flux values for rendering (can be repeated) |
| `-f, --values file` | Layer flux value files for rendering (can be repeated) |
| `--format format` | Assay output format: `console` (default), `json`, `markdown` |
| `--fail-on level` | Assay exit threshold: `error` (default), `warning`, `suggestion` |
| `--max-lines n` | Override assay line-count threshold (default: 150) |
