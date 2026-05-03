# Cast a Mold as a Claude Code Plugin (`cast --claude-plugin`)

`ailloy cast --claude-plugin` packages a mold's flux-rendered output as a Claude Code plugin and writes it to Claude's plugin discovery location. Use this when you want to install a mold as a Claude Code plugin without generating and managing a separate plugin directory yourself.

## Quick Start

```bash
# Project-local: writes to ./.claude/plugins/<slug>/
ailloy cast --claude-plugin

# User-global: writes to ~/.claude/plugins/<slug>/
ailloy cast --claude-plugin --global

# From a remote mold
ailloy cast github.com/nimble-giant/nimble-mold@v0.1.10 --claude-plugin

# Override the plugin name and version
ailloy cast --claude-plugin --plugin-name my-team-plugin --plugin-version 2.0.0

# With flux overrides (same as a normal cast)
ailloy cast --claude-plugin --set greeting=Howdy -f team-values.yaml
```

The plugin directory name is derived from the mold's `name` (slugified to lowercase with non-alphanumeric runs collapsed to dashes). Use `--plugin-name` to override it.

## What gets bundled

`--claude-plugin` runs cast's normal flux/template pipeline and routes the rendered output into a Claude Code plugin layout:

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
| `cast --claude-plugin`                | `./.claude/plugins/<slug>/`     |
| `cast --claude-plugin --global`       | `~/.claude/plugins/<slug>/`     |

Re-running cast against an existing plugin replaces the contents of that single plugin directory. Sibling plugin directories are untouched.

## Flag interactions

- **`--set` / `--values` (`-f`)** — work normally. Flux variables are rendered before packaging.
- **`--with-workflows`** — has no effect with `--claude-plugin`. Workflow blanks are not bundled into Claude Code plugins (a warning is printed if both flags are set).
- **AGENTS.md** — bundled at the plugin root if the mold provides one. The interactive `@AGENTS.md` import prompt for `CLAUDE.md` is skipped (the plugin's AGENTS.md is loaded by Claude when the plugin is active).

## Flags

| Flag                | Default | Description                                                                                          |
| ------------------- | ------- | ---------------------------------------------------------------------------------------------------- |
| `--claude-plugin`       | `false` | Package the rendered mold as a Claude Code plugin instead of installing blanks at their cast destinations |
| `--plugin-name`     | `""`    | Override the plugin name (defaults to the mold's `name`). Requires `--claude-plugin`.                    |
| `--plugin-version`  | `""`    | Override the plugin version (defaults to the mold's `version`, falling back to `0.1.0`). Requires `--claude-plugin`. |
| `--global` / `-g`   | `false` | Write to `~/.claude/plugins/<slug>/` instead of `./.claude/plugins/<slug>/`                          |

## When to use this vs. `ailloy plugin generate`

- **`ailloy cast --claude-plugin`** — for users who want to install a mold as a Claude Code plugin. Goes through the full flux/template pipeline and bundles commands, skills, agents, hooks, AGENTS.md, and README.
- **`ailloy plugin generate`** — author-facing tool that runs a separate transformation step on raw blanks. See [plugin.md](plugin.md).

## Bulk install: every mold in a foundry as a plugin

`ailloy foundry install <name|url> --claude-plugin` installs every mold listed by a foundry as its own Claude Code plugin, including molds pulled in transitively through nested foundries. Each plugin lands at `.claude/plugins/<slug>/` (or `~/.claude/plugins/<slug>/` with `--global`), named after the mold. Pass `--shallow` to skip nested foundries and install only the named foundry's direct molds.

```bash
# Install every mold from the verified default foundry as plugins
ailloy foundry install nimble --claude-plugin

# Or pass a foundry URL
ailloy foundry install github.com/nimble-giant/nimble-foundry --claude-plugin --global
```

Per-plugin overrides (`--plugin-name`, `--plugin-version`) are not supported on the bulk path — each plugin's name and version come from its own `mold.yaml`. The lockfile-based skip-already-installed behavior (`--force`) does not apply to plugin output, since plugin installs do not write to the project lockfile.
