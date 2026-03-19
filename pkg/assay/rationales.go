package assay

// ruleRationales explains *why* each rule exists, shown once per rule group
// in console output to help users understand the reasoning behind a finding.
var ruleRationales = map[string]string{
	"line-count":         "Long instruction files push content out of the active context window and are harder for models to navigate. Splitting into focused topic files improves adherence.",
	"structure":          "Headings let models locate and reference specific instructions precisely. Flat prose is harder to parse and makes selective attention less reliable.",
	"agents-md-presence": "AGENTS.md is the universal entry point read natively by Claude Code, Codex, and other agents. Without it, your AI instructions only work for Claude users.",
	"cross-reference":    "Using @AGENTS.md inside CLAUDE.md ensures Claude imports your shared instructions while keeping them accessible to every other agent.",
	"import-validation":  "A broken @import silently prevents instructions from loading. Models won't receive context they're expected to have, causing subtle, hard-to-debug failures.",
	"empty-file":         "Empty instruction files consume context budget and may mislead models that expect content to be present.",
	"duplicate-topics":   "Duplicated content creates maintenance burden and risks instructions drifting out of sync, leading to conflicting guidance across files.",

	"agent-frontmatter":           "Agent runtimes use the name and description fields to discover and invoke agents. Missing or malformed fields prevent the agent from being registered.",
	"command-frontmatter":         "Agent runtimes only recognize specific frontmatter keys. Unrecognized fields are silently ignored — tool restrictions, model overrides, and hints you set may never take effect.",
	"settings-schema":             "Unrecognized hook event types are silently skipped. A typo in an event name means your hook never fires, with no error to diagnose.",
	"plugin-manifest":             "The plugin manifest is how agent runtimes identify, load, and display your plugin. Missing required fields prevent installation or cause silent failures.",
	"plugin-hooks":                "Malformed hook definitions are silently ignored by the runtime. Hooks without name and event fields will never trigger.",
	"description-length":          "Each skill has exactly one description field that agents use to choose the right skill from potentially 100+ available skills. A concise description ensures reliable selection and correct presentation in tool UIs. See: https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices#writing-effective-descriptions",
	"description-point-of-view":   "Descriptions are injected into the system prompt. Inconsistent point-of-view (first or second person) causes skill discovery problems. Always write in third person. See: https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices#writing-effective-descriptions",
	"description-missing-trigger": "Effective descriptions include both what a skill does and when to use it. Without a trigger clause, the agent may fail to select the skill at the right time. See: https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices#writing-effective-descriptions",
	"name-format":                 "Skill names must be lowercase letters, numbers, and single hyphens only (no leading/trailing/consecutive hyphens), with a maximum of 64 characters. Invalid names prevent registration. See: https://agentskills.io/specification#name-field",
	"name-reserved-words":         "Names containing \"anthropic\" or \"claude\" are reserved and will be rejected by the platform. See: https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices#skill-structure",
	"vague-name":                  "Vague names like \"helper\" or \"utils\" make skills harder to discover and reference. Use specific, descriptive names. See: https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices#naming-conventions",
	"skill-body-length":           "SKILL.md bodies over 500 lines consume excessive context window budget. Split into separate files using progressive disclosure. See: https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices#progressive-disclosure-patterns",
	"commands-deprecated":         "The .claude/commands/ directory is the legacy format. The recommended format is .claude/skills/<name>/SKILL.md, which supports both slash-command invocation (/name) and autonomous invocation by agents. See: https://platform.claude.com/docs/en/agent-sdk/slash-commands#creating-custom-slash-commands",
	"name-directory-mismatch":     "The skill name field must match its parent directory name. A mismatch prevents the skill from being discovered or invoked correctly. See: https://agentskills.io/specification#name-field",
	"description-max-length":      "Descriptions exceeding 1024 characters are rejected by the platform. The skill will fail to register. See: https://agentskills.io/specification#description-field",
	"compatibility-length":        "The compatibility field has a platform limit of 500 characters. Exceeding it causes registration failure. See: https://agentskills.io/specification#compatibility-field",
	"skill-token-budget":          "Skills should keep their SKILL.md body under 5000 tokens to avoid consuming excessive context. Move reference material to separate files. See: https://agentskills.io/specification#progressive-disclosure",
	"description-imperative":      "Descriptions using declarative phrasing ('This skill does...') trigger less reliably than imperative phrasing ('Use this skill when...'). See: https://agentskills.io/skill-creation/optimizing-descriptions#writing-effective-descriptions",
}

// RuleRationale returns the educational rationale for a rule, or empty string if none is defined.
func RuleRationale(name string) string {
	return ruleRationales[name]
}
