// Package agenttargets embeds the agent_targets ore: the harness registry
// that `targets:` output entries expand against. ailloy uses it as the
// built-in default when a mold does not declare the ore itself, so a mold
// can fan out to every harness without a dependency.
package agenttargets

import "embed"

// FS holds the ore's manifest, schema and defaults (the harness registry).
//
//go:embed ore.yaml flux.yaml flux.schema.yaml
var FS embed.FS
