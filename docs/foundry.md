# Remote Molds (Foundry)

Ailloy can resolve molds directly from git repositories — no local clone required. The SCM itself acts as the foundry: versions are git tags, and resolved molds are cached locally for fast subsequent access.

## Publishing Your Own Mold

Any git repository with a `mold.yaml` at its root (or at a subpath) is a valid foundry. There is no registry — your SCM is the distribution layer.

### Quick Setup

```bash
# 1. Create a mold (see docs/smelt.md for full authoring guide)
mkdir my-mold && cd my-mold

# 2. Add manifest and flux config
cat > mold.yaml <<'EOF'
apiVersion: v1
kind: mold
name: my-team-mold
version: 1.0.0
description: "My team's AI workflow blanks"
author:
  name: My Team
  url: https://github.com/my-org
requires:
  ailloy: ">=0.2.0"
EOF

cat > flux.yaml <<'EOF'
output:
  commands: .claude/commands
  skills: .claude/skills

project:
  organization: my-org
EOF

# 3. Add your blanks
mkdir -p commands skills
echo "# My Command" > commands/my-command.md

# 4. Push to a git repo and tag a version
git init && git add -A && git commit -m "initial mold"
git remote add origin git@github.com:my-org/my-mold.git
git push -u origin main
git tag v1.0.0 && git push --tags
```

That's it. Anyone can now install your mold:

```bash
ailloy cast github.com/my-org/my-mold@v1.0.0
```

### Requirements

A foundry repository needs:

1. **A `mold.yaml` manifest** at the root (or at a subpath navigated with `//`)
2. **Semver git tags** for version resolution (e.g. `v1.0.0`, `v1.1.0`)
3. **Public or authenticated access** — if users can `git clone` it, `ailloy cast` can resolve it

Optional but recommended:

- **`flux.yaml`** with output mappings and default values
- **`flux.schema.yaml`** for validation and the `anneal` wizard
- **Conventional semver tags** with `v` prefix (e.g. `v1.2.3`)

### Monorepo Layout

If your repository contains multiple molds, use the `//subpath` syntax:

```text
my-org/mold-collection/
├── molds/
│   ├── frontend/
│   │   ├── mold.yaml
│   │   ├── flux.yaml
│   │   └── commands/
│   └── backend/
│       ├── mold.yaml
│       ├── flux.yaml
│       └── commands/
└── README.md
```

```bash
ailloy cast github.com/my-org/mold-collection@v1.0.0//molds/frontend
ailloy cast github.com/my-org/mold-collection@v1.0.0//molds/backend
```

Both subpaths share the same version tag and bare clone cache.

### Versioning Best Practices

- Tag releases with semver: `git tag v1.0.0 && git push --tags`
- Use caret constraints (`@^1.0.0`) for consumers who want compatible updates
- Breaking changes (new required flux vars, renamed output paths) should bump the major version
- The `ailloy.lock` file pins consumers to exact commits — they won't get new versions unless they delete the lock or change the constraint

### Private Repositories

Private molds work out of the box as long as git authentication is configured:

```bash
# SSH (recommended for CI)
ailloy cast github.com/my-org/private-mold@v1.0.0

# Requires: SSH key with repo access, or gh auth login
```

In CI environments, use a deploy key or `GH_TOKEN`:

```yaml
# GitHub Actions example
- run: ailloy cast github.com/my-org/private-mold@v1.0.0
  env:
    GH_TOKEN: ${{ secrets.MOLD_ACCESS_TOKEN }}
```

## Discovering Molds

Ailloy supports two discovery mechanisms that work together:

1. **Foundry indexes** — YAML-based mold catalogs hosted as git repos or static files (SCM-agnostic)
2. **GitHub Topics** — repositories tagged with the `ailloy-mold` topic on GitHub

### Searching

```bash
# Search across all registered indexes and GitHub Topics
ailloy foundry search blueprint

# Verb-noun ordering also works
ailloy search foundry blueprint

# Search only registered foundry indexes (skip GitHub)
ailloy foundry search blueprint --index-only

# Search only GitHub Topics (skip indexes)
ailloy foundry search blueprint --github-only
```

Results from registered indexes appear first, followed by GitHub Topics results. Duplicates (same source) are collapsed, preferring the index entry.

### Making Your Mold Discoverable

To make your mold appear in GitHub Topics search results, add the `ailloy-mold` topic to your GitHub repository:

1. Go to your repository on GitHub
2. Click the gear icon next to "About"
3. Add `ailloy-mold` to the Topics field
4. Save

To list your mold in a foundry index, submit a PR adding an entry to the index's `foundry.yaml` (see [Foundry Index Format](#foundry-index-format) below).

### Managing Foundry Indexes

Register, list, update, and remove foundry indexes:

```bash
# Register a foundry index (fetches and validates the index)
ailloy foundry add https://github.com/nimble-giant/ailloy-foundry-index

# List all registered foundry indexes and their status
ailloy foundry list

# Refresh all cached foundry indexes from their sources
ailloy foundry update

# Cast every mold listed by a foundry (alias: cast-all)
# Skips molds already in the target lockfile unless --force.
# Supports -g/--global, --with-workflows, --dry-run, --force.
ailloy foundry install foundry

# Remove a registered foundry index by name or URL
ailloy foundry remove nimble-foundry

# Verb-noun ordering works for all commands
ailloy add foundry https://github.com/nimble-giant/ailloy-foundry-index
ailloy list foundry
ailloy update foundry
ailloy remove foundry nimble-foundry
```

Registered foundries are stored in `~/.ailloy/config.yaml`. Cached indexes are stored under `~/.ailloy/cache/indexes/`.

The official nimble-giant foundry (`https://github.com/nimble-giant/foundry`) is always present as a built-in default — it appears in `ailloy foundry list` and is searched by `ailloy foundry search` even before you register any other foundries. It's marked `✓ verified` and cannot be removed; running `ailloy foundry add` against its URL upgrades it to a regular registered entry whose update timestamp and status are persisted.

### Interactive TUI

`ailloy foundries` opens a four-tab Bubble Tea terminal UI for discovering,
installing, managing, and auditing foundries and casted molds (ailloys).
On first run it auto-fetches every registered foundry's index so the
verified default is browsable immediately — no `foundry update` required.

```text
Discover   Installed   Foundries   Health

/ filter (name / desc / tag / source)

Recent (last 7 days)
  · plugin-to-mold — Scaffolds a multi-target ailloy mold ...

▶ [x] nimble-mold  ✓  ● installed v0.4.0  — Distributable, reusable, and agnostic AI workflow blanks
  [ ] replicated-launch  ✓  — Full Replicated integration: SDK, releases, CI/CD ...
  [ ] replicated-ce  ✓  — Helm chart reviews, support triage, and knowledge base
  ...

1 selected · 9 shown · 9 total
space toggle · enter cast all · / search · c clear · r refresh · j/k move
```

#### Tabs

| Tab | What it does |
| --- | --- |
| **Discover** | Browse the merged catalog across every effective foundry. Multi-select with `space`, install all selected with `enter`. Live filter with `/`. The "Recent" section highlights molds whose foundry was indexed in the last 7 days (top 10). Already-installed molds show a blue `● installed <version>` badge. |
| **Installed** | List every casted mold across project (`./ailloy.lock`) and global (`~/ailloy.lock`) scope. `u` re-casts to the latest version; `x` uninstalls (uses the install manifest, skips files modified since cast). Pre-manifest legacy entries display a yellow warning. |
| **Foundries** | Add (`a`), remove (`d`), or refresh (`r`) registered foundry indexes. The verified default is rendered as a built-in entry and can't be removed. |
| **Health** | Drift checks (orphaned molds whose foundry no longer indexes them, legacy install manifests) plus `assay` findings against the rendered blank directories from `.ailloy/state.yaml`. `r` re-runs the checks. |

#### Key bindings

| Scope | Keys | Action |
| --- | --- | --- |
| Global | `tab` / `→` / `l` | next tab |
| Global | `shift+tab` / `←` / `h` | previous tab |
| Global | `q` / `ctrl+c` | quit |
| Per tab | `j` / `↓`, `k` / `↑` | move cursor |
| Per tab | `r` | refresh that tab |
| Discover | `/` | focus filter (esc/enter to leave) |
| Discover | `space` | toggle selection on current row |
| Discover | `enter` | cast all selected molds (sequential) |
| Discover | `c` | clear selection |
| Installed | `u` | re-cast to latest (update) |
| Installed | `x` | uninstall current row |
| Foundries | `a` | open add-foundry input |
| Foundries | `d` | remove current foundry |
| Foundries | `i` | install every mold listed by current foundry (skips already-installed) |

The TUI requires a TTY; piping `ailloy foundries` to a file errors out with a
hint to use the scriptable equivalents (`ailloy foundry list/search/...` and
`ailloy uninstall`).

### Uninstalling a casted mold

```bash
# Project scope (./ailloy.lock):
ailloy uninstall github.com/nimble-giant/nimble-mold

# Preview without touching disk:
ailloy uninstall github.com/nimble-giant/nimble-mold --dry-run

# Global scope (~/ailloy.lock):
ailloy uninstall -g github.com/nimble-giant/nimble-mold

# Force-delete files modified since cast:
ailloy uninstall github.com/nimble-giant/nimble-mold --force
```

When a mold is cast, its `LockEntry` records every file written and a
SHA-256 of each file's content at install time. Uninstall walks that
manifest and:

- Deletes files whose on-disk content still matches the recorded hash.
- Skips files whose hash differs (you've edited them) — listed under
  "Skipped (modified)". Pass `--force` to delete anyway.
- Retains files claimed by another casted mold (rare, e.g. shared
  `AGENTS.md`) — listed under "Retained".
- Reports files already missing under "Already absent".
- Prunes any parent directories left empty.
- Removes the entry from the lockfile (or leaves an empty lockfile when
  it was the last entry — never deletes the file outright).

Lockfile entries written before this manifest existed lack the `files:`
list. Uninstall on those errors with a friendly hint to re-cast first
(re-casting backfills the manifest).

## Foundry Index Format

A foundry index is a `foundry.yaml` file that catalogs available molds. It can live at the root of a git repository or be served as a static YAML file.

### Schema

```yaml
apiVersion: v1
kind: foundry-index
name: my-foundry
description: "My collection of molds"
author:
  name: My Team
  url: https://github.com/my-org
molds:
  - name: workflow-mold
    source: github.com/my-org/mold
    description: "AI workflow blanks"
    tags: ["workflows", "claude"]
```

### Fields

| Field | Required | Description |
| ----- | -------- | ----------- |
| `apiVersion` | Yes | Schema version, currently `v1` |
| `kind` | Yes | Must be `foundry-index` |
| `name` | Yes | Unique name for this foundry index |
| `description` | No | Human-readable description |
| `author.name` | No | Author or organization name |
| `author.url` | No | Author URL |
| `molds[].name` | Yes | Mold name |
| `molds[].source` | Yes | Mold source reference (`host/owner/repo`) |
| `molds[].description` | No | Short description for search results |
| `molds[].tags` | No | Searchable tags/categories |

### Hosting a Foundry Index

**As a git repository** (recommended for versioned indexes):

1. Create a repository with a `foundry.yaml` at the root
2. Users register it with: `ailloy foundry add https://github.com/my-org/my-foundry-index`
3. Updates are fetched with: `ailloy foundry update`

**As a static YAML file** (for simple hosting):

1. Host a `foundry.yaml` file at a stable URL
2. Users register it with: `ailloy foundry add https://example.com/foundry.yaml`
3. URLs ending in `.yaml` or `.yml` are automatically detected as static files

### Submitting to the Official Index

To add your mold to the official Ailloy foundry index:

1. Fork the official index repository
2. Add your mold entry to `foundry.yaml`
3. Submit a pull request with a description of your mold

## Downloading Without Installing

Download a mold or ingot to the local cache without installing it into your project. This is useful for inspecting a package before committing to it:

```bash
# Download a mold — validates mold.yaml and prints the cache path
ailloy mold get github.com/nimble-giant/nimble-mold@v0.1.10
ailloy get mold github.com/nimble-giant/nimble-mold@v0.1.10

# Download an ingot
ailloy ingot get github.com/my-org/my-ingot@v1.0.0
ailloy get ingot github.com/my-org/my-ingot@v1.0.0
```

After download, the manifest (`mold.yaml` or `ingot.yaml`) is validated and the local cache path is printed so you can inspect the contents.

## Adding Ingots

Ingots are reusable template components that can be included in molds via the `{{ingot "name"}}` template function. Use `ingot add` to download an ingot and register it in your project:

```bash
# Download and install an ingot into .ailloy/ingots/
ailloy ingot add github.com/my-org/my-ingot@v1.0.0
ailloy add ingot github.com/my-org/my-ingot@v1.0.0
```

This copies the ingot files into `.ailloy/ingots/<name>/` where the template engine can resolve them during `cast` and `forge`.

## Bidirectional Commands

All compound commands support both noun-verb and verb-noun ordering. Both forms invoke the same handler:

| Noun-Verb | Verb-Noun | Description |
| --------- | --------- | ----------- |
| `ailloy foundry search <query>` | `ailloy search foundry <query>` | Search for molds |
| `ailloy foundry add <url>` | `ailloy add foundry <url>` | Register a foundry |
| `ailloy foundry list` | `ailloy list foundry` | List registered foundries |
| `ailloy foundry remove <name\|url>` | `ailloy remove foundry <name\|url>` | Remove a foundry |
| `ailloy foundry update` | `ailloy update foundry` | Refresh foundry indexes |
| `ailloy mold get <ref>` | `ailloy get mold <ref>` | Download a mold |
| `ailloy ingot get <ref>` | `ailloy get ingot <ref>` | Download an ingot |
| `ailloy ingot add <ref>` | `ailloy add ingot <ref>` | Add an ingot |
| `ailloy mold show <name>` | `ailloy show mold <name>` | Show a mold |

## Reference Format

Remote mold references follow this format:

```text
<host>/<owner>/<repo>[@<version>][//<subpath>]
```

### Examples

```bash
# Latest semver tag
ailloy cast github.com/nimble-giant/nimble-mold

# Explicit latest
ailloy cast github.com/nimble-giant/nimble-mold@latest

# Exact version
ailloy cast github.com/nimble-giant/nimble-mold@v0.1.10

# Semver constraint (caret — compatible with 0.1.x)
ailloy cast github.com/nimble-giant/nimble-mold@^0.1.0

# Semver constraint (tilde — patch-level only)
ailloy cast github.com/nimble-giant/nimble-mold@~0.1.0

# Semver range
ailloy cast github.com/nimble-giant/nimble-mold@>=0.1.0

# Branch pin (mutable — prints a warning)
ailloy cast github.com/nimble-giant/nimble-mold@main

# Commit SHA
ailloy cast github.com/nimble-giant/nimble-mold@abc1234

# Subpath navigation (mold lives in a subdirectory of the repo)
ailloy cast github.com/my-org/mono-repo@v1.0.0//molds/claude

# HTTPS and SSH URL forms also work
ailloy cast https://github.com/nimble-giant/nimble-mold@v0.1.10
ailloy cast git@github.com:nimble-giant/nimble-mold@v0.1.10
```

### Version Types

| Type       | Example                             | Behavior                                          |
| ---------- | ----------------------------------- | ------------------------------------------------- |
| Latest     | (no `@`) or `@latest`               | Resolves to the highest semver tag                |
| Exact      | `@v1.2.3` or `@1.2.3`              | Matches the specific tag                          |
| Constraint | `@^1.0.0`, `@~1.2.0`, `@>=1.0.0`  | Highest tag matching the constraint               |
| Branch     | `@main`                             | Resolves to branch HEAD (mutable, prints warning) |
| SHA        | `@abc1234`                          | Pins to a specific commit                         |

## Local vs Remote Detection

Ailloy distinguishes remote references from local paths using a simple heuristic:

- **Remote**: first path segment contains a dot (`github.com/...`), or starts with `https://`, `http://`, `git@`
- **Local**: starts with `.`, `/`, `~`, or has no dot in the first segment

```bash
# These are remote
ailloy cast github.com/nimble-giant/nimble-mold
ailloy cast https://github.com/nimble-giant/nimble-mold

# These are local
ailloy cast ./my-mold
ailloy cast /path/to/mold
ailloy cast my-local-dir
```

## Caching

Resolved molds are cached at `~/.ailloy/cache/` to avoid re-cloning on every invocation.

### Cache Structure

```text
~/.ailloy/cache/
└── github.com/
    └── nimble-giant/
        └── nimble-mold/
            ├── git/          # Bare clone (shared across versions)
            ├── v0.1.10/      # Extracted snapshot
            └── v0.1.9/       # Another version
```

- The `git/` directory is a bare clone used by all versions of that mold.
- Version directories contain extracted file snapshots from `git archive`.
- Deleting the cache triggers a re-clone on next use — it's safe to remove.

### Cache Hit

On subsequent runs, if a version directory already exists and contains a `mold.yaml` (or `ingot.yaml`), the cached snapshot is used without re-extracting. The bare clone is still fetched to pick up new tags.

## Lock File

When a remote mold is resolved, an `ailloy.lock` file is created (or updated) in the current directory. This pins the exact version and commit SHA so that subsequent runs use the same resolution.

### Format

```yaml
apiVersion: v1
molds:
  - name: nimble-mold
    source: github.com/nimble-giant/nimble-mold
    version: v0.1.10
    commit: 2347a626798553252668a15dc98dd020ab9a9c0c
    timestamp: 2026-02-21T19:30:00Z
```

### Lock Behavior

- **Locked + Latest/Constraint**: uses the locked version (run `ailloy recast` to re-resolve)
- **Locked + Exact**: uses the lock if versions match, otherwise re-resolves
- **Branch/SHA pins**: always re-resolve (lock is updated but not consulted)

### Lifecycle Commands

#### Recast (alias: upgrade)

Re-resolve locked dependencies to their latest available versions:

```bash
# Update all dependencies
ailloy recast

# Update a single dependency by name
ailloy recast nimble-mold

# Preview changes without applying
ailloy recast --dry-run
```

Recast fetches the latest semver tags from each dependency's remote, compares with the currently locked version, and updates `ailloy.lock` with the new resolution. A summary of changes is printed showing old and new versions.

#### Quench (alias: lock)

Confirm that all dependencies are pinned to exact versions:

```bash
ailloy quench
```

Quench verifies that every entry in `ailloy.lock` has an exact version and commit SHA pinned. Subsequent `cast` operations will use only these locked versions until the lock is updated via `recast`.

## Authentication

Foundry relies on your existing git credential chain — no custom authentication is needed:

- SSH keys (`~/.ssh/`)
- Git credential helpers
- `gh auth login` (GitHub CLI)
- `~/.netrc`

If you can `git clone` a repository, `ailloy cast` can resolve it.

## Supported Commands

Remote mold references work with all mold-consuming commands:

| Command        | Example                                                              |
| -------------- | -------------------------------------------------------------------- |
| `cast`         | `ailloy cast github.com/nimble-giant/nimble-mold@v0.1.10`           |
| `forge`        | `ailloy forge github.com/nimble-giant/nimble-mold@v0.1.10`          |
| `anneal`       | `ailloy anneal github.com/nimble-giant/nimble-mold@v0.1.10 -o ore.yaml` |
| `mold get`     | `ailloy mold get github.com/nimble-giant/nimble-mold@v0.1.10`       |
| `ingot get`    | `ailloy ingot get github.com/my-org/my-ingot@v1.0.0`               |
| `ingot add`    | `ailloy ingot add github.com/my-org/my-ingot@v1.0.0`               |
| `recast`       | `ailloy recast` or `ailloy recast nimble-mold --dry-run`                  |
| `quench`       | `ailloy quench`                                                           |
| `foundry search` | `ailloy foundry search blueprint`                                 |
| `foundry add`  | `ailloy foundry add https://github.com/nimble-giant/ailloy-foundry-index` |
| `foundry list` | `ailloy foundry list`                                                     |
| `foundry remove` | `ailloy foundry remove nimble-foundry`                              |
| `foundry update` | `ailloy foundry update`                                             |
