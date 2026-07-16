# Ailloy — Feature Behavioral Contract

Terse, code-backed reference of ailloy's user-facing behaviors. **Regression-prevention contract** and seed for test coverage: do not change these behaviors without updating this file and its tests.

Every entry states behavior + expectation. Deeper docs: `docs/`. Do not duplicate `docs/` or `README.md` here — this is the contract, not a tutorial.

## Core concepts

| Concept | What it is | Composition |
| --- | --- | --- |
| **mold** | A template package: `mold.yaml` manifest + auto-discovered blank templates + optional `ingots/`, `ores/`, `flux.yaml`/`flux.schema.yaml`, output mappings. | Cast into a target project. May declare mold/ingot/ore dependencies in `mold.yaml`. |
| **ingot** | A reusable template fragment (partial), either a bare `ingots/name.md` or a manifest dir (`ingot.yaml` + `files:`). | Embedded into blanks via the `{{ingot "name"}}` template function; rendered with the same flux context; nested ingot calls allowed; circular refs error. |
| **ore** | A versioned behavior package: flux-schema fragment + defaults + optional `output:` mappings + optional `blanks/`. | Overlays a consuming mold: schema/defaults are namespaced under `ore.<namespace>.*`; gated by `{{if .ore.<ns>.enabled}}` (default `enabled: false`). |
| **blank** | A markdown template file inside a mold, auto-discovered from the mold tree (reserved dirs/files excluded). | Rendered by Go `text/template`; supports flux vars, conditionals, ranges, `{{ingot}}`. |

- Reserved files (never installed as blanks): `mold.yaml`, `flux.yaml`, `flux.schema.yaml`, `ingot.yaml`, `ore.yaml`, `README.md`, `LICENSE`, `.ailloyignore`, etc.
- `.ailloyignore` (or `mold.yaml` `ignore:`) excludes files from `cast`/`forge` (not `smelt`).

## cast (`install`)

Renders a mold's blanks with resolved flux and writes them to destination paths in the target project.

- **Flux precedence** (low→high): `mold.yaml` inline `flux:`/`output:` defaults → `flux.yaml` defaults + ore overlays → persisted `~/.ailloy/flux/<slug>.yaml` then `./.ailloy/flux/<slug>.yaml` → `-f`/`--values` files (layered left→right) → `--set key=value` (highest).
- `--set` uses dotted paths (`project.organization=acme`); YAML-structured values parse; plain scalars stay strings.
- Flux validation runs during cast (required non-empty, type conformance); violations warn, not fatal.
- Declared ore deps are auto-installed to `.ailloy/ores/` before rendering.
- Writes `.ailloy/installed.yaml` (provenance: source, version, commit, file SHA-256s for uninstall drift). Updates `ailloy.lock` only if it already exists.
- `--claude-plugin` packages rendered output as a Claude Code plugin instead of loose files.

### Output mapping (source → destination)

- Forms: string (`output: .claude` — dirs nested under it, root files at project root); map (`{commands: .claude/commands}`); expanded (`{key: {dest, process, set, strategy}}`).
- **One source → many destinations**: a source may list multiple targets, each with its own `dest`, `strategy`, and `set:` render context. Example: `AGENTS.md` written to both `AGENTS.md` and `CLAUDE.md`, each rendered with per-destination `set:` overrides. Resolver emits one file per `(src, dest, set)` tuple; `(dest, set)` tuples are deduped.
- **Strategies** (per target, on existing destination):
  - `replace` (default): whole-file overwrite.
  - `merge`: deep-merge JSON/YAML by extension (maps merge, arrays concat+dedup, ints preserved). Errors on unparseable destination unless `--force-replace-on-parse-error`.
  - `append`: markdown only. Wraps content in an idempotent HTML-comment sentinel keyed by mold name (`<!-- ailloy:mold=<name>:start -->…:end -->`); re-cast replaces that block in place, preserving foreign content and other molds' blocks.
- Ore-supplied `output:` entries merge into the consumer's; consumer key wins on collision; two ores claiming the same key (unresolved by consumer) error. Consumer may pull ore blanks via `from: ore/<namespace>/<path>`.

## flux

- Schema sources (precedence): `flux.schema.yaml` > `mold.yaml` inline `flux:` > `mold.yaml` `output:`.
- `flux.yaml` = defaults + output mapping only (no validation). `flux.schema.yaml` = types + validation, drives the anneal wizard.
- Var fields: `name` (dotted path), `type` (string|bool|int|list|select), `required`, `default`, `options` (for select), `discover` (dynamic population during anneal).
- Ore schema/defaults are authored **unprefixed**; the loader prefixes schema with `ore.<namespace>.` and wraps defaults under `ore.<namespace>:` at merge time. Mold-local values always override installed-ore values on collision.

## anneal (`configure`)

- Interactive wizard that fills flux values per `flux.schema.yaml` and persists them (project/global flux files) for later casts.

## forge (`template`, `blank`)

- Dry-run preview: renders all blanks with resolved flux ("what cast would produce"). Analogous to `helm template`.
- **Ephemeral**: resolves ore deps without writing `.ailloy/ores/`; no `installed.yaml`, no lock update, no provenance recording.
- Default: prints each blank to stdout prefixed `--- <dest> ---` (no trailing ceremony stamp, pipe-safe). `--output <dir>` writes files (respecting strategy). `--debug` prints resolved mapping with origin (mold vs `ore:<ns>`).

## temper (`validate`)

- Auto-detects `mold.yaml` / `ingot.yaml` / `ore.yaml` at root and validates: manifest parse, required fields, semver, `requires.ailloy` constraint, flux types/select options/discover, dependency shape (exactly one of ingot/ore/mold per dep), output dir existence, template syntax, ingot `files:` existence.
- Ore checks: `kind: ore`, snake_case name, unprefixed schema/defaults, `enabled: bool` required. Ephemerally resolves ore deps and reports overlay collisions / shadowed keys / orphan defaults.
- Non-zero exit on errors; exit 0 on warnings-only.
- `--assay` (alias `--lint`): also renders blanks to a temp dir and runs the assay linter on output (molds only). Supports `--set`, `-f`, `--format`, `--fail-on`, `--max-lines`.

## assay (`lint`)

- Lints rendered AI-instruction output against best-practice rules (severity: error/warning/suggestion). Consumed by `temper --assay`.

## smelt (`package`)

Packages a mold for distribution. Two output modes via `-o` (`--output-format`); `--output` sets the target directory.

| Mode | Flag | Artifact | Contents |
| --- | --- | --- | --- |
| Tarball (default) | `-o tar` | `<name>-<version>.tar.gz` | mold.yaml, flux.yaml/schema, output-mapped files, full `ingots/` tree. No transitive deps — offline cast needs a warm cache. |
| Binary | `-o binary` | `<name>-<version>` (executable) | Everything in the tarball **plus** the full transitive dep tree (`deps/{molds,ores,ingots}` + `deps/manifest.json`) embedded via stuffbin. Self-contained: casts offline end-to-end. |

- Stuffbin embeds files under archive paths (`disk-path:/archive-path`); the binary unstuffs its own embedded `fs.FS` (`UnstuffFS`) to cast without network or cache.

### Ingot resolution (disk + embedded)

- **Expectation (regression-critical):** `{{ingot "name"}}` MUST resolve when casting a smelted (`-o binary`) mold fully offline — for **both** the mold's own embedded `ingots/` **and** its embedded dependencies' `ingots/`. Do not break offline embedded-ingot resolution for either case. (A prior regression left the mold's own root-level ingots unresolvable in a smelted binary — pin both cases.)
- For on-disk casts, `{{ingot "name"}}` resolves against disk search paths in order: **mold source root → cwd → `.ailloy/` → `~/.ailloy/`** (`buildIngotResolver`). First match wins; manifest ingots concatenate `files:` in order.
- For a smelted-binary cast, embedded ingots are made resolvable regardless of on-disk presence (see `internal/commands/cast_deps.go` / the ingot resolver). The expectation above is the contract; the mechanism (e.g. staging embedded ingots to disk vs. an `fs.FS`-native resolver) is an implementation detail and may change.
- Offline casts prefer the embedded dep store over the network.

## foundry (dependency resolution & versioning)

- A foundry is an **SCM-native registry**: a git repo of molds/ingots/ores. Versions are git tags; no central index required.
- Version refs: `latest`/none (highest semver, always re-resolves), exact (`@v1.2.3`), constraint (`@^1.0.0`, `@~1.2`, `@>=1.0`), branch (`@main`, mutable — warns), SHA (`@abc1234`).
- Resolution uses `git ls-remote --tags` (no clone to pick a version). Monorepo subpaths prefer `<subpath>-v*` tags, falling back to plain tags.
- **`ailloy.lock`** (opt-in via `quench`): pins each dep to an exact commit SHA. On resolve, a locked non-`latest`/branch/SHA ref that still satisfies its constraint skips remote resolution; `latest` always re-resolves.
- **`.ailloy/installed.yaml`**: always written by cast; records source/version/commit/timestamp/file hashes and `InstalledAs` (direct|transitive) for cascade-uninstall.
- Cache: `~/.ailloy/cache/<host>/<owner>/<repo>/` (shared bare clone + per-version snapshots).

## Other commands (behavior summaries)

- **recast** (`upgrade`): re-resolve installed molds to newer versions and re-render; refreshes `installed.yaml` and (if present) `ailloy.lock`. Layers `--set`/`-f`/`--with-workflows` on top of the original cast's recorded options.
- **quench**: opt into `ailloy.lock` by pinning everything in `installed.yaml`; `--verify` is a CI drift check.
- **evolve** (`reinstall`): self-upgrade the ailloy binary from the latest GitHub release; refuses on Homebrew installs.
- **cache clear**: clear on-disk cache under `~/.ailloy/cache/` (`--molds`, `--indexes`, `--dry-run`, `--yes`).
- **mold new/list/show**: scaffold / list / display molds.
