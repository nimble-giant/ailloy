# Skill: Add Ailloy Template

## What is Ailloy?

Ailloy is a Go CLI tool that generates and manages Claude Code command templates. It uses Go's `embed.FS` to bundle Markdown templates at compile time, then copies them into projects during `ailloy init`. Templates are also transformed into Claude Code plugin commands via `ailloy plugin generate`.

An **Ailloy template** is a Markdown file that defines a Claude Code slash command. Each template contains purpose, invocation syntax, flags, workflow steps, and rules that instruct Claude how to behave when the command is invoked. Templates live in `pkg/templates/claude/commands/` and are embedded into the Go binary.

## When to Use This Skill

Use this skill whenever the user asks to add, create, or implement a new template for the Ailloy CLI. This includes requests like:
- "Add a new command template for X"
- "Create an ailloy template for Y"
- "Implement a new /Z command"

## Workflow

Follow these steps in order. Each step must be completed before moving to the next.

### Step 1: Design the template locally

Create the template as a `.claude/commands/<name>.md` file first. This lets the user test the command interactively in Claude Code before it gets baked into the CLI.

The template must follow the established structure (see existing templates for reference):
- `# Title` — H1 heading with the command name
- `## Purpose` — What the command does and why
- `## Command Name` — The slash command name in backticks
- `## Invocation Syntax` — Usage pattern with flags
- `## Workflow` — Numbered steps Claude must follow
- Additional sections as needed (methodology, output format, rules, etc.)

### Step 2: Add to embedded templates

Copy the template to the embedded templates directory:

```
pkg/templates/claude/commands/<name>.md
```

This is the source of truth. The `//go:embed all:claude` directive in `pkg/templates/templates.go` picks it up automatically at compile time.

### Step 3: Register in init.go

Add the template filename to the `templates` slice in `internal/commands/init.go` inside `copyTemplateFiles()`:

```go
templates := []string{
    "<name>.md",       // ADD HERE (alphabetical order preferred)
    "pr-description.md",
    // ... existing templates
}
```

This ensures `ailloy init` copies the new template into projects.

### Step 4: Add icon mapping

Add a case to `getTemplateIcon()` in `internal/commands/template.go`:

```go
case strings.Contains(templateName, "<name>"):
    return "<emoji>"
```

Place the case **before** any broader matches that would catch it first (e.g., a template containing "pr" in its name would match the "pr" case before a more specific one).

### Step 5: Update tests

Update all test files that enumerate expected templates:

1. **`pkg/templates/templates_test.go`** — Add `"<name>.md"` to the `expected` slice in `TestListTemplates_ExpectedTemplates`

2. **`internal/commands/init_integration_test.go`** — Add `"<name>.md"` to the `expectedTemplates` slice in `TestIntegration_CopyTemplateFiles`

3. **`internal/commands/template_test.go`** — Add a test case for the new icon: `{"<name>", "<emoji>"}`

### Step 6: Update documentation

1. **`README.md`** — Add the template to the "Available Templates" list under `## Templates`
2. **`CLAUDE.md`** — Add the template to the "Available Commands" list

### Step 7: Verify CI requirements

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

### Step 8: Commit with conventional commit format

The project enforces conventional commits via `conform`. Use the `feat` type for new templates:

```
feat: add <name> template for <short description>
```

The commit message header must be:
- Lowercase
- Imperative mood
- No trailing period
- Max 89 characters

## Files Modified When Adding a Template

| File | Change |
|------|--------|
| `.claude/commands/<name>.md` | New file — local command template |
| `pkg/templates/claude/commands/<name>.md` | New file — embedded source template |
| `internal/commands/init.go` | Add to `templates` slice |
| `internal/commands/template.go` | Add icon case to `getTemplateIcon()` |
| `pkg/templates/templates_test.go` | Add to expected templates list |
| `internal/commands/init_integration_test.go` | Add to expected templates list |
| `internal/commands/template_test.go` | Add icon test case |
| `README.md` | Add to Available Templates |
| `CLAUDE.md` | Add to Available Commands |

## Key Conventions

- Templates use `{{variable_name}}` syntax for configurable values (e.g., `{{default_board}}`, `{{organization}}`). Only use variables when the template needs team-specific configuration.
- The template file name becomes the slash command name (e.g., `brainstorm.md` becomes `/brainstorm`).
- Templates are automatically picked up by the plugin generator — no changes needed in `pkg/plugin/`.
- Keep templates self-contained. Each template should fully describe what Claude must do without referencing external docs.
- The embedded templates directory (`pkg/templates/claude/commands/`) and the local commands directory (`.claude/commands/`) should have identical content for templates that ship with ailloy.
