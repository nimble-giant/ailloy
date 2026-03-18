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

	"agent-frontmatter":   "Claude Code uses the name and description fields to discover and invoke agents. Missing or malformed fields prevent the agent from being registered.",
	"command-frontmatter": "Claude Code only recognizes specific frontmatter keys. Unrecognized fields are silently ignored — tool restrictions, model overrides, and hints you set may never take effect.",
	"settings-schema":     "Unrecognized hook event types are silently skipped. A typo in an event name means your hook never fires, with no error to diagnose.",
	"plugin-manifest":     "The plugin manifest is how Claude Code identifies, loads, and displays your plugin. Missing required fields prevent installation or cause silent failures.",
	"plugin-hooks":        "Malformed hook definitions are silently ignored by the runtime. Hooks without name and event fields will never trigger.",
	"description-length":  "AI tools truncate or ignore overly long descriptions. A concise description ensures your command, agent, or plugin is presented correctly in tool UIs and selection menus.",
}

// RuleRationale returns the educational rationale for a rule, or empty string if none is defined.
func RuleRationale(name string) string {
	return ruleRationales[name]
}
