# Skills Are Software Now. We Built the Package Manager.

### What shipped in Ailloy this spring — a linter, a registry, and real dependencies for AI instructions

*Draft — for the engineering blog. Audience: software engineers and leaders building with agentic AI and authoring skills/agents.*

---

## The problem nobody named

Sometime in the last year, the files that configure your AI tools quietly became some of the most important code in your repo.

Your `CLAUDE.md` and `AGENTS.md`. Your `.claude/skills/`. Your slash commands and subagents and hooks. These artifacts decide whether an agent reaches for the right tool, follows your conventions, and stays inside your guardrails. They are, functionally, software — they have inputs, behavior, and bugs.

So how are most teams managing them?

Copy-paste between repos. No versions. No tests. No way to know that the skill you pasted last month still matches the spec, or that your instruction file isn't silently consuming a quarter of the model's context window before it reads a single line of your actual task.

We've been guilty of all of it. Ailloy is our answer: **a package manager for AI instructions**, the way Helm is a package manager for Kubernetes. You author reusable, versioned, configurable packages — we call them *molds* — and `cast` them into any project, for any AI tool that reads file-based instructions.

We started Ailloy earlier this year. But the work since mid-March is what turned it from a clever templating tool into something that earns the name "package manager." Across more than two dozen releases (v0.6.8 → v0.6.34), three capabilities stand out.

---

## 1. A linter for AI instructions: `ailloy assay`

If skills are software, they deserve a linter. `ailloy assay` is exactly that — think ESLint or `golangci-lint`, but pointed at your AI instructions instead of your application code.

Point it at a project and it auto-detects what's there — Claude, Cursor, Codex, Copilot, generic `AGENTS.md` — and runs over two dozen rules grouped into content quality and schema validation. The rules aren't invented in a vacuum; they're drawn from [Anthropic's agent-skill best practices](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices) and the open [agentskills.io specification](https://agentskills.io/specification). A few that matter most to anyone authoring skills:

- **Descriptions that actually get your skill discovered.** A skill is only invoked if the model's description-matching picks it. Assay flags descriptions written in the wrong point of view (first/second person instead of third), descriptions that say what a skill *does* but never *when to use it*, declarative phrasing where imperative works better, and descriptions over the platform's hard length limits — the kind that get silently truncated so your skill never fires.
- **Naming and structure.** Reserved words (`claude`, `anthropic`), vague names (`helper`, `utils`) that won't discriminate at selection time, name/directory mismatches that break registration, and the deprecation of `.claude/commands/` in favor of the `SKILL.md` layout.
- **Progressive disclosure and token budgets.** Skill bodies that have outgrown ~5,000 tokens and should push reference material into separate files.

The rule I'd put on a slide, though, is **`context-usage`**. It expands every file *including recursive `@`-imports*, estimates the token cost, and warns when your instructions exceed a percentage of the model's *effective* context window — total window minus system-prompt overhead, auto-detected per platform (≈184K for Claude, ≈113K for Cursor, and so on). Defaults: warn at 10%, error at 25%. It checks both individual files and per-plugin/per-project rollups.

That last one reframes instruction authoring as what it really is — a budget. Every token your `CLAUDE.md` spends is a token the agent can't spend reasoning about the actual task. Now you can see the bill, and fail your build on it:

```bash
ailloy assay --fail-on warning
```

It emits `console`, `json`, or `markdown` (for posting on PRs), supports a `.ailloyrc.yaml` for per-repo rule config, and even has a starter `--fix` for the easy wins. Console output uses clickable file links in modern terminals.

For a team standardizing on skills and agents, this is the difference between "we have a style guide somewhere" and "the style guide is enforced in CI."

---

## 2. A git-native registry and discovery layer: `ailloy foundry`

The second thing a package manager needs is a way to *find* packages. Ailloy's answer is the **foundry** — and the design decision we're happiest with is that there's no server to run.

Your SCM is the registry. Any git repo with a `mold.yaml` is installable; versions are just git tags. `ailloy cast github.com/your-org/your-mold@^1.0.0` resolves the constraint, fetches, caches, and renders — no clone, no publish step, no account.

On top of that primitive, this spring added the discovery layer:

- **Foundry indexes** — SCM-agnostic catalogs that let an org publish a curated set of molds, searchable by `ailloy foundry search` across both registered indexes and GitHub Topics.
- **A verified built-in default** so the official catalog shows up with zero setup.
- **Nested / transitive foundries**, which aggregate molds across linked indexes — so a parent foundry can surface everything its children publish.
- **An interactive four-tab TUI** (`ailloy foundries`): Discover, Installed, Foundries, and Health. Browse the catalog, multi-install with a keystroke, see what's already cast, and surface drift and lint findings — all in the terminal.

The result is that sharing a battle-tested set of agent instructions across a company looks less like emailing a zip file and more like `npm install`.

---

## 3. Real dependencies and reproducibility

The third pillar — the one that separates a package manager from a downloader — is dependency management. A mold can now depend on other molds:

```yaml
# mold.yaml
dependencies:
  - mold: github.com/my-org/issue-helpers
    version: "^1.0.0"
  - mold: github.com/my-org/release-helpers
    version: "^2.0.0"
    as: release
    with:
      release_channel: "stable"
```

Cast the parent and Ailloy resolves the full graph and installs every transitive dependency. Resolution is **highest-compatible semver**, the model npm and cargo use: constraints from every parent referencing the same source are intersected, the highest satisfying tag wins, cycles are detected and reported with the offending path, and genuine conflicts fail the cast with the exact constraints that disagree.

That unlocks the composability we wanted from the start: publish small, focused, single-purpose instruction packages and assemble them, instead of maintaining one monolithic prompt that nobody is brave enough to edit.

Reproducibility got the same treatment. Lock files (`ailloy quench`) pin everything and support a `--verify` drift check for CI. Install manifests record exactly which files each cast wrote (with hashes), which makes `ailloy uninstall` safe — it leaves files you've edited alone, and won't delete files another mold still claims. `ailloy recast` re-renders installed molds in place and remembers the options you cast with, and version constraints (`requires.ailloy`) are now enforced at cast time so a mold can't quietly assume a newer CLI than you're running.

Same mold, same flux values, same output — every time, on every machine.

---

## Everything else that landed since March

The three headliners sit on top of a lot of quieter work that makes the day-to-day feel solid:

- **Tool-agnostic output and Claude Code plugins.** Multi-destination output mapping lets one mold render to several locations with per-destination context, and `cast --claude-plugin` packages a rendered mold as a first-class Claude Code plugin — while the core pipeline stays neutral across Cursor, Copilot, Windsurf, and anything that reads [agents.md](https://agents.md).
- **`ore` — reusable flux partials.** Opt-in, shareable data structures for business logic (GitHub Project status, priority, iteration), so the fiddly bits live in one packaged place instead of being hand-rolled per mold — with a lint rule that catches the shadowing mistakes from migrating off the old in-tree convention.
- **In-CLI documentation.** A navigable terminal docs browser (built on bubbletea, rendered with glamour) with recursive auto-discovery — read the docs without leaving the shell.
- **Self-upgrade.** `ailloy evolve` downloads the latest release, verifies its checksum, and atomically swaps the running binary in place (with a deliberately over-the-top retro-RPG evolution animation, because tools should occasionally be fun).
- **Distribution and hardening.** A Homebrew tap, a one-line install script, plus ongoing Go toolchain upgrades and `govulncheck` fixes to keep the supply chain clean.

---

## Why this matters now

Agentic development is moving from "I have a clever prompt" to "my team has a shared, evolving body of instructions that real work depends on." The moment that body of instructions becomes load-bearing, it needs the things every other kind of load-bearing code already has: versioning, dependencies, linting, reproducible builds, and a way to share without copy-paste.

That's the bet behind Ailloy. Skills and agents are software. This spring we shipped the tooling to treat them that way.

Ailloy is open source (Apache-2.0) and still alpha — the toolchain is solid and we use it in production, but formats may shift before 1.0. If you author skills or agents, we'd love your feedback, especially on the lint rules: what should a linter for AI instructions catch that it doesn't yet?

**Try it:**

```bash
brew install nimble-giant/tap/ailloy
# or
curl -fsSL https://raw.githubusercontent.com/nimble-giant/ailloy/main/install.sh | bash

ailloy cast github.com/nimble-giant/nimble-mold
ailloy assay --fail-on warning
```

Repo: https://github.com/nimble-giant/ailloy
