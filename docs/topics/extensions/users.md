# Extensions (User Guide)

Ailloy ships lean. Functionality that doesn't need to be in every CI
job — most notably the rich in-CLI documentation TUI — lives in
**extensions**: separate binaries that ailloy downloads on demand and
execs to provide additional commands.

This page covers how to use them. If you're building one, see the
[developer guide](developer-guide.md).

## Quick reference

```bash
ailloy extensions list             # show available + installed
ailloy extensions install docs     # install an official extension
ailloy extensions update           # update everything
ailloy extensions remove docs      # uninstall
ailloy extensions reset            # wipe and start over
ailloy ext list                    # alias: ext = extensions
```

Bidirectional verbs are also supported:

```bash
ailloy install extension docs
ailloy update extension docs
ailloy remove extension docs
ailloy list extensions
ailloy show extension docs
```

## What's installed where

Extensions live under `~/.ailloy/extensions/<name>/<version>/<binary>`,
tracked by `~/.ailloy/extensions/manifest.yaml`. Each install records:

- The GitHub repo it came from
- The semver tag of the binary
- The SHA-256 hash (verified at install time and on every exec)
- The timestamp of the install
- Whether you've granted consent and whether auto-update is enabled

Inspect the manifest entry for any extension with:

```bash
ailloy extensions show docs
```

## The docs extension

The first official extension is `docs`. It contains the same
documentation that ships embedded in ailloy itself, plus pre-rendered
[glamour](https://github.com/charmbracelet/glamour) output across
multiple widths and themes. Page loads are instant and the bubbletea
TUI gives you a tree-view of topics with a scrollable reading pane.

```bash
ailloy docs                # rich TUI when extension installed
ailloy docs flux           # always renders to stdout (pipe-friendly)
ailloy docs --list         # always prints the topics table
ailloy docs --no-extension # force the in-binary fallback
```

The first time you run `ailloy docs` without the extension installed,
ailloy offers to install it. Decline once and it remembers — re-prompt
with `ailloy ext reset --consent-only`.

## Versioning

Each extension carries its own semver tag. Extensions whose content
mirrors ailloy's docs (currently just `docs`) also record an "ailloy
docs version" — the ailloy release the extension was built from. The
host warns if there's drift:

```
docs extension is based on ailloy v0.42.0; you're on v0.43.0 — some
new commands may not yet be documented
```

A scheduled job in the extension's repo polls ailloy releases and
opens a sync PR when new ailloy versions land, so the extension stays
close to head automatically.

## Updates

By default ailloy auto-updates installed extensions in the background
when you run them, no more often than once every 24 hours. You'll see
a one-line banner the next time you launch the extension after a new
release ships. Disable per-extension with:

```bash
ailloy extensions show docs   # check current state
# (auto-update lives in the manifest; manage with `ailloy ext` or edit
#  ~/.ailloy/extensions/manifest.yaml directly)
```

Force an update check now:

```bash
ailloy extensions update docs
```

## Trust and verification

Every install verifies a SHA-256 checksum from the release's
`checksums.txt` asset before placing the binary on disk. Mismatches
abort the install and leave nothing behind. Future releases will add
sigstore/cosign signature verification for stronger supply-chain
guarantees — track issue #N for status.

## Troubleshooting

| Symptom | Try |
|---|---|
| `ailloy docs` keeps using the fallback after I declined | `ailloy ext reset --consent-only`, then `ailloy docs` |
| The extension is using stale docs | `ailloy ext update docs` |
| Manifest is somehow corrupt | `ailloy ext reset` (full wipe; binaries removed) |
| I'm offline / behind a proxy | `ailloy docs --no-extension` always works on the in-binary fallback |
| I want to uninstall | `ailloy ext remove docs` |

## Installing alongside ailloy

Two ways to get `ailloy + docs extension` in one shot:

```bash
# install.sh
curl -fsSL https://raw.githubusercontent.com/nimble-giant/ailloy/main/install.sh | bash -s -- --with-docs

# Homebrew (sibling formula)
brew install nimble-giant/tap/ailloy-with-docs
```

Both run `ailloy ext install docs` after the main install.
