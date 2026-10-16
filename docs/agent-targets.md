# Agent Targets

Cast one mold to every AI agent harness — Claude Code, Codex, OpenCode, Gemini CLI, Cursor, Copilot and 60+ more — from a single set of sources.

A mold names a **surface** (`agents`, `skills`, `commands`, `rules`, `instructions`, `mcp`) instead of a path. At cast time ailloy looks each enabled harness up in the built-in `agent_targets` ore's registry and writes the source to that harness's native location, with per-harness context so one template can emit the right format.

## Quick start

```yaml
# flux.yaml
output:
  agents:
    targets: agents          # agents/ -> every enabled harness's agents dir
  skills:
    targets: skills
  INSTRUCTIONS.md:
    targets: instructions    # -> CLAUDE.md, AGENTS.md, GEMINI.md, ...
```

```bash
ailloy cast ./my-mold --set 'agent.targets=[claude,codex,opencode,cursor]'
```

produces

```
CLAUDE.md                         (claude)
AGENTS.md                         (codex, opencode, cursor: written once)
.claude/agents/reviewer.md
.codex/agents/reviewer.toml       (ext renamed for Codex)
.opencode/agents/reviewer.md
.cursor/agents/reviewer.md
.claude/skills/...  .agents/skills/...  .opencode/skills/...  .cursor/skills/...
```

## Choosing targets

Targets are harness ids from the [registry](#registry). The list is read from, in order:

1. `agent.targets` (the established convention: `--set 'agent.targets=[claude,codex]'`, `-f`, or persisted flux)
2. `ore.agent_targets.targets` (the ore default, `[claude]`)

The registry is built into ailloy, so a mold does **not** need to declare the ore. A mold that wants to pin a version, or a project that wants to add or override harnesses, can declare it as a dependency or set values under `ore.agent_targets.harnesses.<id>` in flux:

```yaml
ore:
  agent_targets:
    harnesses:
      my-agent:
        name: Our in-house agent
        instructions: .my-agent/RULES.md
        agents: .my-agent/agents
```

## Output entry

```yaml
output:
  <source>:
    targets: <surface>       # required
    strategy: append         # optional, overrides the harness's strategy
    process: false           # optional
    set: {key: value}        # optional, merged into every harness's context
```

- One destination is produced per enabled harness that declares the surface. Harnesses that share a destination (`AGENTS.md`, `.agents/skills`) are **written once**.
- A surface with an `ext` renames each file written under a directory (e.g. `.codex/agents/reviewer.toml`, `.github/agents/reviewer.agent.md`).
- If no enabled harness declares the surface, the source is **skipped** (it is not installed at its own path).
- An unknown target id is an error that lists the known ids.

## Template context

Each render pass gets:

| Key | Value |
| --- | --- |
| `agent.current_target` | the harness id (the first, when a destination is shared) |
| `agent.current_targets` | every harness id sharing this destination |
| `agent.format` | the surface's format (default `markdown`), e.g. `codex-toml`, `opencode`, `cursor-mdc` |
| `agent.targets` | every enabled target |

Branch on **`agent.format`**, not the harness id: 65 harnesses collapse to a dozen formats. [Ingots](ingots.md) see the same context, so one shared ingot can hold the per-format frontmatter:

```markdown
<!-- ingots/agent-header.md -->
{{- if eq .agent.format "codex-toml" -}}
name = "reviewer"
description = "Reviews code"
developer_instructions = """
{{- else if eq .agent.format "opencode" -}}
---
description: Reviews code
mode: subagent
---
{{- else -}}
---
name: reviewer
description: Reviews code
---
{{- end }}
```

```markdown
<!-- agents/reviewer.md -->
{{ ingot "agent-header" }}
Review the diff carefully.
{{ if eq .agent.format "codex-toml" }}"""{{ end }}
```

## `--claude-plugin`

Plugin casts pin `agent.targets=[claude]` and `agent.current_target=claude`, so `targets:` entries and blanks gated on those variables render into the plugin.

## Formats

| Format | Shape |
| --- | --- |
| `markdown` | Markdown + YAML frontmatter (`name`, `description`), Claude style |
| `opencode` | Markdown + YAML frontmatter, filename is the name (`description`, `mode`, `model`, `permission`) |
| `codex-toml` | Codex custom agent TOML (`name`, `description`, `developer_instructions`) |
| `vibe-toml` | Mistral Vibe agent TOML |
| `copilot-agent` | GitHub Copilot `.agent.md` (`description` required; `tools`, `model`) |
| `copilot-prompt` | GitHub Copilot `.prompt.md` (`description`, `agent`, `model`, `tools`) |
| `copilot-instructions` | `.instructions.md` with an `applyTo` glob |
| `gemini-toml` | Command TOML (`description`, `prompt`) |
| `kiro-json` | Kiro / Amazon Q custom agent JSON (`name`, `description`, `prompt`, `tools`) |
| `cursor-mdc` | Cursor `.mdc` rule (`description`, `globs`, `alwaysApply`) |
| `windsurf-rule` | Markdown rule with `trigger` / `globs` / `description` |
| `kiro-steering` | Markdown steering with `inclusion` / `fileMatchPattern` |
| `continue-rule` | Markdown rule with `name` / `globs` / `regex` / `alwaysApply` |
| `augment-rule` | Markdown rule with `type` / `description` |
| `paths-rule` | Markdown rule with optional `paths:` globs |
| `mcp-json` | `{"mcpServers": {...}}` |
| `vscode-mcp-json` | `{"servers": {...}}` |
| `opencode-json` | `{"mcp": {...}}` (`opencode.json`, `kilo.jsonc`) |

MCP surfaces use `strategy: merge`, so a mold adds its servers without clobbering the project's own. TOML-configured MCP (Codex, Mistral Vibe, Grok) is not in the registry until TOML merge lands.

## Registry

Project-relative paths researched from vendor docs and source in October 2026. Global (`--global`) casts prefix the same paths with `~`. Harnesses that only read a UI setting for a surface omit it.

| id | Harness | instructions | agents | skills | commands | rules | mcp |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `claude` | Claude Code | `CLAUDE.md` | `.claude/agents` | `.claude/skills` | `.claude/commands` | `.claude/rules` (paths-rule) | `.mcp.json` (mcp-json) |
| `codex` | OpenAI Codex | `AGENTS.md` | `.codex/agents` (.toml, codex-toml) | `.agents/skills` |  |  |  |
| `opencode` | OpenCode | `AGENTS.md` | `.opencode/agents` (opencode) | `.opencode/skills` | `.opencode/commands` (opencode) |  | `opencode.json` (opencode-json) |
| `gemini` | Gemini CLI | `GEMINI.md` | `.gemini/agents` | `.agents/skills` | `.gemini/commands` (.toml, gemini-toml) |  | `.gemini/settings.json` (mcp-json) |
| `antigravity` | Google Antigravity | `AGENTS.md` |  | `.agents/skills` | `.agents/workflows` | `.agents/rules` |  |
| `copilot` | GitHub Copilot (VS Code, CLI, cloud agent) | `.github/copilot-instructions.md` | `.github/agents` (.agent.md, copilot-agent) | `.github/skills` | `.github/prompts` (.prompt.md, copilot-prompt) | `.github/instructions` (.instructions.md, copilot-instructions) | `.vscode/mcp.json` (vscode-mcp-json) |
| `cursor` | Cursor (IDE, CLI, cloud agents) | `AGENTS.md` | `.cursor/agents` | `.cursor/skills` | `.cursor/commands` | `.cursor/rules` (.mdc, cursor-mdc) | `.cursor/mcp.json` (mcp-json) |
| `amp` | Amp | `AGENTS.md` |  | `.agents/skills` |  |  |  |
| `goose` | goose | `AGENTS.md` | `.agents/agents` | `.agents/skills` |  |  |  |
| `crush` | Crush | `AGENTS.md` |  | `.agents/skills` |  |  |  |
| `qwen` | Qwen Code | `QWEN.md` | `.qwen/agents` | `.qwen/skills` | `.qwen/commands` | `.qwen/rules` (paths-rule) | `.qwen/settings.json` (mcp-json) |
| `kimi` | Kimi Code | `AGENTS.md` | `.agents/agents` | `.agents/skills` |  |  | `.kimi-code/mcp.json` (mcp-json) |
| `kiro` | Kiro (IDE and CLI) | `AGENTS.md` | `.kiro/agents` (.json, kiro-json) | `.kiro/skills` |  | `.kiro/steering` (kiro-steering) | `.kiro/settings/mcp.json` (mcp-json) |
| `amazon-q` | Amazon Q Developer | `AmazonQ.md` | `.amazonq/cli-agents` (.json, kiro-json) |  |  | `.amazonq/rules` |  |
| `factory` | Factory Droid | `AGENTS.md` | `.factory/droids` | `.factory/skills` | `.factory/commands` | `.factory/rules` | `.factory/mcp.json` (mcp-json) |
| `warp` | Warp | `AGENTS.md` |  | `.agents/skills` |  |  |  |
| `auggie` | Augment (Auggie CLI and IDE) | `AGENTS.md` | `.augment/agents` | `.augment/skills` | `.augment/commands` | `.augment/rules` (augment-rule) |  |
| `pi` | Pi | `AGENTS.md` |  | `.agents/skills` | `.pi/prompts` |  | `.pi/mcp.json` (mcp-json) |
| `kilo` | Kilo Code | `AGENTS.md` | `.kilo/agents` (opencode) | `.kilo/skills` | `.kilo/commands` | `.kilo/rules` | `kilo.jsonc` (opencode-json) |
| `cline` | Cline | `AGENTS.md` |  | `.cline/skills` | `.clinerules/workflows` | `.clinerules` (paths-rule) |  |
| `rovo` | Atlassian Rovo Dev | `AGENTS.md` | `.rovodev/subagents` | `.agents/skills` |  |  |  |
| `letta` | Letta Code | `AGENTS.md` | `.letta/agents` | `.agents/skills` |  |  |  |
| `continue` | Continue | `AGENTS.md` |  | `.continue/skills` |  | `.continue/rules` (continue-rule) |  |
| `iflow` | iFlow CLI | `IFLOW.md` | `.iflow/agents` |  | `.iflow/commands` |  | `.iflow/settings.json` (mcp-json) |
| `vibe` | Mistral Vibe | `AGENTS.md` | `.vibe/agents` (.toml, vibe-toml) | `.agents/skills` |  |  |  |
| `aider` | Aider | `CONVENTIONS.md` |  |  |  |  |  |
| `gptme` | gptme | `AGENTS.md` |  |  |  |  |  |
| `open-interpreter` | Open Interpreter | `AGENTS.md` |  | `.agents/skills` |  |  |  |
| `grok` | Grok Build | `AGENTS.md` | `.grok/agents` | `.grok/skills` | `.grok/commands` | `.grok/rules` |  |
| `openhands` | OpenHands | `AGENTS.md` | `.openhands/agents` | `.agents/skills` |  |  |  |
| `junie` | JetBrains Junie | `.junie/AGENTS.md` |  | `.junie/skills` |  |  | `.junie/mcp/mcp.json` (mcp-json) |
| `qoder` | Qoder | `AGENTS.md` | `.qoder/agents` | `.qoder/skills` |  | `.qoder/rules` |  |
| `codebuddy` | CodeBuddy Code | `CODEBUDDY.md` |  | `.codebuddy/skills` | `.codebuddy/commands` |  |  |
| `forge` | ForgeCode | `AGENTS.md` | `.forge/agents` | `.forge/skills` |  |  | `.mcp.json` (mcp-json) |
| `reasonix` | DeepSeek Reasonix | `AGENTS.md` |  | `.agents/skills` | `.reasonix/commands` |  |  |
| `deepagents` | LangChain Deep Agents CLI | `.deepagents/AGENTS.md` |  | `.agents/skills` |  |  |  |
| `hermes` | Hermes Agent | `AGENTS.md` |  | `.agents/skills` |  |  |  |
| `windsurf` | Windsurf | `AGENTS.md` |  |  | `.windsurf/workflows` | `.windsurf/rules` (windsurf-rule) |  |
| `zed` | Zed | `AGENTS.md` |  | `.agents/skills` |  |  |  |
| `roo` | Roo Code | `AGENTS.md` |  | `.roo/skills` | `.roo/commands` | `.roo/rules` | `.roo/mcp.json` (mcp-json) |
| `trae` | Trae | `AGENTS.md` |  |  |  | `.trae/rules` |  |
| `tabnine` | Tabnine |  |  |  |  |  | `.tabnine/agent/settings.json` (mcp-json) |
| `firebase-studio` | Firebase Studio | `.idx/airules.md` |  |  |  |  | `.idx/mcp.json` (mcp-json) |
| `android-studio` | Gemini in Android Studio | `AGENT.md` |  |  |  |  |  |
| `void` | Void | `.voidrules` |  |  |  |  |  |
| `lingma` | Tongyi Lingma |  |  | `.lingma/skills` |  | `.lingma/rules` |  |
| `zcode` | Z.ai ZCode | `AGENTS.md` |  | `.zcode/skills` |  |  |  |
| `firebender` | Firebender | `AGENTS.md` |  | `.agents/skills` |  |  |  |
| `avante` | avante.nvim | `avante.md` |  |  |  |  |  |
| `codecompanion` | CodeCompanion.nvim | `AGENTS.md` |  | `.codecompanion/skills` |  |  |  |
| `ona` | Ona | `AGENTS.md` |  | `.ona/skills` |  |  |  |
| `jules` | Google Jules | `AGENTS.md` |  |  |  |  |  |
| `devin` | Devin | `AGENTS.md` |  | `.agents/skills` |  |  |  |
| `cosine` | Cosine Genie | `AGENTS.md` |  |  |  |  |  |
| `codegen` | Codegen | `AGENTS.md` |  |  |  |  |  |
| `tembo` | Tembo | `AGENTS.md` |  |  |  |  |  |
| `sweep` | Sweep | `SWEEP.md` |  |  |  |  |  |
| `replit` | Replit Agent | `replit.md` |  | `.agents/skills` |  |  |  |
| `gitlab-duo` | GitLab Duo Agent Platform | `AGENTS.md` |  | `.agents/skills` |  |  | `.gitlab/duo/mcp.json` (mcp-json) |
| `claude-review` | Claude Code Review | `REVIEW.md` |  |  |  |  |  |
| `cursor-bugbot` | Cursor Bugbot | `.cursor/BUGBOT.md` |  |  |  |  |  |
| `gemini-review` | Gemini Code Assist on GitHub | `.gemini/styleguide.md` |  |  |  |  |  |
| `greptile` | Greptile | `.greptile/rules.md` |  |  |  |  |  |
| `qodo` | Qodo (IDE and Merge) | `AGENTS.md` |  |  |  |  |  |
| `coderabbit` | CodeRabbit | `AGENTS.md` |  |  |  |  |  |

Source: [`ores/agent_targets/flux.yaml`](../ores/agent_targets/flux.yaml).
