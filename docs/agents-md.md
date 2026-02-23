# AGENTS.md Support

[AGENTS.md](https://agents.md) is a standardized, tool-agnostic markdown file for AI coding agent instructions. It is supported by Claude Code, GitHub Copilot, Cursor, Aider, and many other tools.

Ailloy molds can include an `AGENTS.md` template at their root to provide tool-agnostic agent instructions alongside tool-specific blanks.

## Including AGENTS.md in a Mold

Place an `AGENTS.md` file at the root of your mold directory, alongside `mold.yaml`:

```
my-mold/
├── mold.yaml
├── flux.yaml
├── AGENTS.md          # Tool-agnostic agent instructions
├── commands/
│   ├── create-issue.md
│   └── open-pr.md
└── skills/
    └── brainstorm.md
```

The file supports flux template variables like any other blank:

```markdown
# {{project_name}} Agent Instructions

## Build & Test
- Run tests: `make test`
- Lint: `make lint`

## Code Style
- Follow {{org}} conventions
```

## How It Works

When `ailloy cast` runs, root-level files like `AGENTS.md` are automatically discovered and installed to the project root. This works with all output forms:

- **Identity** (`output:` omitted): `AGENTS.md` → `AGENTS.md`
- **String** (`output: .claude`): directories go under `.claude/`, but root files go to the project root
- **Map** (`output: {commands: .claude/commands}`): explicitly map with `AGENTS.md: AGENTS.md`

Metadata files (`mold.yaml`, `flux.yaml`, `README.md`, etc.) are excluded from auto-discovery.

## Claude Code Integration

In Claude Code, `CLAUDE.md` can import `AGENTS.md` using the `@` import syntax:

```markdown
@AGENTS.md

# Claude-Specific Instructions
- Use /commit for commits
- Prefer TypeScript over JavaScript
```

After casting a mold that includes `AGENTS.md`, Ailloy will offer to add the `@AGENTS.md` import to your existing `CLAUDE.md` automatically.

## Reserved Root Files

The following root-level files are treated as mold metadata and are **not** installed:

| File | Purpose |
|------|---------|
| `mold.yaml` | Mold manifest |
| `flux.yaml` | Flux variable defaults |
| `flux.schema.yaml` | Flux validation schema |
| `ingot.yaml` | Ingot manifest |
| `README.md` | Mold documentation |
| `PLUGIN_SUMMARY.md` | Plugin summary |
| `LICENSE` | License file |
