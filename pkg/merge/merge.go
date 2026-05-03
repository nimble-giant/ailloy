// Package merge implements deep-merge of JSON and YAML files for
// `ailloy cast --strategy=merge` output entries. See issue #171.
package merge

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Options configures MergeFile.
type Options struct {
	// ForceReplaceOnParseError causes MergeFile to replace an existing
	// destination file with newContent (instead of returning a *ParseError)
	// when the on-disk file cannot be parsed in its expected format.
	ForceReplaceOnParseError bool
}

// ParseError is returned when MergeFile cannot parse the existing file at
// destPath. Callers can detect it with errors.As to suggest the
// --force-replace-on-parse-error flag.
type ParseError struct {
	Path   string
	Format string // "json" or "yaml"
	Err    error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("merge: cannot parse existing %s file %s: %v", e.Format, e.Path, e.Err)
}

func (e *ParseError) Unwrap() error { return e.Err }

// MergeFile merges newContent into the file at destPath.
//
// Behavior:
//   - destPath does not exist: write newContent verbatim, creating parent dirs.
//   - extension not .json/.yaml/.yml: write newContent verbatim (replace).
//   - existing file unparseable: return *ParseError unless opts.ForceReplaceOnParseError.
//   - otherwise: parse both, deep-merge, serialize, atomically replace destPath.
//
// Files are written with mode 0644.
func MergeFile(destPath string, newContent []byte, opts Options) error {
	format := detectFormat(destPath)

	// Unknown ext or no existing file → replace.
	if format == "" {
		return writeAll(destPath, newContent)
	}
	existing, err := os.ReadFile(destPath) // #nosec G304 -- caller-controlled cast destination
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return writeAll(destPath, newContent)
		}
		return fmt.Errorf("read existing %s: %w", destPath, err)
	}

	loadFn, dumpFn := loaderFor(format)
	baseTree, perr := loadFn(existing)
	if perr != nil {
		if opts.ForceReplaceOnParseError {
			return writeAll(destPath, newContent)
		}
		return &ParseError{Path: destPath, Format: format, Err: perr}
	}
	overlayTree, oerr := loadFn(newContent)
	if oerr != nil {
		// New content failed to parse. This is a programmer / template bug,
		// not a user-edited file. Surface plainly.
		return fmt.Errorf("merge: cannot parse new %s content for %s: %w", format, destPath, oerr)
	}

	merged := mergeNodes(baseTree, overlayTree)
	out, derr := dumpFn(merged)
	if derr != nil {
		return fmt.Errorf("merge: serialize %s: %w", destPath, derr)
	}
	return writeAll(destPath, out)
}

func detectFormat(p string) string {
	switch strings.ToLower(filepath.Ext(p)) {
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	default:
		return ""
	}
}

func loaderFor(format string) (func([]byte) (*node, error), func(*node) ([]byte, error)) {
	if format == "json" {
		return loadJSON, dumpJSON
	}
	return loadYAML, dumpYAML
}

func writeAll(destPath string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0750); err != nil { // #nosec G301 -- project directories need group read access
		return fmt.Errorf("create dir for %s: %w", destPath, err)
	}
	//#nosec G306 -- mold outputs need to be readable
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return fmt.Errorf("write %s: %w", destPath, err)
	}
	return nil
}
