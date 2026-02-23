# Validation (`ailloy temper`)

The `temper` command validates and lints mold and ingot packages. It checks manifest fields, file references, template syntax, and flux schema consistency — catching errors before you distribute your package.

Alias: `lint`

## Quick Start

```bash
# Validate a mold
ailloy temper ./my-mold

# Validate an ingot
ailloy temper ./my-ingot

# Using the alias
ailloy lint ./my-mold
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

## CI Integration

Run `ailloy temper` in your CI pipeline to catch issues before packaging or releasing:

```yaml
# GitHub Actions example
- name: Validate mold
  run: ailloy temper ./my-mold
```

A recommended workflow:

```bash
# Validate structure
ailloy temper ./my-mold

# Preview rendered output
ailloy forge ./my-mold

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

Note: Temper checks syntax only, not whether variables resolve to values. Use `ailloy forge` with your actual flux values to verify that all variables are populated.
