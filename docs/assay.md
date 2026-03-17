# Linting AI Instructions (`ailloy assay`)

The `assay` command lints rendered AI instruction files — CLAUDE.md, AGENTS.md, Cursor rules, Copilot instructions, and more — against cross-platform best practices.

Alias: `lint`

> **Note:** To validate mold or ingot package structure, use [`ailloy temper`](temper.md) instead.

## Quick Start

```bash
# Lint the current project
ailloy assay

# Lint a specific directory
ailloy assay ./my-project

# Using the alias
ailloy lint
```

## What Gets Checked

### Content quality rules

| Rule | Severity | Description |
|------|----------|-------------|
| `line-count` | Warning | File exceeds 150 lines (configurable); suggest splitting |
| `structure` | Warning | Markdown file lacks headings — unstructured instructions reduce adherence |
| `agents-md-presence` | Suggestion | Project has platform-specific files but no `AGENTS.md` |
| `cross-reference` | Warning | `CLAUDE.md` exists alongside `AGENTS.md` but doesn't import it via `@AGENTS.md` |
| `import-validation` | Error | `@path/to/file` reference does not resolve to an existing file |
| `empty-file` | Warning | Instruction file exists but has no meaningful content |
| `duplicate-topics` | Warning | Same heading in multiple files with similar content — consider centralizing |

### Schema validation rules (Claude-specific)

| Rule | Severity | Description |
|------|----------|-------------|
| `agent-frontmatter` | Error/Warning | `.claude/agents/*.yml` missing required `name` or recommended `description` |
| `command-frontmatter` | Warning | `.claude/commands/*.md` frontmatter contains unknown fields |
| `settings-schema` | Error | `.claude/settings.json` has invalid JSON or unknown hook event types |

## Platform Detection

Assay auto-detects platforms by file presence:

| Platform | Files |
|----------|-------|
| Claude | `CLAUDE.md`, `.claude/CLAUDE.md`, `.claude/rules/*.md`, `CLAUDE.local.md` |
| Cursor | `.cursor/rules/*.md`, `.cursorrules` |
| Codex | `AGENTS.md`, `codex.md` |
| Copilot | `.github/copilot-instructions.md` |
| Generic | `AGENTS.md` (root and nested directories) |

Use `--platform` to limit linting to a single platform:

```bash
ailloy assay --platform claude
```

## Severity Levels

Assay reports three severity levels:

- **Error** — blocking issues (e.g., broken imports). Causes non-zero exit by default.
- **Warning** — informational issues (e.g., missing structure, long files).
- **Suggestion** — best-practice recommendations (e.g., consider adding `AGENTS.md`).

## Output Formats

```bash
# Styled terminal output (default)
ailloy assay --format console

# Machine-readable JSON
ailloy assay --format json

# Markdown (for CI comments)
ailloy assay --format markdown
```

## CI Integration

Use `--fail-on` to control the exit code threshold:

```bash
# Fail on errors only (default)
ailloy assay --fail-on error

# Also fail on warnings
ailloy assay --fail-on warning

# Fail on any finding
ailloy assay --fail-on suggestion
```

GitHub Actions example:

```yaml
- name: Lint AI instructions
  run: ailloy assay --fail-on warning
```

## Configuration

Assay supports a `.ailloyrc.yaml` config file in the project root. Generate a starter config:

```bash
ailloy assay --init
```

Example `.ailloyrc.yaml`:

```yaml
assay:
  rules:
    line-count:
      enabled: true
      options:
        max-lines: 200       # override default 150
    structure:
      enabled: true
    agents-md-presence:
      enabled: false         # suppress this suggestion
    duplicate-topics:
      enabled: true
  ignore:
    - "vendor/**"
    - ".claude/rules/generated-*.md"
  platforms:
    - claude
    - cursor               # only lint these platforms
```

CLI flags override config values:

```bash
# Override line-count threshold
ailloy assay --max-lines 300
```

## Auto-Fix

The `--fix` flag enables automatic fixes for supported rules. This feature is planned for future releases.

```bash
ailloy assay --fix
```
