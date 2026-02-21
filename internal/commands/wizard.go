package commands

import (
	"strings"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

// parseCustomVars parses "key=value\n" text into a map
func parseCustomVars(raw string) map[string]string {
	vars := make(map[string]string)
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key != "" {
			vars[key] = value
		}
	}
	return vars
}

// --- helpers ---

// getFluxString retrieves a string value from nested flux by dotted path.
func getFluxString(flux map[string]any, path string) (string, bool) {
	val, ok := mold.GetNestedAny(flux, path)
	if !ok {
		return "", false
	}
	s, ok := val.(string)
	return s, ok
}

// getFluxBool retrieves a bool value from nested flux by dotted path.
func getFluxBool(flux map[string]any, path string) (bool, bool) {
	val, ok := mold.GetNestedAny(flux, path)
	if !ok {
		return false, false
	}
	b, ok := val.(bool)
	return b, ok
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
