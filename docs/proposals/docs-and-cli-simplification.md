# Proposal: Flatten the Learning Curve (Docs + CLI)

**Status:** Proposal — the first slice ships in the same PR as this document.
**Goal:** Keep everything that makes ailloy's docs and CLI good (depth, the
metallurgy brand, Helm-style ergonomics) while making the first hour much
easier.

> This file lives under `docs/proposals/`, which is intentionally **not**
> embedded into the CLI (`clidocs.go` embeds `docs/*.md` and `docs/topics/`).

## Diagnosis

The docs are thorough (~5,000 lines across 18 topics) and the CLI is
polished. The steep curve comes from three things that stack on a new user:

1. **Vocabulary before value.** A newcomer meets mold, blank, flux, ingot,
   ore, foundry, cast, forge, smelt, temper, assay, anneal, quench, recast
   and evolve before they have installed anything. Each term is defined in its
   own page, but there was no single page translating all of them.
2. **No path through the docs.** `docs/README.md` listed pages by category,
   which suits someone who already knows what they need, but not someone
   asking "where do I start?" The authoring tutorial
   (`topics/tutorials/first-mold.md`) wasn't linked from the index at all.
3. **Long first commands and a wide top-level CLI.** The first command
   anyone types was `ailloy cast github.com/nimble-giant/nimble-mold`, and
   `ailloy --help` prints a ~40-line banner followed by 32 top-level
   commands, where noun-first (`mold list`) and verb-first (`list mold`)
   forms sit side by side.

## Shipped in this PR

| Change | Why |
|--------|-----|
| `owner/repo` defaults to `github.com` (`pkg/foundry/reference.go`) | The most common reference gets 11 characters shorter. Explicit hosts (`gitlab.com/…`, self-hosted) keep working unchanged. An existing local directory with the same shape still wins, so no current workflow breaks. |
| [`docs/glossary.md`](../glossary.md) | One page that translates every term into plain English with npm and Helm equivalents, plus a "commands by what you want to do" table. Embedded, so `ailloy docs glossary` works offline. |
| [`docs/README.md`](../README.md) rewritten as a learning path | A "pick your path" table (use / write / share) and four numbered stages. Every page previously linked is still linked; the first-mold tutorial, ore migration, assay, and extensions guides are now linked too. |
| [Mold References](../foundry.md#mold-references) section in `foundry.md` | One place that documents every accepted reference form, including the new shorthand and the local-wins rule. |
| README and Getting Started use the shorthand | The first command a reader copies is the short one. |

## Proposed next steps — Docs

Ranked by impact for effort.

1. **"Ailloy in 5 minutes" landing in the README.** Replace the three
   pipeline tables near the top of `README.md` (Helm table, How It Works,
   Getting Started's pipeline table repeat each other) with one short
   paragraph and a single three-command demo: `cast`, `forge`, `anneal`.
   Move the full pipeline table to the glossary.
2. **Adopt a Diátaxis split without moving files.** Tag each page as
   *tutorial*, *how-to*, *reference*, or *concept* in `clidocs.go`
   summaries and the index. Long pages such as `foundry.md` (754 lines)
   mix all four; split them over time into `foundry.md` (concept + how-to)
   and `topics/reference/references.md` (exhaustive syntax), leaving
   anchors behind so old links keep working.
3. **"Next steps" footer on every page.** Two or three links at the bottom of
   each topic so a reader is never stranded at the end of a page.
4. **A plain-English line under each themed H1.** For example
   `# Temper` followed by *"Validate a mold before you publish it."* The
   brand stays the headline, and the meaning comes one line later.
5. **Copy-paste recipes page.** Common jobs as single blocks: "add ailloy to
   CI", "pin versions for a team", "cast into ~ for every project", "publish
   a monorepo of molds".
6. **Docs tests for links.** Extend `docs/clidocs_test.go` to fail on
   broken relative links and anchors, so restructuring stays safe.

## Proposed next steps — CLI

Each keeps the metallurgy names as the primary commands, since the brand is
the point, and focuses on fewer keystrokes and less to scan.

1. **Group `ailloy --help`.** Use cobra command groups to show
   *Use molds* (cast, forge, anneal, recast, uninstall), *Author molds*
   (mold, temper, assay, smelt), *Registries* (foundry, foundries, ingot,
   ore), and *Maintenance* (quench, cache, evolve, extensions, config, docs).
   Hide the verb-first duplicates (`add`, `list`, `new`, `remove`, `search`,
   `show`, `update`, `get`, `install`, `clear`) from the listing while keeping
   them working. This takes the top-level list from 32 entries to about 18.
2. **Show the banner only where it helps.** Print the ASCII art for bare
   `ailloy` and `--version`, not for `--help`, so help output starts with the
   commands.
3. **Cast by index name.** `ailloy cast nimble-mold` resolves a single word
   through registered foundry indexes when no local directory has that name
   (local still wins, as with the shorthand). The foundry index already
   carries names and sources; this reuses the same "local first, then
   remote" rule as `owner/repo`.
4. **A configurable default host.** `ailloy config set default-host
   gitlab.example.com` (or `AILLOY_DEFAULT_HOST`) for teams whose molds all
   live on one non-GitHub SCM. This only needs to swap the
   `foundry.DefaultHost` constant for a lookup.
5. **Shorter flags for frequent options.** `--head` as an alias for
   `--latest-on-no-tags`, and `-s` for `--set` on `cast`, `forge`, `recast`, and
   `temper`, matching `anneal`, which already accepts `-s`.
6. **Project-file cast.** Bare `ailloy cast` in a project that already has
   an installed manifest re-casts what is recorded there, similar to
   `npm install`. Today bare `cast` is only meaningful inside a smelted
   binary.
7. **Did-you-mean on references.** When `owner/repo` fails to resolve on
   github.com, suggest the full form for other hosts and any close matches
   from registered indexes.

## Overlap with other open work

- **Multi-harness casting (#275)** adds `targets:` output entries. CLI idea 6
  and any future `--as <format>` replacement for `--claude-plugin` should be
  designed alongside it. This PR does not touch output mapping.
- **TOML merge (#274)** touches `strategy: merge` docs in `blanks.md` and
  `features.md`. This PR only adds one bullet under the foundry section of
  `features.md`, so the two should not conflict.
- **Moving bookkeeping out of projects** will change where
  `.ailloy/installed.yaml` and `ailloy.lock` live. The glossary's
  "Files ailloy writes" table and CLI idea 6 should be updated when it lands.
