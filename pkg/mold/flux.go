package mold

import (
	"fmt"
	"maps"
	"strconv"
	"strings"
)

// ApplyFluxDefaults returns a new flux map with default values applied for any
// schema variables that have a default and are not already set in the input map.
// The input map is not mutated.
func ApplyFluxDefaults(schema []FluxVar, flux map[string]string) map[string]string {
	result := make(map[string]string, len(flux))
	maps.Copy(result, flux)

	for _, fv := range schema {
		if fv.Default == "" {
			continue
		}
		if _, exists := result[fv.Name]; !exists {
			result[fv.Name] = fv.Default
		}
	}

	return result
}

// ValidateFlux validates provided flux values against the schema declarations.
// It checks that all required variables are present and that values match their
// declared types. All errors are collected and returned at once.
func ValidateFlux(schema []FluxVar, flux map[string]string) error {
	var errs []string

	for _, fv := range schema {
		val, exists := flux[fv.Name]

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
	default:
		return fmt.Sprintf("flux %q has unknown type %q", name, typ)
	}
}
