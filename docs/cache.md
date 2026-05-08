# Cache Management (`ailloy cache clear`)

Ailloy stores two kinds of artifacts under `~/.ailloy/cache/`:

- **Mold artifacts** — bare git clones and version snapshots of every mold
  fetched by `cast`, `forge`, `mold get`, etc.
- **Foundry indexes** — the `foundry.yaml` files downloaded for every
  registered foundry.

`cache clear` wipes those caches when you need a clean slate — for example,
after a corrupt fetch, when a registry has been republished, or when you
want to reclaim disk space.

> **Note:** `cache clear` only touches the global cache. It does not delete
> project metadata (`.ailloy/installed.yaml`, `ailloy.lock`) or rendered
> output. Use [`uninstall`](foundry.md#uninstalling-a-casted-mold) for
> casted files.

## Quick Start

```bash
# Preview what would be cleared without deleting anything
ailloy cache clear --dry-run

# Clear everything (prompts for confirmation in a TTY)
ailloy cache clear

# Skip the prompt
ailloy cache clear --yes
```

The bidirectional verb form works too:

```bash
ailloy clear cache --dry-run
```

## Preview and Confirmation

Before deleting anything, `cache clear` prints a preview of what's in the
cache and asks for confirmation:

```
Ailloy cache:  ~/.ailloy/cache

  Molds      5 refs, 9 versions  (6.6 MB)
  Indexes    1 indexes              (32.5 KB)

  Total:                            (6.6 MB)

Proceed? [y/N]
```

In a non-interactive shell (CI, piped stdin, etc.), `cache clear` refuses
to run without `--yes`:

```
Error: refusing to clear cache without --yes in non-interactive shell
```

This prevents accidental wipes from stray invocations in scripts.

## Narrowing the Scope

By default, both subtrees are cleared. Use one of the scope flags to clear
only one:

| Flag        | Removes                                    | Preserves                  |
| ----------- | ------------------------------------------ | -------------------------- |
| _(none)_    | Mold artifacts **and** foundry indexes     | Project metadata           |
| `--molds`   | Mold artifacts only                        | `~/.ailloy/cache/indexes/` |
| `--indexes` | Foundry indexes only                       | All mold artifacts         |

`--molds` is useful when you want to force every mold to refetch on the
next `cast` / `recast` but keep your registry data intact. `--indexes` is
useful after a foundry has been republished — clearing the cached index
makes the next `foundry update` fetch fresh metadata.

## Dry Run

`--dry-run` prints the same preview block but does **not** delete anything
and does **not** prompt. Pair it with the scope flags to see exactly what
each invocation would remove:

```bash
ailloy cache clear --dry-run            # preview both
ailloy cache clear --molds --dry-run    # preview molds only
ailloy cache clear --indexes --dry-run  # preview indexes only
```

## Result Summary

After a successful clear, `cache clear` prints what it removed and how much
space it freed:

```
Cleared 5 molds (9 versions), 1 indexes — freed 6.6 MB.
```

If a particular file or directory could not be removed (e.g., due to file
permissions), `cache clear` continues with the rest, prints a `warning:`
line for each failure, and exits non-zero with a final summary:

```
warning: remove /Users/me/.ailloy/cache/github.com/foo/bar: permission denied
Cleared 4 molds (8 versions), 1 indexes — freed 6.4 MB.
Cleared with 1 errors.
```

## CLI Reference

```
ailloy cache clear [flags]
ailloy clear cache [flags]
```

| Flag         | Short | Description                                              |
| ------------ | ----- | -------------------------------------------------------- |
| `--molds`    |       | Clear only the mold artifact cache                       |
| `--indexes`  |       | Clear only the foundry index cache                       |
| `--dry-run`  |       | Preview what would be cleared without deleting           |
| `--yes`      | `-y`  | Skip the confirmation prompt (required in non-TTY)       |

## Common Workflows

```bash
# Force all molds to refetch on the next cast
ailloy cache clear --molds --yes

# Refresh registry metadata after a foundry has been republished
ailloy cache clear --indexes --yes
ailloy foundry update

# See total cache footprint before deciding to clear
ailloy cache clear --dry-run
```
