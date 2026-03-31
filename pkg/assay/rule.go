// Package assay provides linting for AI instruction files (CLAUDE.md, AGENTS.md, etc.)
// against cross-platform best practices.
package assay

import (
	"github.com/nimble-giant/ailloy/pkg/mold"
)

// Platform represents a supported AI coding tool.
type Platform string

const (
	PlatformClaude  Platform = "claude"
	PlatformCursor  Platform = "cursor"
	PlatformCodex   Platform = "codex"
	PlatformCopilot Platform = "copilot"
	PlatformGeneric Platform = "generic"
)

// AllPlatforms returns all supported platforms.
func AllPlatforms() []Platform {
	return []Platform{PlatformClaude, PlatformCursor, PlatformCodex, PlatformCopilot, PlatformGeneric}
}

// DetectedFile represents a discovered AI instruction file.
type DetectedFile struct {
	Path      string
	Platform  Platform
	Content   []byte
	PluginDir string // non-empty if the file belongs to a Claude plugin directory
}

// FileContextStat holds the estimated context window usage for a single instruction file.
type FileContextStat struct {
	File            string // relative file path
	EstimatedTokens int    // estimated token count after expanding all @imports
	ImportCount     int    // number of resolved @imports (transitive)
	PluginDir       string // non-empty if the file belongs to a plugin
}

// RuleContext is passed to each rule, providing the full scan context.
type RuleContext struct {
	RootDir string
	Files   []DetectedFile
	Config  *Config

	// ContextStats is populated by the context-usage rule with per-file token estimates.
	ContextStats  []FileContextStat
	ContextWindow int // context window size in tokens, set by context-usage rule
}

// Rule is the interface every lint rule implements.
type Rule interface {
	// Name returns the rule identifier (e.g. "line-count").
	Name() string
	// DefaultSeverity returns the default severity for this rule's findings.
	DefaultSeverity() mold.DiagSeverity
	// Platforms returns which platforms this rule applies to.
	// An empty slice means all platforms.
	Platforms() []Platform
	// Check runs the rule and returns any diagnostics found.
	Check(ctx *RuleContext) []mold.Diagnostic
}

// Fixer is an optional interface for rules that support auto-fix.
type Fixer interface {
	Rule
	// Fix applies automatic fixes for this rule's findings.
	Fix(ctx *RuleContext) error
}

// registry holds all registered rules.
var registry []Rule

// Register adds a rule to the global registry.
func Register(r Rule) {
	registry = append(registry, r)
}

// AllRules returns all registered rules.
func AllRules() []Rule {
	return registry
}
