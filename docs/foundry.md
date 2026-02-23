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

Search for molds published on GitHub using the `ailloy-mold` topic:

```bash
# Search for molds matching a query
ailloy foundry search blueprint

# Verb-noun ordering also works
ailloy search foundry blueprint
```

Results include the repository name, description, and URL. The search queries the GitHub API for repositories tagged with the `ailloy-mold` topic.

### Making Your Mold Discoverable

To make your mold appear in search results, add the `ailloy-mold` topic to your GitHub repository:

1. Go to your repository on GitHub
2. Click the gear icon next to "About"
3. Add `ailloy-mold` to the Topics field
4. Save

### Managing Foundry Indexes

Register foundry index URLs for use by search and resolution commands:

```bash
# Register a foundry index URL
ailloy foundry add https://github.com/nimble-giant/ailloy-foundry-index

# Verb-noun ordering
ailloy add foundry https://github.com/nimble-giant/ailloy-foundry-index
```

Registered foundries are stored in `~/.ailloy/config.yaml`.

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
