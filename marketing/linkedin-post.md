# LinkedIn Post

> Audience: software engineers and engineering leaders working on agentic AI and skill/agent authoring.
> Suggested length for LinkedIn: trim to taste — the version below runs long enough to use as a carousel caption or to cut down to ~150 words.

---

Your AI coding setup is software. So why are we still managing it like a folder of dotfiles?

Skills, agents, slash commands, CLAUDE.md, AGENTS.md — these now decide how your AI tools behave. Yet most teams copy-paste them between repos, never version them, and never test them. We did that too. So we spent this spring fixing it.

Since mid-March we've shipped more than two dozen releases of **Ailloy**, the package manager for AI instructions. Three things I'm genuinely excited about:

**1. A linter for AI instructions (`ailloy assay`).**
Think ESLint, but for skills and agents. It checks two dozen+ rules drawn from Anthropic's agent-skill best practices and the open agentskills.io spec — description quality that actually drives skill discovery, naming, progressive disclosure, and a context-budget rule that flags when your instructions are quietly eating 25% of the model's context window before the real work even starts. Drop it in CI with `--fail-on warning`.

**2. A git-native registry (`ailloy foundry`).**
Discover, search, and install molds straight from git — versions are just tags, no registry to run. Verified defaults, nested foundries, and a four-tab terminal UI for browsing, installing, and spotting drift.

**3. Real dependencies.**
Molds can now depend on other molds, with npm/cargo-style semver resolution, conflict detection, and lock files. Publish small, focused, composable instruction packages instead of one giant prompt nobody dares to touch.

It's tool-agnostic — Claude Code, Cursor, Copilot, Windsurf, anything that reads file-based instructions.

If you're authoring skills or agents, I'd love your take. What would *you* want a linter to catch?

Repo + docs in the comments.

#AgenticAI #AIEngineering #DeveloperTools #ClaudeCode #LLM #SoftwareEngineering #DevEx
