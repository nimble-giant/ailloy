# Codex Skills and Agents

Ailloy supports Codex's native repository and user-level instruction surfaces:

| Capability | Project destination | User destination |
|---|---|---|
| Repository instructions | `AGENTS.md` | — |
| Agent Skills | `.agents/skills/<name>/SKILL.md` | `~/.agents/skills/<name>/SKILL.md` |
| Custom agents | `.codex/agents/<name>.toml` | `~/.codex/agents/<name>.toml` |

These layouts follow OpenAI's current [skill](https://learn.chatgpt.com/docs/build-skills) and [custom agent](https://learn.chatgpt.com/docs/agent-configuration/subagents) documentation.

## Package layout

A Codex-focused mold can keep authoring paths descriptive and map them to Codex at cast time:

```text
codex-workflows/
├── mold.yaml
├── flux.yaml
├── AGENTS.md
├── skills/
│   └── reviewing/
│       ├── SKILL.md
│       ├── references/
│       │   └── checklist.md
│       └── scripts/
│           └── collect-diff.sh
└── agents/
    └── reviewer.toml
```

```yaml
# flux.yaml
output:
  skills: .agents/skills
  agents: .codex/agents
  AGENTS.md: AGENTS.md
```

`ailloy cast` renders the files with the mold's flux values and preserves each source directory's relative structure.

## Agent Skills

Each skill is a directory containing a `SKILL.md` file and optional scripts, references, and assets. `SKILL.md` requires YAML frontmatter with `name` and `description`:

```markdown
---
name: reviewing
description: Reviews code changes for correctness, security, and missing tests.
---

# Reviewing

Follow the review workflow in this skill.
```

With the output mapping above, `skills/reviewing/SKILL.md` is installed as `.agents/skills/reviewing/SKILL.md`.

## Custom agents

Each Codex custom agent is a TOML configuration layer. The required fields are `name`, `description`, and `developer_instructions`:

```toml
# agents/reviewer.toml
name = "reviewer"
description = "Reviews changes for correctness, security, and missing tests."
sandbox_mode = "read-only"
developer_instructions = """
Review code like an owner.
Lead with concrete findings and cite files and symbols.
Do not edit application code.
"""
```

Custom agents may also use other supported Codex configuration fields, such as `model`, `model_reasoning_effort`, `mcp_servers`, and `skills.config`.

## Install and validate

Install into the current repository:

```bash
ailloy cast ./codex-workflows
```

Install user-wide skills and agents under the current user's home directory:

```bash
ailloy cast ./codex-workflows --global
```

Lint rendered Codex instructions:

```bash
ailloy assay --platform codex
```

Assay discovers:

- `AGENTS.md` and nested `AGENTS.md` files.
- Agent Skill markdown under repository `.agents/skills/` directories.
- Custom agents under `.codex/agents/*.toml`.

It verifies required skill frontmatter and parses custom-agent TOML, including the three required agent fields.

## Native-layout molds

A mold may instead ship files already arranged at their final Codex paths:

```text
codex-workflows/
├── mold.yaml
├── .agents/
│   └── skills/
│       └── reviewing/
│           └── SKILL.md
└── .codex/
    └── agents/
        └── reviewer.toml
```

When `output:` is omitted, Ailloy preserves `.agents/skills/` and `.codex/agents/` at those exact destinations even though other dot-prefixed directories remain excluded.

## Multi-platform packages

Agent Skills use an open directory format, so one source can target multiple compatible tools:

```yaml
output:
  skills:
    - .agents/skills
    - .claude/skills
```

Provider-native custom-agent formats should use separate outer templates. Shared behavioral instructions can live in an [ingot](ingots.md) and be included by each target's template:

```text
multi-platform-workflows/
├── claude-agents/
│   └── reviewer.md
├── codex-agents/
│   └── reviewer.toml
└── ingots/
    └── reviewer-body.md
```

```yaml
output:
  claude-agents: .claude/agents
  codex-agents: .codex/agents
```
