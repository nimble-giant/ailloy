# Mold Dependencies

A mold may declare other molds it depends on. When the parent mold is cast,
Ailloy resolves the full dependency graph and installs all transitive
dependencies alongside the parent. Authors can publish small, focused molds
and compose them into larger workflows without forcing consumers to cast each
one manually.

## Declaring a mold dependency

Mold-on-mold dependencies live in the same `dependencies:` list that already
holds ingot/ore dependencies. Use a `mold:` field to identify the dependency:

```yaml
# mold.yaml
apiVersion: v1
kind: mold
name: team-workflow
version: 1.0.0
dependencies:
  - mold: github.com/my-org/issue-helpers
    version: "^1.0.0"
  - mold: github.com/my-org/release-helpers
    version: "^2.0.0"
    as: release
    with:
      release_channel: "stable"
```

A dependency entry must set exactly one of `mold:`, `ingot:`, or `ore:`. The
`mold:` reference accepts the same syntax as top-level `ailloy cast` — a
foundry-resolvable name, a git URL, or a local path.

| Field      | Meaning                                                                |
|------------|------------------------------------------------------------------------|
| `mold:`    | The dependency's source reference (host/owner/repo, git URL, or path). |
| `version:` | A semver constraint (`^1.0.0`, `~1.2`, `>=1.0,<2.0`) or exact tag.     |
| `as:`      | Optional alias used to scope flux overrides (`--set <as>.<key>=…`).    |
| `with:`    | Helm-style sub-flux values seeded into the dep's flux at cast time.    |

## Resolution policy

Ailloy uses **highest-compatible** semver resolution (npm/cargo style):

1. The graph is built by DFS from the parent. Cycles are detected and
   reported with the offending path.
2. Constraints from every parent that references the same `(source, subpath)`
   are intersected. The highest available tag satisfying every constraint
   wins.
3. If no version satisfies every constraint, the cast fails with a list of
   the conflicting constraints.
4. Non-semver pins (branch names, exact SHAs) must agree across every
   reference — any disagreement is reported.

**Example.** With `parent → A@^1.0.0` and `parent → B@^1.0.0`, where both A
and B depend on `D`, A pins `D@^1.0.0` and B pins `D@^1.2.0`, Ailloy will
install the highest published `D` in `>=1.2.0,<2.0.0`. If A pinned `D@^1.0.0`
and B pinned `D@^2.0.0`, the cast fails with a conflict error.

## How dependencies are installed

After the parent mold is cast, transitive dependencies are cast in
leaves-first topological order. Each transitive uses the same destination as
the parent (project root, or `~/` when `--global` is set) and its files are
recorded in `.ailloy/installed.yaml`:

```yaml
molds:
  - name: parent
    source: github.com/my-org/team-workflow
    version: v1.0.0
    installedAs: direct
  - name: issue-helpers
    source: github.com/my-org/issue-helpers
    version: v1.4.2
    installedAs: transitive
    installedBy:
      - github.com/my-org/team-workflow
```

`installedAs` distinguishes user-cast molds (`direct`) from molds Ailloy
pulled in transitively. `installedBy` records the parents that pulled the
mold in — when the last parent goes away, the transitive is
garbage-collected on `ailloy uninstall`.

### Flux propagation (Helm-style)

Each transitive mold renders with its own flux defaults, but parents may
override them via the `with:` block on the dep declaration. From most-default
to highest-precedence:

1. The transitive's own `flux.yaml` defaults / inline `flux:` schema.
2. The parent's `with:` block on the dep entry.
3. Root cast `--set <alias>.<key>=<value>` overrides where `<alias>` is the
   dep's `as:` value (defaults to the dep's mold name).

### `--with-workflows` cascades

When the parent is cast with `--with-workflows`, every transitive in the
graph also contributes its `.github/` files. Without the flag, no `.github/`
files are emitted for parent or transitives.

## Lock & recast

`ailloy.lock` (created by `ailloy quench`) pins every node in the dependency
graph — direct casts and transitives both. `ailloy recast` re-resolves the
full graph: transitives that are no longer required after a constraint
change are pruned; new transitives that newly appear are installed.

> **Note.** As of this release, `quench` and `recast` track the directly-
> cast molds; full graph-aware behavior across constraint changes is
> rolling out in follow-up work — see issue #193.

## Uninstall cascade

`ailloy uninstall <parent>` removes the parent and walks its transitives.
For each transitive whose `installedBy` becomes empty after the parent is
stripped, the transitive's files are removed and the manifest entry is
dropped — mirroring how Ailloy already handles ingots/ores via `Dependents`.

Direct casts (`installedAs: direct`) are never garbage-collected even if a
transitive parent edge happens to be stripped from their `installedBy`.

## Inspecting the graph

Authors can preview the resolved graph before casting:

```bash
ailloy temper           # validates the manifest, including dep declarations
ailloy forge <mold>     # dry-run renders without writing to disk
```

`temper` errors on conflicting `dependencies:` entries (e.g., setting both
`mold:` and `ingot:`). The `forge --show-graph` flag (where available)
prints the resolved tree with the chosen versions.

## See also

- [`docs/foundry.md`](foundry.md) — publishing molds, monorepos, version
  tagging.
- [`docs/ingots.md`](ingots.md) — non-mold (ingot) dependencies.
- [`docs/ore.md`](ore.md) — flux-overlay (ore) dependencies.
