# Cast a Mold as a Claude Code Plugin

**Date:** 2026-05-01
**Status:** Draft
**Author:** Kris Coleman

## Summary

Add a `--as-plugin` flag to `ailloy cast` that packages a mold's rendered output as a Claude Code plugin and writes it to Claude's plugin discovery location (project-local `.claude/plugins/<slug>/` by default, or `~/.claude/plugins/<slug>/` with `--global`).

This opens a new distribution channel: end users with a mold (local or published) can install it directly as a Claude Code plugin without going through the existing `ailloy plugin generate` author workflow.

## Motivation

Today there are two consumer paths for a mold:

1. **`ailloy cast`** — renders blanks with flux variables and installs them at their declared destinations (`.claude/commands/`, `.claude/skills/`, `.github/workflows/`, etc.). This is the "install into a project" path.
2. **`ailloy plugin generate`** — bundles a mold into Claude Code plugin format (`.claude-plugin/plugin.json`, `commands/`, `hooks/`, `scripts/install.sh`) using a custom `Transformer`. It does **not** render flux templates and does not handle skills, AGENTS.md, or workflows. This is an author-facing tool.

There is no path for a user who wants the simplicity of "install this mold as a Claude Code plugin" with full flux rendering and the same blank coverage `cast` already provides. `--as-plugin` fills that gap.

## Non-Goals

- Deprecating or removing `ailloy plugin generate`. It coexists with `--as-plugin` for now; deprecation is a future decision.
- Plugin marketplace, signing, or registry concerns.
- Re-cast / smart merge semantics. Re-running `cast --as-plugin` overwrites the target plugin directory's contents; that is the refresh story.
- Workflow blanks inside plugins. Claude Code plugins do not host GitHub Actions; `--with-workflows` is silently skipped (with a warning) when combined with `--as-plugin`.

## User-Facing Surface

### Flags on `cast`

| Flag                  | Type    | Default | Description                                                                                          |
| --------------------- | ------- | ------- | ---------------------------------------------------------------------------------------------------- |
| `--as-plugin`         | bool    | false   | Write rendered output as a Claude Code plugin instead of installing blanks at their cast destinations |
| `--plugin-name`       | string  | (mold)  | Override plugin name (default: name from `mold.yaml`). Only valid with `--as-plugin`.                 |
| `--plugin-version`    | string  | (mold)  | Override plugin version (default: version from `mold.yaml`, falling back to `0.1.0`). Only valid with `--as-plugin`. |

Existing flags (`--global`, `--with-workflows`, `--set`, `--values`) interact as follows:

- **`--global`**: routes the plugin to `~/.claude/plugins/<slug>/` instead of project-local `.claude/plugins/<slug>/`.
- **`--with-workflows`**: ignored when combined with `--as-plugin` (warn-and-continue). Workflow blanks have no place in a Claude Code plugin.
- **`--set` / `--values`**: unchanged. Flux rendering happens before packaging.

The CLAUDE.md `@AGENTS.md` import prompt that cast normally offers is skipped when `--as-plugin` is set (AGENTS.md is bundled inside the plugin instead).

### Output Location

| Mode                              | Path                                |
| --------------------------------- | ----------------------------------- |
| `cast --as-plugin`                | `.claude/plugins/<slug>/`           |
| `cast --as-plugin --global`       | `~/.claude/plugins/<slug>/`         |

`<slug>` is the plugin name (mold-derived or override) lowercased with non-alphanumeric runs collapsed to single `-` and trimmed. Empty slug after normalization → error.

If Claude Code's actual plugin discovery uses a different path, the implementation will use the correct path. The principle holds: cast drops the plugin where Claude already looks.

## Architecture

### Layering Decision

Plugin output lives **purely in ailloy**. Molds do not need to be plugin-aware. The `plugin.json` manifest is synthesized from `mold.yaml` (with optional flag overrides). Mold authors get plugin distribution for free.

Code-wise, `cast --as-plugin` reuses `pkg/plugin` for manifest synthesis and final layout, but feeds it cast's *flux-rendered* blanks rather than raw blanks. The existing `Transformer` is bypassed.

### Components

#### New: `pkg/plugin/packager.go`

```go
type Packager struct {
    OutputDir string // resolved plugin dir, e.g. .claude/plugins/<slug>/
}

type RenderedFile struct {
    CastDest string // intended cast destination, e.g. ".claude/commands/foo.md"
    Content  []byte
}

type ManifestInput struct {
    Name        string
    Version     string
    Description string
    Author      mold.Author
}

func (p *Packager) Package(files []RenderedFile, manifest ManifestInput, agentsMd []byte, readme []byte) error
```

Responsibilities:

1. Wipe the contents of `OutputDir` (do not touch siblings).
2. Translate each `RenderedFile.CastDest` into a plugin-internal path via the table below; drop unrecognized paths with a debug log.
3. Detect path collisions (two `CastDest`s mapping to the same plugin internal path) → error naming both sources.
4. Write `.claude-plugin/plugin.json` from `ManifestInput`.
5. Write `AGENTS.md` and `README.md` at the plugin root when provided; otherwise skip.

#### Modified: `internal/commands/cast.go`

When `--as-plugin` is set:

1. Validate flag combinations (see Error Handling).
2. Run the existing flux/render pipeline, but collect rendered files into `[]RenderedFile` keyed by `CastDest` instead of writing them to project paths.
3. Filter out any `.github/workflows/*` entries from the collection regardless of `--with-workflows` (plugins don't host them); if `--with-workflows` was set, emit the one-time warning here.
4. Render the mold's `AGENTS.md` and `README.md` (if present) through flux.
5. Build `ManifestInput` from `mold.yaml`, applying `--plugin-name` / `--plugin-version` overrides if provided.
6. Resolve the target directory (project-local or global), slugify the plugin name.
7. Call `Packager.Package(...)`.
8. Skip the existing direct-write-to-project loop and the CLAUDE.md import prompt.

### Path Translation

| `CastDest` prefix       | Plugin internal path        | Notes                            |
| ----------------------- | --------------------------- | -------------------------------- |
| `.claude/commands/`     | `commands/`                 |                                  |
| `.claude/skills/`       | `skills/`                   |                                  |
| `.claude/agents/`       | `agents/`                   |                                  |
| `.claude/hooks/`        | `hooks/`                    |                                  |
| `.github/workflows/`    | *dropped*                   | Filtered upstream in cast; defense-in-depth here |
| `AGENTS.md` (root)      | `AGENTS.md`                 | Bundled inside plugin            |
| `README.md` (root)      | `README.md`                 | From mold's README, or generated |
| anything else           | preserved relative path     | Forward-compatible default      |

## Data Flow

```
ailloy cast --as-plugin [--global] [--plugin-name X] [--plugin-version Y] [--set ...] [-f ...]
        │
        ▼
┌─────────────────────────────────────────┐
│ cast.go                                 │
│  1. Load mold (existing MoldReader)     │
│  2. Resolve flux (existing layering)    │
│  3. ResolveFiles per output mapping     │
│  4. Render each blank via               │
│     mold.ProcessTemplate                │
│  5. Collect into []RenderedFile         │
│  6. Render AGENTS.md + README.md        │
│  7. Build ManifestInput                 │
│  8. Resolve target dir (slugify name)   │
└─────────────────────────────────────────┘
        │
        ▼
┌─────────────────────────────────────────┐
│ pkg/plugin/Packager.Package(...)        │
│  1. Wipe target dir contents            │
│  2. Translate CastDest → plugin path    │
│  3. Write .claude-plugin/plugin.json    │
│  4. Write commands/, skills/, agents/,  │
│     hooks/ files                        │
│  5. Write AGENTS.md and README.md       │
└─────────────────────────────────────────┘
        │
        ▼
   <target>/<slug>/
     .claude-plugin/plugin.json
     commands/...
     skills/...
     agents/...
     hooks/...
     AGENTS.md
     README.md
```

## Manifest Synthesis Rules

| Field         | Source                                                                       |
| ------------- | ---------------------------------------------------------------------------- |
| `name`        | `--plugin-name` if set; else `mold.yaml` name (required somewhere)            |
| `version`     | `--plugin-version` if set; else `mold.yaml` version; else `0.1.0`             |
| `description` | `mold.yaml` description; else `mold.yaml` summary; else omit                  |
| `author`      | `mold.yaml` author block; omit field entirely if missing (no empty strings)   |

## Error Handling

**Pre-flight (cast.go):**
- `--plugin-name` or `--plugin-version` without `--as-plugin` → flag-validation error.
- No name available (no mold name and no override) → error: `"plugin requires a name; set mold name in mold.yaml or pass --plugin-name"`.
- Slugified name is empty → error: `"plugin name produces empty slug; pass --plugin-name <slug>"`.
- `--with-workflows` + `--as-plugin` → log warning ("workflows are not supported in Claude Code plugins; skipping") and continue.

**Packager:**
- Target dir parent doesn't exist → create with `MkdirAll`.
- Target path exists and is a file → error: `"plugin target <path> exists and is not a directory"`.
- Target dir exists with content → wipe contents (overwrite). Scope strictly to that one directory.
- Unrecognized `CastDest` → debug log, drop. Not an error (forward-compat).
- Path collision → error naming both source paths.

**User-facing output:**
- Success: `"plugin written to <abs-path>"` plus a one-line hint about Claude's plugin discovery.
- Failure: standard error path. Wipe-before-write makes retries safe.

## Testing Strategy

### Unit tests — `pkg/plugin/packager_test.go` (new)

- Each known `CastDest` prefix translates correctly (table-driven).
- Unrecognized `CastDest` is dropped, not errored.
- Path collision returns an error naming both sources.
- `plugin.json` fields populated correctly; missing author/description omitted; missing version defaults to `0.1.0`.
- `AGENTS.md` and `README.md` written at plugin root when provided; skipped when nil/empty.
- Wipe semantics: pre-existing files in target dir are removed; sibling plugin directories untouched.
- Target dir parent created when missing.
- Target path is a file → error.

### Unit tests — `internal/commands/cast_test.go` (extend)

- `--plugin-name` / `--plugin-version` without `--as-plugin` → flag error.
- Slugification (`"My Cool Mold"` → `"my-cool-mold"`; `"!!!"` → empty → error).
- Overrides flow into `ManifestInput`.
- Empty derived name without override → error.

### Integration tests — `internal/commands/cast_plugin_test.go` (new)

- Cast a fixture mold (commands + skill + AGENTS.md) with `--as-plugin` to a temp dir; assert on-disk layout and `plugin.json` contents.
- `--with-workflows --as-plugin`: workflows skipped, warning emitted, no error.
- `--set foo=bar`: flux value rendered in plugin output.
- Re-cast against an existing plugin dir: contents replaced; sibling plugin dir untouched.
- `--global`: routes to a temp `HOME`'s `.claude/plugins/<slug>/`.

### Fixture

Small mold at `testdata/molds/plugin-fixture/` with: 2-3 commands, 1 skill, AGENTS.md, README.md, one template using a flux variable.

## Open Questions

- **Exact Claude Code plugin discovery paths.** This design assumes `.claude/plugins/<slug>/` (project) and `~/.claude/plugins/<slug>/` (global). Implementation must verify these against Claude Code's actual discovery rules; if different, use the correct paths and update this section.

## Future Work

- Once `cast --as-plugin` is solid, evaluate deprecating `ailloy plugin generate`. The Packager could absorb its raw-blank + Transformer flow as an optional pre-step, leaving a single code path for plugin output.
- Mold-side `plugin:` metadata block for plugin-specific overrides (originally option B in the brainstorm; deferred — option A "pure derivation" was chosen for v1).
- Plugin marketplace integration, signing, registry metadata — separate efforts.
