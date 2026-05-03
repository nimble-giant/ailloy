// Package merge implements deep-merge of JSON and YAML files for
// `ailloy cast --strategy=merge` output entries. See issue #171.
package merge

import "fmt"

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
	return fmt.Errorf("merge: not implemented")
}
