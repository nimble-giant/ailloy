package mold

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strings"
	"text/template"

	"dario.cat/mergo"
)

// templateVarRefPatterns extract data-path references from templates.
// Two patterns are used to avoid false positives on template-local variables ($var.field):
//   - directVarRef: {{.path}} or {{- .path -}} (data access at start of action)
//   - actionVarRef: .path preceded by whitespace within actions (e.g., {{if .path}})
var (
	directVarRefPattern = regexp.MustCompile(`\{\{-?\s*\.(\w[\w.]*?)[\s}-]`)
	actionVarRefPattern = regexp.MustCompile(`\{\{[^}]*?\s\.(\w[\w.]*?)[\s}]`)
)

// bareVarPattern matches simple {{variable}} or {{dotted.path}} references
// that lack the Go template dot prefix. The preprocessor adds the dot automatically
// so template authors can use the simpler {{variable}} syntax.
var bareVarPattern = regexp.MustCompile(`\{\{(-?\s*)([a-zA-Z]\w*(?:\.\w+)*)(\s*-?)\}\}`)

// goTemplateKeywords are tokens that must not be dot-prefixed by the preprocessor.
var goTemplateKeywords = map[string]bool{
	"if": true, "else": true, "end": true, "range": true,
	"with": true, "define": true, "block": true, "template": true,
	"nil": true, "true": true, "false": true,
	"not": true, "and": true, "or": true,
	"len": true, "index": true, "print": true, "printf": true,
	"println": true, "call": true,
	"eq": true, "ne": true, "lt": true, "le": true, "gt": true, "ge": true,
	"ingot": true,
}

// TemplateOption configures optional behaviour for ProcessTemplate.
type TemplateOption func(*templateConfig)

type templateConfig struct {
	ingotResolver *IngotResolver
}

// WithIngotResolver enables the {{ingot "name"}} template function.
func WithIngotResolver(r *IngotResolver) TemplateOption {
	return func(c *templateConfig) {
		c.ingotResolver = r
	}
}

// preProcessTemplate normalises simple {{variable}} references into
// Go template {{.variable}} syntax. This lets template authors use the
// shorter form while keeping full Go template compatibility.
func preProcessTemplate(content string) string {
	return bareVarPattern.ReplaceAllStringFunc(content, func(match string) string {
		sub := bareVarPattern.FindStringSubmatch(match)
		if len(sub) < 4 {
			return match
		}
		prefix, token, suffix := sub[1], sub[2], sub[3]

		firstSegment, _, _ := strings.Cut(token, ".")

		if goTemplateKeywords[firstSegment] {
			return match
		}
		return "{{" + prefix + "." + token + suffix + "}}"
	})
}

// ProcessTemplate renders a template string using Go's text/template engine.
//
// It supports:
//   - Simple variable access: {{default_board}}, {{project_id}}
//   - Dotted path access: {{ore.status.field_id}}
//   - Go template conditionals: {{if .ore.status.enabled}}...{{end}}
//   - Go template ranges: {{range $k, $v := .ore.status.options}}...{{end}}
//   - Nested data access: {{.ore.status.options.ready.id}}
//
// Simple {{variable}} references are automatically normalised to {{.variable}}
// before parsing. Unresolved variables produce logged warnings and resolve to
// empty strings. Returns an error only for template parse/execution failures.
func ProcessTemplate(content string, flux map[string]any, opts ...TemplateOption) (string, error) {
	if content == "" {
		return "", nil
	}

	var cfg templateConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	content = preProcessTemplate(content)

	data := BuildTemplateData(flux)

	funcMap := template.FuncMap{}
	if cfg.ingotResolver != nil {
		funcMap["ingot"] = cfg.ingotResolver.Resolve
	}

	tmpl, err := template.New("").Funcs(funcMap).Option("missingkey=zero").Parse(content)
	if err != nil {
		return "", fmt.Errorf("template parse error: %w", err)
	}

	warnUnresolvedVars(content, data)

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("template execution error: %w", err)
	}

	return buf.String(), nil
}

// BuildTemplateData creates the data map passed to Go's text/template.Execute.
// Flux variables are deep-merged into the data map.
func BuildTemplateData(flux map[string]any) map[string]any {
	data := make(map[string]any)

	if flux != nil {
		_ = mergo.Merge(&data, flux, mergo.WithOverride)
	}

	return data
}

// warnUnresolvedVars scans a template for variable references
// and logs warnings for any that cannot be resolved from the data map.
func warnUnresolvedVars(content string, data map[string]any) {
	seen := make(map[string]bool)

	for _, re := range []*regexp.Regexp{directVarRefPattern, actionVarRefPattern} {
		for _, match := range re.FindAllStringSubmatch(content, -1) {
			if len(match) < 2 {
				continue
			}
			path := match[1]
			if seen[path] {
				continue
			}
			seen[path] = true

			if !resolveDataPath(data, path) {
				log.Printf("warning: unresolved template variable: {{.%s}}", path)
			}
		}
	}
}

// resolveDataPath checks whether a dotted path (e.g., "ore.status.field_id")
// can be resolved against the given data map.
func resolveDataPath(data map[string]any, path string) bool {
	parts := strings.Split(path, ".")
	var current any = data

	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return false
		}
		current, ok = m[part]
		if !ok {
			return false
		}
	}

	return true
}
