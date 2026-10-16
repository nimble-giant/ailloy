# Glossary — Ailloy in Plain English

Ailloy borrows its vocabulary from metalworking. The names are part of the fun, but you only need about six of them to be productive. This page translates each one into plain English and into the tools you may already know.

## The six words you need first

| Ailloy | In plain English | If you know npm | If you know Helm |
|--------|------------------|-----------------|------------------|
| **mold** | A package of AI instruction files you can install | package | chart |
| **blank** | One template file inside a mold (a command, skill, `AGENTS.md`, …) | source file | template |
| **flux** | The settings that fill in a mold's templates | config / env | values |
| **cast** | Install a mold into your project | `npm install` | `helm install` |
| **forge** | Preview what `cast` would write, without writing it | `--dry-run` | `helm template` |
| **foundry** | Where molds live: any git repo, plus optional indexes for discovery | registry | chart repo |

Everything else builds on these. A typical first session uses only `cast`:

```bash
ailloy cast nimble-giant/nimble-mold
```

## Building blocks for mold authors

| Ailloy | In plain English | Learn more |
|--------|------------------|------------|
| **ingot** | A reusable snippet you can include in many blanks with `{{ingot "name"}}` | [Ingots](ingots.md) |
| **ore** | A reusable, opt-in bundle of settings (schema + defaults) a mold can pull in | [Ore](ore.md) |
| **flux schema** | The typed description of a mold's settings, used for validation and the wizard | [Flux](flux.md) |
| **output mapping** | Where each blank lands in the project (`.claude/commands`, `AGENTS.md`, …) | [Blanks](blanks.md) |

## Commands by what you want to do

| I want to… | Command | Plain-English alias |
|------------|---------|---------------------|
| Install a mold | `ailloy cast <mold>` | `ailloy install` |
| Preview before installing | `ailloy forge <mold>` | `ailloy template` |
| Answer a mold's settings interactively | `ailloy anneal <mold>` | `ailloy configure` |
| Update installed molds | `ailloy recast` | `ailloy upgrade` |
| Pin exact versions for CI | `ailloy quench` | `ailloy lock` |
| Remove a mold | `ailloy uninstall <mold>` | — |
| Check a mold I'm writing | `ailloy temper` | `ailloy validate` |
| Lint rendered instructions | `ailloy assay` | `ailloy lint` |
| Package a mold to share | `ailloy smelt` | `ailloy package` |
| Upgrade ailloy itself | `ailloy evolve` | `ailloy reinstall` |
| Browse molds | `ailloy foundries` | — |

Every metallurgy-named command has a plain-English alias, so use whichever reads better in your scripts.

## Naming a mold

A mold reference is `[<host>/]<owner>/<repo>[@<version>][//<subpath>]`. The host defaults to `github.com`; include it for GitLab, Bitbucket, or a self-hosted SCM.

```bash
ailloy cast nimble-giant/nimble-mold              # github.com, latest version
ailloy cast nimble-giant/nimble-mold@^0.1.0       # semver range
ailloy cast gitlab.com/my-team/my-mold@v1.0.0     # another SCM
ailloy cast ./my-mold                             # a local directory
```

See [Mold References](foundry.md#mold-references) for every accepted form.

## Files ailloy writes

| File | What it is |
|------|------------|
| `.ailloy/installed.yaml` | Record of what `cast` wrote, so `recast` and `uninstall` can find it |
| `ailloy.lock` | Optional exact-version pins, created by `quench` |
| `.ailloy/flux/<mold>.yaml` | Saved settings from `anneal` or the foundries TUI |
| `~/.ailloy/cache/` | Downloaded molds and indexes; safe to clear with `ailloy cache clear` |
