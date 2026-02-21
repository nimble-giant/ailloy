# Skill: Add Ailloy Blank

## What is Ailloy?

Ailloy is a Go CLI tool that manages Claude Code command blanks packaged as molds. Mold directories contain `mold.yaml` metadata, `flux.yaml` default values, and `.claude/commands/*.md` blank files. Blanks are rendered with flux variables via `ailloy forge` or `ailloy cast`, and can also be transformed into Claude Code plugin commands via `ailloy plugin generate`.

An **Ailloy blank** is a Markdown file that defines a Claude Code slash command. Each blank contains purpose, invocation syntax, flags, workflow steps, and rules that instruct Claude how to behave when the command is invoked. Blanks live in a mold directory under `.claude/commands/` and are loaded at runtime via `pkg/blanks.MoldReader`.

## When to Use This Skill

Use this skill whenever the user asks to add, create, or implement a new blank for the Ailloy CLI. This includes requests like:
- "Add a new command blank for X"
- "Create an ailloy blank for Y"
- "Implement a new /Z command"

## Workflow

Follow these steps in order. Each step must be completed before moving to the next.

### Step 1: Design the blank locally

Create the blank as a `.claude/commands/<name>.md` file first. This lets the user test the command interactively in Claude Code before it gets baked into the CLI.

The blank must follow the established structure (see existing blanks for reference):
- `# Title` — H1 heading with the command name
- `## Purpose` — What the command does and why
- `## Command Name` — The slash command name in backticks
- `## Invocation Syntax` — Usage pattern with flags
- `## Workflow` — Numbered steps Claude must follow
- Additional sections as needed (methodology, output format, rules, etc.)

### Step 2: Add to the mold directory

Copy the blank to the mold's commands directory:

```
nimble-mold/.claude/commands/<name>.md
```

This is the source of truth. The `mold.yaml` manifest lists all commands, so also add the filename to the `commands:` section:

```yaml
commands:
  - <name>.md
  # ... existing commands
```

### Step 3: Update documentation

1. **`README.md`** — Add the blank to the "Available Blanks" list under `## Blanks`
2. **`CLAUDE.md`** — Add the blank to the "Available Commands" list

### Step 4: Verify CI requirements

Run all checks that CI and git hooks enforce:

```bash
# Formatting (pre-commit hook)
gofmt -l .

# Vet (pre-commit hook)
go vet ./...

# Build (pre-push hook)
go build ./...

# Tests with race detector (pre-push hook)
go test -race ./...

# Linter (pre-push hook)
golangci-lint run
```

All must pass with zero issues before committing.

### Step 5: Commit with conventional commit format

The project enforces conventional commits via `conform`. Use the `feat` type for new blanks:

```
feat: add <name> blank for <short description>
```

The commit message header must be:
- Lowercase
- Imperative mood
- No trailing period
- Max 89 characters

## Files Modified When Adding a Blank

| File | Change |
|------|--------|
| `.claude/commands/<name>.md` | New file — local command blank |
| `nimble-mold/.claude/commands/<name>.md` | New file — mold source blank |
| `nimble-mold/mold.yaml` | Add to `commands:` list |
| `README.md` | Add to Available Blanks |
| `CLAUDE.md` | Add to Available Commands |

## Key Conventions

- Blanks use `{{variable_name}}` syntax for configurable values (e.g., `{{default_board}}`, `{{organization}}`). Only use variables when the blank needs team-specific configuration.
- The blank file name becomes the slash command name (e.g., `brainstorm.md` becomes `/brainstorm`).
- Blanks are automatically picked up by the plugin generator — no changes needed in `pkg/plugin/`.
- Keep blanks self-contained. Each blank should fully describe what Claude must do without referencing external docs.
- The mold directory (`nimble-mold/.claude/commands/`) and the local commands directory (`.claude/commands/`) should have identical content for blanks that ship with ailloy.
