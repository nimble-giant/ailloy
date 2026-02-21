package mold

import (
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"

	"dario.cat/mergo"
	"github.com/goccy/go-yaml"
)

// SetNestedValue sets a value in a nested map using a dotted path.
// For example, SetNestedValue(m, "scm.provider", "GitHub") creates
// m["scm"] = map[string]any{"provider": "GitHub"}.
func SetNestedValue(m map[string]any, dottedKey string, value string) {
	segments := strings.Split(dottedKey, ".")
	current := m
	for i, seg := range segments {
		if i == len(segments)-1 {
			current[seg] = value
			return
		}
		next, ok := current[seg].(map[string]any)
		if !ok {
			next = make(map[string]any)
			current[seg] = next
		}
		current = next
	}
}

// GetNestedValue retrieves a leaf string value from a nested map using a dotted path.
func GetNestedValue(m map[string]any, dottedPath string) (string, bool) {
	segments := strings.Split(dottedPath, ".")
	var current any = m
	for _, seg := range segments {
		cm, ok := current.(map[string]any)
		if !ok {
			return "", false
		}
		current, ok = cm[seg]
		if !ok {
			return "", false
		}
	}
	s, ok := current.(string)
	return s, ok
}

// ApplyFluxDefaults returns a new flux map with default values applied for any
// schema variables that have a default and are not already set in the input map.
// The input map is not mutated.
func ApplyFluxDefaults(schema []FluxVar, flux map[string]any) map[string]any {
	result := make(map[string]any, len(flux))
	_ = mergo.Merge(&result, flux, mergo.WithOverride)

	for _, fv := range schema {
		if fv.Default == "" {
			continue
		}
		if _, found := GetNestedValue(result, fv.Name); !found {
			SetNestedValue(result, fv.Name, fv.Default)
		}
	}

	return result
}

// ValidateFlux validates provided flux values against the schema declarations.
// It checks that all required variables are present and that values match their
// declared types. All errors are collected and returned at once.
func ValidateFlux(schema []FluxVar, flux map[string]any) error {
	var errs []string

	for _, fv := range schema {
		val, exists := GetNestedValue(flux, fv.Name)

		// Check required
		if fv.Required && (!exists || val == "") {
			errs = append(errs, fmt.Sprintf("flux %q is required but not provided", fv.Name))
			continue
		}

		// Skip type validation if value is not set
		if !exists || val == "" {
			continue
		}

		// Type validation
		if err := validateFluxType(fv.Type, fv.Name, val); err != "" {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("flux validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

// LoadFluxFile loads a nested map from a YAML file in the given filesystem.
// Returns an empty map (not an error) if the file does not exist.
func LoadFluxFile(fsys fs.FS, path string) (map[string]any, error) {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return make(map[string]any), nil //nolint:nilerr // missing file is not an error
	}

	var vals map[string]any
	if err := yaml.Unmarshal(data, &vals); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if vals == nil {
		return make(map[string]any), nil
	}
	return vals, nil
}

// LoadFluxSchema loads a FluxVar schema from a YAML file in the given filesystem.
// Returns nil (not an error) if the file does not exist.
func LoadFluxSchema(fsys fs.FS, path string) ([]FluxVar, error) {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, nil //nolint:nilerr // missing file is not an error
	}

	var schema []FluxVar
	if err := yaml.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return schema, nil
}

// ApplyFluxFileDefaults returns a new flux map with defaults from the given map
// applied for any keys not already set. The input map is not mutated.
func ApplyFluxFileDefaults(defaults, flux map[string]any) map[string]any {
	result := make(map[string]any)
	_ = mergo.Merge(&result, flux, mergo.WithOverride)
	_ = mergo.Merge(&result, defaults)

	return result
}

// LayerFluxFiles loads YAML files from OS paths and deep-merges them left-to-right.
// Each successive file overrides values from the previous ones.
func LayerFluxFiles(paths []string) (map[string]any, error) {
	result := make(map[string]any)

	for _, p := range paths {
		data, err := os.ReadFile(p) // #nosec G304 -- CLI tool reads user-specified flux files
		if err != nil {
			return nil, fmt.Errorf("reading flux file %s: %w", p, err)
		}

		var vals map[string]any
		if err := yaml.Unmarshal(data, &vals); err != nil {
			return nil, fmt.Errorf("parsing flux file %s: %w", p, err)
		}
		if vals != nil {
			_ = mergo.Merge(&result, vals, mergo.WithOverride)
		}
	}

	return result, nil
}

// ApplySetOverrides applies --set key=value flags to a flux map using dotted paths.
func ApplySetOverrides(flux map[string]any, setFlags []string) error {
	for _, flag := range setFlags {
		parts := strings.SplitN(flag, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid --set format: %q (expected key=value)", flag)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			return fmt.Errorf("--set key cannot be empty")
		}
		SetNestedValue(flux, key, value)
	}
	return nil
}

// GetNestedAny retrieves any value (not just string) from a nested map by dotted path.
func GetNestedAny(m map[string]any, dottedPath string) (any, bool) {
	segments := strings.Split(dottedPath, ".")
	var current any = m
	for _, seg := range segments {
		cm, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = cm[seg]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

// SetNestedAny sets any value (not just string) in a nested map by dotted path.
func SetNestedAny(m map[string]any, dottedKey string, value any) {
	segments := strings.Split(dottedKey, ".")
	current := m
	for i, seg := range segments {
		if i == len(segments)-1 {
			current[seg] = value
			return
		}
		next, ok := current[seg].(map[string]any)
		if !ok {
			next = make(map[string]any)
			current[seg] = next
		}
		current = next
	}
}

// validateFluxType checks that a value conforms to the declared type.
// Returns an error message string, or empty string if valid.
func validateFluxType(typ, name, val string) string {
	switch typ {
	case "string":
		// Any value is valid
		return ""
	case "bool":
		lower := strings.ToLower(val)
		if lower != "true" && lower != "false" {
			return fmt.Sprintf("flux %q must be a bool (true/false), got %q", name, val)
		}
		return ""
	case "int":
		if _, err := strconv.Atoi(val); err != nil {
			return fmt.Sprintf("flux %q must be an int, got %q", name, val)
		}
		return ""
	case "list":
		// Any non-empty comma-separated value is valid; already checked non-empty above
		return ""
	case "select":
		// Any value is valid (must match one of the declared options at runtime)
		return ""
	default:
		return fmt.Sprintf("flux %q has unknown type %q", name, typ)
	}
}
