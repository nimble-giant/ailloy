package config

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strings"
	"text/template"
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
}

// preProcessTemplate normalises simple {{variable}} references into
// Go template {{.variable}} syntax. This lets template authors use the
// shorter form while keeping full Go template compatibility.
//
// Actions that already have a dot prefix, contain Go template directives
// (if, range, etc.), or use dollar-prefixed loop variables ($key) are
// left untouched.
func preProcessTemplate(content string) string {
	return bareVarPattern.ReplaceAllStringFunc(content, func(match string) string {
		sub := bareVarPattern.FindStringSubmatch(match)
		if len(sub) < 4 {
			return match
		}
		prefix, token, suffix := sub[1], sub[2], sub[3]

		// Extract the first path segment to check against keywords
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
//   - Dotted path access: {{models.status.field_id}}
//   - Go template conditionals: {{if .models.status.enabled}}...{{end}}
//   - Go template ranges: {{range $k, $v := .models.status.options}}...{{end}}
//   - Nested model data access: {{.models.status.options.ready.id}}
//
// Simple {{variable}} references are automatically normalised to {{.variable}}
// before parsing. Unresolved variables produce logged warnings and resolve to
// empty strings. Returns an error only for template parse/execution failures.
func ProcessTemplate(content string, variables map[string]string, models *Models) (string, error) {
	if content == "" {
		return "", nil
	}

	// Normalise simple {{variable}} references to {{.variable}}
	content = preProcessTemplate(content)

	// Build the data map available to templates
	data := BuildTemplateData(variables, models)

	// Parse the template
	tmpl, err := template.New("").Option("missingkey=zero").Parse(content)
	if err != nil {
		return "", fmt.Errorf("template parse error: %w", err)
	}

	// Warn about unresolved variable references
	warnUnresolvedVars(content, data)

	// Execute the template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("template execution error: %w", err)
	}

	return buf.String(), nil
}

// BuildTemplateData creates the data map passed to Go's text/template.Execute.
// Flat variables are placed at the top level; model data is nested under "models".
func BuildTemplateData(variables map[string]string, models *Models) map[string]any {
	data := make(map[string]any)

	// Add flat variables at top level
	for k, v := range variables {
		data[k] = v
	}

	// Add models as nested structure (always present so conditionals work)
	if models != nil {
		data["models"] = modelsToTemplateMap(*models)
	} else {
		data["models"] = modelsToTemplateMap(DefaultModels())
	}

	return data
}

// modelsToTemplateMap converts the Models struct into a nested map structure
// suitable for Go template field access (e.g., {{.models.status.enabled}}).
func modelsToTemplateMap(models Models) map[string]any {
	return map[string]any{
		"status":    modelConfigToMap(models.Status),
		"priority":  modelConfigToMap(models.Priority),
		"iteration": modelConfigToMap(models.Iteration),
	}
}

// modelConfigToMap converts a single ModelConfig into a map for template access.
func modelConfigToMap(mc ModelConfig) map[string]any {
	m := map[string]any{
		"enabled":       mc.Enabled,
		"field_mapping": mc.FieldMapping,
		"field_id":      mc.FieldID,
	}

	opts := make(map[string]any)
	for k, v := range mc.Options {
		opts[k] = map[string]any{
			"label": v.Label,
			"id":    v.ID,
		}
	}
	m["options"] = opts

	return m
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

// resolveDataPath checks whether a dotted path (e.g., "models.status.field_id")
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
