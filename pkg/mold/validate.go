package mold

import (
	"fmt"
	"io/fs"
	"path/filepath"
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

// ValidateOutputSources checks that all source directories/files referenced in
// the output mapping actually exist in the mold filesystem.
func ValidateOutputSources(m *Mold, fsys fs.FS) error {
	if m.Output == nil {
		return nil // no output mapping, identity mode — discovery handles existence
	}

	resolved, err := ResolveFiles(m, fsys)
	if err != nil {
		return fmt.Errorf("resolving output mapping: %w", err)
	}

	// If we get here without error, all source dirs/files exist (walk succeeded).
	_ = resolved
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

// DiagSeverity indicates whether a diagnostic is an error (blocking) or warning (informational).
type DiagSeverity int

const (
	// SeverityError is a blocking issue that causes validation to fail.
	SeverityError DiagSeverity = iota
	// SeverityWarning is an informational issue that does not cause failure.
	SeverityWarning
)

// Diagnostic represents a single validation finding with severity and location.
type Diagnostic struct {
	Severity DiagSeverity
	Message  string
	File     string // file path, if applicable
}

// TemperResult holds the outcome of a temper validation run.
type TemperResult struct {
	Diagnostics  []Diagnostic
	ManifestKind string // "mold" or "ingot"
	Name         string
	Version      string
}

// HasErrors returns true if any diagnostic is an error.
func (r *TemperResult) HasErrors() bool {
	for _, d := range r.Diagnostics {
		if d.Severity == SeverityError {
			return true
		}
	}
	return false
}

// Errors returns only error-severity diagnostics.
func (r *TemperResult) Errors() []Diagnostic {
	var errs []Diagnostic
	for _, d := range r.Diagnostics {
		if d.Severity == SeverityError {
			errs = append(errs, d)
		}
	}
	return errs
}

// Warnings returns only warning-severity diagnostics.
func (r *TemperResult) Warnings() []Diagnostic {
	var warnings []Diagnostic
	for _, d := range r.Diagnostics {
		if d.Severity == SeverityWarning {
			warnings = append(warnings, d)
		}
	}
	return warnings
}

// Temper validates a mold or ingot at the given filesystem root.
// It checks manifest presence, manifest fields, file references,
// template syntax, and flux schema consistency.
func Temper(fsys fs.FS) *TemperResult {
	result := &TemperResult{}

	// Detect manifest type
	hasMold := fileExists(fsys, "mold.yaml")
	hasIngot := fileExists(fsys, "ingot.yaml")

	if !hasMold && !hasIngot {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Severity: SeverityError,
			Message:  "no mold.yaml or ingot.yaml found",
		})
		return result
	}

	if hasMold {
		result.ManifestKind = "mold"
		temperMold(fsys, result)
	} else {
		result.ManifestKind = "ingot"
		temperIngot(fsys, result)
	}

	return result
}

// temperMold validates a mold package.
func temperMold(fsys fs.FS, result *TemperResult) {
	m, err := LoadMoldFromFS(fsys, "mold.yaml")
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Severity: SeverityError,
			Message:  fmt.Sprintf("failed to parse mold.yaml: %v", err),
			File:     "mold.yaml",
		})
		return
	}

	result.Name = m.Name
	result.Version = m.Version

	// Validate manifest fields
	if err := ValidateMold(m); err != nil {
		for _, line := range extractValidationErrors(err) {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Severity: SeverityError,
				Message:  line,
				File:     "mold.yaml",
			})
		}
	}

	// Validate output source references
	if err := ValidateOutputSources(m, fsys); err != nil {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Severity: SeverityError,
			Message:  err.Error(),
			File:     "mold.yaml",
		})
	}

	// Validate flux schema consistency
	temperFluxSchema(fsys, m.Flux, result)

	// Validate template syntax for all .md files
	validateTemplates(fsys, result)
}

// temperIngot validates an ingot package.
func temperIngot(fsys fs.FS, result *TemperResult) {
	i, err := LoadIngotFromFS(fsys, "ingot.yaml")
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Severity: SeverityError,
			Message:  fmt.Sprintf("failed to parse ingot.yaml: %v", err),
			File:     "ingot.yaml",
		})
		return
	}

	result.Name = i.Name
	result.Version = i.Version

	// Validate manifest fields
	if err := ValidateIngot(i); err != nil {
		for _, line := range extractValidationErrors(err) {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Severity: SeverityError,
				Message:  line,
				File:     "ingot.yaml",
			})
		}
	}

	// Validate file references
	if len(i.Files) > 0 {
		for _, f := range i.Files {
			if !fileExists(fsys, f) {
				result.Diagnostics = append(result.Diagnostics, Diagnostic{
					Severity: SeverityError,
					Message:  fmt.Sprintf("referenced file not found: %s", f),
					File:     "ingot.yaml",
				})
			}
		}
	}

	// Validate template syntax for all .md files
	validateTemplates(fsys, result)
}

// temperFluxSchema checks flux declarations for consistency.
func temperFluxSchema(fsys fs.FS, manifestFlux []FluxVar, result *TemperResult) {
	// Check inline flux declarations (already validated by ValidateMold)
	// Also check flux.schema.yaml if present
	schemaFlux, err := LoadFluxSchema(fsys, "flux.schema.yaml")
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Severity: SeverityError,
			Message:  fmt.Sprintf("failed to parse flux.schema.yaml: %v", err),
			File:     "flux.schema.yaml",
		})
		return
	}

	if schemaFlux == nil {
		return
	}

	// Validate schema file flux declarations
	for i, f := range schemaFlux {
		if f.Name == "" {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Severity: SeverityError,
				Message:  fmt.Sprintf("flux[%d].name is required", i),
				File:     "flux.schema.yaml",
			})
		}
		if f.Type == "" {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Severity: SeverityError,
				Message:  fmt.Sprintf("flux[%d].type is required", i),
				File:     "flux.schema.yaml",
			})
		} else if !validFluxTypes[f.Type] {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Severity: SeverityError,
				Message:  fmt.Sprintf("flux[%d].type %q is not valid (allowed: string, bool, int, list)", i, f.Type),
				File:     "flux.schema.yaml",
			})
		}
	}

	// Warn if both manifest and schema file define flux vars
	if len(manifestFlux) > 0 && len(schemaFlux) > 0 {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Severity: SeverityWarning,
			Message:  "flux variables defined in both mold.yaml and flux.schema.yaml; schema file takes precedence at runtime",
		})
	}
}

// validateTemplates parses all .md files through Go text/template to catch syntax errors.
func validateTemplates(fsys fs.FS, result *TemperResult) {
	_ = fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}
		// Skip manifest files
		if path == "mold.yaml" || path == "ingot.yaml" {
			return nil
		}

		data, readErr := fs.ReadFile(fsys, path)
		if readErr != nil {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Severity: SeverityError,
				Message:  fmt.Sprintf("failed to read file: %v", readErr),
				File:     path,
			})
			return nil
		}

		content := preProcessTemplate(string(data))
		if _, parseErr := template.New(path).Option("missingkey=zero").Parse(content); parseErr != nil {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Severity: SeverityError,
				Message:  fmt.Sprintf("template syntax error: %v", parseErr),
				File:     path,
			})
		}

		return nil
	})
}

// fileExists checks if a file exists in the given filesystem.
func fileExists(fsys fs.FS, path string) bool {
	_, err := fs.Stat(fsys, path)
	return err == nil
}

// extractValidationErrors splits a multi-line validation error into individual messages.
func extractValidationErrors(err error) []string {
	msg := err.Error()
	// Validation errors are formatted as "...:\n  - error1\n  - error2"
	parts := strings.SplitN(msg, ":\n", 2)
	if len(parts) < 2 {
		return []string{msg}
	}
	lines := strings.Split(parts[1], "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimPrefix(line, "  - ")
		trimmed = strings.TrimSpace(trimmed)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
