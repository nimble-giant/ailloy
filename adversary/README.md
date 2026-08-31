# ailloy-guardian

A repo-specific [Adversary Labs](https://adversarylabs.ai/docs) code reviewer for
`github.com/nimble-giant/ailloy`. It runs on pull requests via
`.github/workflows/adversary.yml` and complements the existing CI (build/test,
`conform`, `gofmt`, `golangci-lint`) by checking things a generic linter can't.

## Rules

| Rule | Kind | Fires when |
| --- | --- | --- |
| `ailloy.features-contract` | deterministic | Behavior code (`internal/commands/**`, `pkg/**`, non-test `.go`) changed but `features.md` was not updated in the same change — enforces AGENTS.md's standing rule. |
| `ailloy.docs-drift` | deterministic | Command surface (`internal/commands/**`, `cmd/**`) changed but no `docs/`, `README.md`, or `AGENTS.md` update. |
| `ailloy.mold-correctness` | deterministic | A bundled `mold.yaml`/`ore.yaml` is missing `name`/`version` (or `kind: ore`), or an `ingot.yaml` `files:` entry doesn't exist on disk. |
| `ailloy.go-quality` | heuristic + model | Risky Go patterns on changed files (e.g. `os.Exit` below `main`, `exec.Command`). Uses the model broker for synthesis when a provider is configured; still produces findings without one. |

The first three rules need **no model / no API key**. Only `ailloy.go-quality`
benefits from a configured model provider.

## Develop

Requires Node.js 22+.

```bash
npm ci
npm test        # vitest: unit tests for predicates + rule behavior (no Adversary Labs auth needed)
npm run build   # tsc -> dist/
```

## Run locally

```bash
# from the repo root, against the working tree
adversary run ./adversary --path .
```

Self-authored (unsigned) adversaries need host-execution to be allowed
explicitly (interactive confirmation, or `--allow-unsafe-host-execution` in
non-interactive contexts).

## CI

`.github/workflows/adversary.yml` builds this adversary and runs it on PRs by
explicit reference (`adversaries: ./adversary` + `build: true`). It is
**advisory** by default (`fail-on-findings: false`) — it annotates PRs without
blocking merges. Flip `fail-on-findings: true` to gate.

> Note: `adversaries: auto` cannot be combined with `build`/`builder` — the
> action only accepts those flags with an explicit adversary reference, so a
> local custom adversary must be named rather than auto-discovered.

The `ailloy.go-quality` synthesis step needs a model provider. Add a
`FIREWORKS_API_KEY` repository secret to enable it; without the secret the other
three rules still run.
