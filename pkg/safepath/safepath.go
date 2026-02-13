// Package safepath provides utilities for safe file path operations
// to prevent directory traversal attacks (CWE-22).
package safepath

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Clean sanitizes a path by resolving it to an absolute path and
// ensuring it doesn't contain directory traversal sequences.
func Clean(path string) (string, error) {
	// Clean the path to remove . and .. components
	cleaned := filepath.Clean(path)

	// Convert to absolute path
	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	return abs, nil
}

// ValidateUnder ensures the given path is under the specified base directory.
// This prevents directory traversal attacks by ensuring resolved paths
// don't escape the intended directory.
func ValidateUnder(base, path string) (string, error) {
	// Clean and resolve base path
	cleanBase, err := Clean(base)
	if err != nil {
		return "", fmt.Errorf("invalid base path: %w", err)
	}

	// Clean and resolve target path
	cleanPath, err := Clean(path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// Ensure the path is under the base directory
	if !strings.HasPrefix(cleanPath, cleanBase+string(filepath.Separator)) && cleanPath != cleanBase {
		return "", fmt.Errorf("path %q is outside base directory %q", path, base)
	}

	return cleanPath, nil
}

// Join safely joins path elements and validates the result is under the base.
func Join(base string, elem ...string) (string, error) {
	// Clean base first
	cleanBase, err := Clean(base)
	if err != nil {
		return "", fmt.Errorf("invalid base path: %w", err)
	}

	// Join elements
	joined := filepath.Join(append([]string{cleanBase}, elem...)...)

	// Validate the joined path is still under base
	return ValidateUnder(cleanBase, joined)
}

// IsTraversal checks if a path contains directory traversal sequences.
func IsTraversal(path string) bool {
	// Check for obvious traversal patterns
	if strings.Contains(path, "..") {
		return true
	}

	// Clean the path and compare
	cleaned := filepath.Clean(path)
	return strings.HasPrefix(cleaned, "..")
}
