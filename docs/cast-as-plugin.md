# Cast a Mold as a Claude Code Plugin (`cast --as-plugin`)

`ailloy cast --as-plugin` packages a mold's flux-rendered output as a Claude Code plugin and writes it to Claude's plugin discovery location. Use this when you want to install a mold as a Claude Code plugin without generating and managing a separate plugin directory yourself.

## Quick Start

```bash
# Project-local: writes to ./.claude/plugins/<slug>/
ailloy cast --as-plugin

# User-global: writes to ~/.claude/plugins/<slug>/
ailloy cast --as-plugin --global

# From a remote mold
ailloy cast github.com/nimble-giant/nimble-mold@v0.1.10 --as-plugin

# Override the plugin name and version
ailloy cast --as-plugin --plugin-name my-team-plugin --plugin-version 2.0.0

# With flux overrides (same as a normal cast)
ailloy cast --as-plugin --set greeting=Howdy -f team-values.yaml
```

The plugin directory name is derived from the mold's `name` (slugified to lowercase with non-alphanumeric runs collapsed to dashes). Use `--plugin-name` to override it.

## What gets bundled

`--as-plugin` runs cast's normal flux/template pipeline and routes the rendered output into a Claude Code plugin layout:

| Cast destination          | Plugin internal path |
| ------------------------- | -------------------- |
| `.claude/commands/...`    | `commands/...`       |
| `.claude/skills/...`      | `skills/...`         |
| `.claude/agents/...`      | `agents/...`         |
| `.claude/hooks/...`       | `hooks/...`          |
| `AGENTS.md` (root)        | `AGENTS.md`          |
| `README.md` (mold root)   | `README.md`          |
| anything else             | dropped              |

The plugin manifest at `.claude-plugin/plugin.json` is synthesized from `mold.yaml`:

| Manifest field | Source                                                      |
| -------------- | ----------------------------------------------------------- |
| `name`         | `--plugin-name` if set; else `mold.yaml` `name`             |
| `version`      | `--plugin-version` if set; else `mold.yaml` `version`; else `0.1.0` |
| `description`  | `mold.yaml` `description`; omitted if missing               |
| `author.name`  | `mold.yaml` `author.name`; whole `author` field omitted if missing |

## Output location

| Mode                              | Path                            |
| --------------------------------- | ------------------------------- |
| `cast --as-plugin`                | `./.claude/plugins/<slug>/`     |
| `cast --as-plugin --global`       | `~/.claude/plugins/<slug>/`     |

Re-running cast against an existing plugin replaces the contents of that single plugin directory. Sibling plugin directories are untouched.

## Flag interactions

- **`--set` / `--values` (`-f`)** — work normally. Flux variables are rendered before packaging.
- **`--with-workflows`** — has no effect with `--as-plugin`. Workflow blanks are not bundled into Claude Code plugins (a warning is printed if both flags are set).
- **AGENTS.md** — bundled at the plugin root if the mold provides one. The interactive `@AGENTS.md` import prompt for `CLAUDE.md` is skipped (the plugin's AGENTS.md is loaded by Claude when the plugin is active).

## Flags

| Flag                | Default | Description                                                                                          |
| ------------------- | ------- | ---------------------------------------------------------------------------------------------------- |
| `--as-plugin`       | `false` | Package the rendered mold as a Claude Code plugin instead of installing blanks at their cast destinations |
| `--plugin-name`     | `""`    | Override the plugin name (defaults to the mold's `name`). Requires `--as-plugin`.                    |
| `--plugin-version`  | `""`    | Override the plugin version (defaults to the mold's `version`, falling back to `0.1.0`). Requires `--as-plugin`. |
| `--global` / `-g`   | `false` | Write to `~/.claude/plugins/<slug>/` instead of `./.claude/plugins/<slug>/`                          |

## When to use this vs. `ailloy plugin generate`

- **`ailloy cast --as-plugin`** — for users who want to install a mold as a Claude Code plugin. Goes through the full flux/template pipeline and bundles commands, skills, agents, hooks, AGENTS.md, and README.
- **`ailloy plugin generate`** — author-facing tool that runs a separate transformation step on raw blanks. See [plugin.md](plugin.md).
