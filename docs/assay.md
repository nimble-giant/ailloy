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
| `agents-md-presence` | Suggestion | Project has platform-specific files but no `AGENTS.md`; when `CLAUDE.md` is detected the tip gives a concrete migration path using `@AGENTS.md` |
| `cross-reference` | Warning | `CLAUDE.md` exists alongside `AGENTS.md` but doesn't import it via `@AGENTS.md` |
| `import-validation` | Error | `@path/to/file` reference does not resolve to an existing file |
| `empty-file` | Warning | Instruction file exists but has no meaningful content |
| `duplicate-topics` | Warning | Same heading in multiple files with similar content — consider centralizing |

### Schema validation rules (Claude-specific)

| Rule | Severity | Description |
|------|----------|-------------|
| `agent-frontmatter` | Error/Warning | `.claude/agents/*.yml` (or plugin `agents/`) missing required `name` or recommended `description` |
| `command-frontmatter` | Warning | `.claude/commands/*.md` (or plugin `commands/` / `skills/`) frontmatter contains unknown fields; auto-fixable via `ailloy lint --fix` or `ailloy config allow-fields` |
| `settings-schema` | Error | `.claude/settings.json` has invalid JSON or unknown hook event types |
| `plugin-manifest` | Error | `.claude-plugin/plugin.json` is invalid JSON or missing required fields (`name`, `version`, `description`) |
| `plugin-hooks` | Error/Warning | Plugin `hooks/*.json` is invalid JSON, missing the `hooks` array, or contains hook entries without required `name`/`event` fields |
| `description-length` | Warning | Description field exceeds 100 characters (configurable); long descriptions are truncated or ignored by AI tools |

## Claude Plugin Directory Support

Assay automatically detects and lints **Claude plugin directories** — any directory containing a `.claude-plugin/plugin.json` manifest. This includes marketplace repositories that bundle multiple plugins at arbitrary nesting depths.

For each plugin found, assay scans and applies rules to:

| Subdirectory | File type | Rules applied |
|---|---|---|
| `.claude-plugin/plugin.json` | JSON | `plugin-manifest` |
| `commands/**` | `.md` | `command-frontmatter`, `description-length`, content rules |
| `skills/**` | `.md` | `command-frontmatter`, `description-length`, content rules |
| `rules/**` | `.md` | content rules (`structure`, `line-count`, `empty-file`, …) |
| `agents/**` | `.yml` / `.yaml` | `agent-frontmatter`, `description-length` |
| `hooks/**` | `.json` | `plugin-hooks` |

All subdirectories are scanned recursively.

### Example plugin structure

```
my-plugin/
  .claude-plugin/
    plugin.json          ← plugin-manifest rule
  commands/
    create-issue.md      ← command-frontmatter rule
    sub/
      helper.md          ← also scanned recursively
  skills/
    brainstorm.md        ← command-frontmatter rule
  rules/
    style.md             ← content quality rules
  agents/
    reviewer.yml         ← agent-frontmatter rule
  hooks/
    hooks.json           ← plugin-hooks rule
```

### Marketplace support

Assay recursively walks the project tree, so a marketplace directory containing many plugins is fully covered in one pass:

```bash
ailloy assay ./marketplace   # lints all nested plugins
```

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
    description-length:
      enabled: true
      options:
        max-length: 100      # override default 100
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

The `--fix` flag automatically resolves fixable diagnostics after the lint run. Currently supported:

| Rule | What `--fix` does |
|------|-------------------|
| `command-frontmatter` | Adds all detected unknown field names to `extra-allowed-fields` in `.ailloyrc.yaml` |

```bash
# Suppress all unknown-frontmatter warnings in one shot
ailloy lint --fix
```

Re-run `ailloy lint` after fixing to confirm the warnings are gone.

## Managing Allowed Frontmatter Fields

If your commands or skills use custom metadata fields (e.g. `topic`, `source`, `tags`), add them to the allow-list so the `command-frontmatter` rule ignores them:

```bash
# Add fields interactively via CLI
ailloy config allow-fields topic source created updated tags

# Or set them in .ailloyrc.yaml directly
```

```yaml
# .ailloyrc.yaml
assay:
  rules:
    command-frontmatter:
      options:
        extra-allowed-fields: [topic, source, created, updated, tags]
```

`ailloy config allow-fields` merges the new fields into `.ailloyrc.yaml` (creating it with starter defaults if absent), deduplicates, and sorts the list alphabetically.
