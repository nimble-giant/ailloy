package mold

import (
	"fmt"
	"io/fs"
	"regexp"
	"strings"
	"text/template"
)

// semverRegex matches semver strings like "1.0.0", "0.2.0-beta.1", etc.
var semverRegex = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)` +
	`(-((0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(\.(0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?` +
	`(\+([0-9a-zA-Z-]+(\.[0-9a-zA-Z-]+)*))?$`)

// versionConstraintRegex matches version constraints like ">=0.2.0", "^1.0.0", "~1.2.0".
var versionConstraintRegex = regexp.MustCompile(`^[>=<^~!]*\d+\.\d+\.\d+`)

// validFluxTypes is the set of allowed types for flux variable declarations.
var validFluxTypes = map[string]bool{
	"string": true,
	"bool":   true,
	"int":    true,
	"list":   true,
}

// ValidationMessage represents a single validation finding with optional location info.
type ValidationMessage struct {
	File    string // file path, empty if not applicable
	Line    int    // line number, 0 if unknown
	Message string
}

// ValidationResult collects errors (blocking) and warnings (informational) from validation checks.
type ValidationResult struct {
	Errors   []ValidationMessage
	Warnings []ValidationMessage
}

// AddError adds a blocking error to the result.
func (r *ValidationResult) AddError(file, msg string) {
	r.Errors = append(r.Errors, ValidationMessage{File: file, Message: msg})
}

// AddWarning adds an informational warning to the result.
func (r *ValidationResult) AddWarning(file, msg string) {
	r.Warnings = append(r.Warnings, ValidationMessage{File: file, Message: msg})
}

// HasErrors returns true if any blocking errors were found.
func (r *ValidationResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// Merge incorporates all errors and warnings from another result.
func (r *ValidationResult) Merge(other *ValidationResult) {
	r.Errors = append(r.Errors, other.Errors...)
	r.Warnings = append(r.Warnings, other.Warnings...)
}

// bareVarPattern matches simple {{variable}} references that lack the Go template
// dot prefix. Mirrors the preprocessor in pkg/config/template.go.
var bareVarPattern = regexp.MustCompile(`\{\{(-?\s*)([a-zA-Z]\w*(?:\.\w+)*)(\s*-?)\}\}`)

// templateKeywords are tokens that must not be dot-prefixed by the preprocessor.
var templateKeywords = map[string]bool{
	"if": true, "else": true, "end": true, "range": true,
	"with": true, "define": true, "block": true, "template": true,
	"nil": true, "true": true, "false": true,
	"not": true, "and": true, "or": true,
	"len": true, "index": true, "print": true, "printf": true,
	"println": true, "call": true,
	"eq": true, "ne": true, "lt": true, "le": true, "gt": true, "ge": true,
	"ingot": true,
}

// preProcessTemplate normalises bare {{variable}} references into {{.variable}}
// so the template parser treats them as data access rather than function calls.
func preProcessTemplate(content string) string {
	return bareVarPattern.ReplaceAllStringFunc(content, func(match string) string {
		sub := bareVarPattern.FindStringSubmatch(match)
		if len(sub) < 4 {
			return match
		}
		prefix, token, suffix := sub[1], sub[2], sub[3]
		firstSegment, _, _ := strings.Cut(token, ".")
		if templateKeywords[firstSegment] {
			return match
		}
		return "{{" + prefix + "." + token + suffix + "}}"
	})
}

// ValidateTemplateSyntax parses template content through Go's text/template
// to catch syntax errors without rendering. Returns validation results with
// any parse errors reported. Bare {{variable}} references are normalised to
// {{.variable}} before parsing, matching the project's template convention.
func ValidateTemplateSyntax(name string, content []byte) *ValidationResult {
	result := &ValidationResult{}
	// Normalise bare {{variable}} to {{.variable}} before parsing.
	normalised := preProcessTemplate(string(content))
	// Use a dummy func map with "ingot" so templates using {{ingot "name"}} don't fail parsing.
	funcMap := template.FuncMap{
		"ingot": func(name string) string { return "" },
	}
	_, err := template.New(name).Funcs(funcMap).Option("missingkey=zero").Parse(normalised)
	if err != nil {
		result.AddError(name, fmt.Sprintf("template syntax error: %v", err))
	}
	return result
}

// TemperMold runs all validation checks on a mold package and returns a unified result.
func TemperMold(m *Mold, fsys fs.FS, basePath string) *ValidationResult {
	result := &ValidationResult{}

	// Manifest structure validation
	if err := ValidateMold(m); err != nil {
		result.AddError("mold.yaml", err.Error())
	}

	// File reference validation
	if err := ValidateMoldFiles(m, fsys, basePath); err != nil {
		result.AddError("mold.yaml", err.Error())
	}

	// Template syntax validation for commands
	for _, cmd := range m.Commands {
		path := basePath + "/claude/commands/" + cmd
		content, err := fs.ReadFile(fsys, path)
		if err != nil {
			continue // already reported by ValidateMoldFiles
		}
		result.Merge(ValidateTemplateSyntax(path, content))
	}

	// Template syntax validation for skills
	for _, skill := range m.Skills {
		path := basePath + "/claude/skills/" + skill
		content, err := fs.ReadFile(fsys, path)
		if err != nil {
			continue // already reported by ValidateMoldFiles
		}
		result.Merge(ValidateTemplateSyntax(path, content))
	}

	// Flux schema consistency (type declarations valid)
	for i, f := range m.Flux {
		if f.Type != "" && !validFluxTypes[f.Type] {
			result.AddWarning("mold.yaml", fmt.Sprintf("flux[%d] %q has unknown type %q", i, f.Name, f.Type))
		}
	}

	return result
}

// TemperIngot runs all validation checks on an ingot package and returns a unified result.
func TemperIngot(i *Ingot, fsys fs.FS, basePath string) *ValidationResult {
	result := &ValidationResult{}

	// Manifest structure validation
	if err := ValidateIngot(i); err != nil {
		result.AddError("ingot.yaml", err.Error())
	}

	// File reference validation
	if err := ValidateIngotFiles(i, fsys, basePath); err != nil {
		result.AddError("ingot.yaml", err.Error())
	}

	return result
}

// ValidateMold validates a Mold manifest for required fields and correct formats.
func ValidateMold(m *Mold) error {
	var errs []string

	if m.APIVersion == "" {
		errs = append(errs, "apiVersion is required")
	}
	if m.Kind == "" {
		errs = append(errs, "kind is required")
	} else if m.Kind != "mold" {
		errs = append(errs, fmt.Sprintf("kind must be \"mold\", got %q", m.Kind))
	}
	if m.Name == "" {
		errs = append(errs, "name is required")
	}
	if m.Version == "" {
		errs = append(errs, "version is required")
	} else if !semverRegex.MatchString(m.Version) {
		errs = append(errs, fmt.Sprintf("version %q is not valid semver", m.Version))
	}

	if m.Requires.Ailloy != "" && !versionConstraintRegex.MatchString(m.Requires.Ailloy) {
		errs = append(errs, fmt.Sprintf("requires.ailloy %q is not a valid version constraint", m.Requires.Ailloy))
	}

	for i, f := range m.Flux {
		if f.Name == "" {
			errs = append(errs, fmt.Sprintf("flux[%d].name is required", i))
		}
		if f.Type == "" {
			errs = append(errs, fmt.Sprintf("flux[%d].type is required", i))
		} else if !validFluxTypes[f.Type] {
			errs = append(errs, fmt.Sprintf("flux[%d].type %q is not valid (allowed: string, bool, int, list)", i, f.Type))
		}
	}

	for i, d := range m.Dependencies {
		if d.Ingot == "" {
			errs = append(errs, fmt.Sprintf("dependencies[%d].ingot is required", i))
		}
		if d.Version == "" {
			errs = append(errs, fmt.Sprintf("dependencies[%d].version is required", i))
		} else if !versionConstraintRegex.MatchString(d.Version) {
			errs = append(errs, fmt.Sprintf("dependencies[%d].version %q is not a valid version constraint", i, d.Version))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("mold validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

// ValidateMoldFiles checks that all files referenced in the mold manifest exist within the given filesystem.
func ValidateMoldFiles(m *Mold, fsys fs.FS, base string) error {
	var missing []string

	for _, cmd := range m.Commands {
		path := base + "/claude/commands/" + cmd
		if _, err := fs.Stat(fsys, path); err != nil {
			missing = append(missing, path)
		}
	}
	for _, skill := range m.Skills {
		path := base + "/claude/skills/" + skill
		if _, err := fs.Stat(fsys, path); err != nil {
			missing = append(missing, path)
		}
	}
	for _, wf := range m.Workflows {
		path := base + "/github/workflows/" + wf
		if _, err := fs.Stat(fsys, path); err != nil {
			missing = append(missing, path)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("mold references missing files:\n  - %s", strings.Join(missing, "\n  - "))
	}
	return nil
}

// ValidateIngot validates an Ingot manifest for required fields and correct formats.
func ValidateIngot(i *Ingot) error {
	var errs []string

	if i.APIVersion == "" {
		errs = append(errs, "apiVersion is required")
	}
	if i.Kind == "" {
		errs = append(errs, "kind is required")
	} else if i.Kind != "ingot" {
		errs = append(errs, fmt.Sprintf("kind must be \"ingot\", got %q", i.Kind))
	}
	if i.Name == "" {
		errs = append(errs, "name is required")
	}
	if i.Version == "" {
		errs = append(errs, "version is required")
	} else if !semverRegex.MatchString(i.Version) {
		errs = append(errs, fmt.Sprintf("version %q is not valid semver", i.Version))
	}

	if i.Requires.Ailloy != "" && !versionConstraintRegex.MatchString(i.Requires.Ailloy) {
		errs = append(errs, fmt.Sprintf("requires.ailloy %q is not a valid version constraint", i.Requires.Ailloy))
	}

	if len(errs) > 0 {
		return fmt.Errorf("ingot validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

// ValidateIngotFiles checks that all files referenced in the ingot manifest exist within the given filesystem.
func ValidateIngotFiles(i *Ingot, fsys fs.FS, base string) error {
	var missing []string

	for _, f := range i.Files {
		path := base + "/" + f
		if _, err := fs.Stat(fsys, path); err != nil {
			missing = append(missing, path)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("ingot references missing files:\n  - %s", strings.Join(missing, "\n  - "))
	}
	return nil
}
