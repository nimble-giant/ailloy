# Plugins (`ailloy plugin`)

> **Note:** Plugin generation currently supports Claude Code. The core Ailloy pipeline (`cast`, `forge`, `smelt`) is tool-agnostic and works with any AI coding tool through configurable [output mappings](flux.md#output-mapping).

The `plugin` command generates Claude Code plugins from Ailloy molds. A plugin bundles your mold's commands into a format Claude Code can load directly, including a plugin manifest, documentation, and installation scripts.

## Quick Start

```bash
# Generate a plugin from a mold
ailloy plugin generate --mold ./my-mold

# Update an existing plugin with latest blanks
ailloy plugin update --mold ./my-mold

# Validate plugin structure
ailloy plugin validate
```

## Generating a Plugin

```bash
ailloy plugin generate --mold ./my-mold
```

This creates a plugin directory (default: `./ailloy/`) containing:

- **Plugin manifest** (`plugin.json`) — Metadata and configuration
- **Commands** — All command blanks from the mold
- **README** — Documentation for the plugin
- **Installation scripts** — Scripts to install the plugin locally
- **Hooks and agents** — Configuration files for Claude Code integration

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--mold` | | | Mold directory to generate from (required) |
| `--output` | `-o` | `ailloy` | Output directory for the generated plugin |
| `--watch` | `-w` | `false` | Watch blanks and regenerate on changes |
| `--force` | `-f` | `false` | Overwrite existing plugin without prompting |

If the output directory already exists and `--force` is not set, you will be prompted for confirmation before overwriting.

### Example

```bash
# Generate to a custom directory
ailloy plugin generate --mold ./my-mold --output ./my-plugin

# Force overwrite without prompting
ailloy plugin generate --mold ./my-mold --force
```

## Updating a Plugin

```bash
ailloy plugin update --mold ./my-mold [path]
```

Updates an existing plugin with the latest blanks from your mold while preserving custom additions. The default plugin path is `./ailloy/`.

Before updating, a backup is created automatically (unless `--force` is set). After the update, a summary shows how many files were updated, added, and preserved.

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--mold` | | | Mold directory to update from (required) |
| `--force` | `-f` | `false` | Skip backup before updating |

## Validating a Plugin

```bash
ailloy plugin validate [path]
```

Checks that a plugin has the correct structure and all required files. The default path is `./ailloy/`.

Validation checks:

- **Plugin manifest** — `plugin.json` exists and is valid
- **Commands** — At least one command is present
- **README** — Documentation file exists (warning if missing)

### Example output

```
Plugin structure is valid!

  ✓ Plugin manifest found
  ✓ 9 commands found
  ✓ README documentation present
```

## Testing a Plugin Locally

After generating a plugin, test it with Claude Code:

```bash
# Test with the plugin directory
claude --plugin-dir ./ailloy

# Or run the install script
cd ailloy && ./scripts/install.sh
```
