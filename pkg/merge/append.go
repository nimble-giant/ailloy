package merge

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// AppendOptions configures AppendFile.
type AppendOptions struct {
	// MoldName is the mold whose content is being appended. Used as the key
	// for the sentinel block so re-casting the same mold updates its block
	// in place rather than producing duplicate entries.
	MoldName string
}

// ErrUnsupportedAppendExt is returned when AppendFile is called with a
// destination whose extension is not one of the markdown variants we
// currently support. Callers should surface this to the user so they
// understand that strategy: append works only on markdown files in v1.
var ErrUnsupportedAppendExt = fmt.Errorf("strategy: append currently supports only .md and .markdown destinations")

// AppendFile appends newContent to the file at destPath, idempotently per
// opts.MoldName. The content is wrapped in a sentinel block:
//
//	<!-- ailloy:mold=<name>:start -->
//	<newContent>
//	<!-- ailloy:mold=<name>:end -->
//
// On re-cast the existing block for the mold is updated in place.
//
// Files are written with mode 0644.
func AppendFile(destPath string, newContent []byte, opts AppendOptions) error {
	if opts.MoldName == "" {
		return fmt.Errorf("AppendFile: MoldName is required")
	}
	switch strings.ToLower(filepath.Ext(destPath)) {
	case ".md", ".markdown":
		// supported
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedAppendExt, destPath)
	}

	// Build the sentinel block.
	startMark := fmt.Sprintf("<!-- ailloy:mold=%s:start -->", opts.MoldName)
	endMark := fmt.Sprintf("<!-- ailloy:mold=%s:end -->", opts.MoldName)
	body := bytes.TrimRight(newContent, "\n")
	block := fmt.Sprintf("%s\n%s\n%s\n", startMark, body, endMark)

	existing, err := os.ReadFile(destPath) // #nosec G304 -- caller-controlled cast destination
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("read existing %s: %w", destPath, err)
		}
		// Doesn't exist — create with just our block.
		return writeAll(destPath, []byte(block))
	}

	// Look for an existing block keyed by MoldName.
	pattern := regexp.MustCompile(
		"(?s)" + regexp.QuoteMeta(startMark) + `\s*\n.*?` + regexp.QuoteMeta(endMark) + `\n?`,
	)
	if pattern.Match(existing) {
		// Replace in place.
		updated := pattern.ReplaceAll(existing, []byte(block))
		return writeAll(destPath, updated)
	}

	// Append a new block. Ensure separation from existing content.
	out := bytes.TrimRight(existing, "\n")
	var buf bytes.Buffer
	buf.Write(out)
	if len(out) > 0 {
		buf.WriteString("\n\n")
	}
	buf.WriteString(block)
	return writeAll(destPath, buf.Bytes())
}
