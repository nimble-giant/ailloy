package index

import (
	"fmt"
	"os"

	"github.com/nimble-giant/ailloy/pkg/foundry"
)

// TemperOptions configures Temper behavior.
type TemperOptions struct {
	// Offline skips all network checks; only schema validation runs.
	Offline bool
	// NoRecurse skips descent into nested foundries (still validates the
	// immediate index's own molds).
	NoRecurse bool
	// MoldVerifier overrides the per-mold reference check, primarily for
	// testing. When nil, foundry.ResolveVersion via DefaultGitRunner is used.
	MoldVerifier func(source string) error
	// FoundryFetcher overrides nested-foundry fetching, primarily for testing.
	// When nil, the default network-backed Fetcher is used (which also writes
	// the on-disk cache).
	FoundryFetcher func(source string) (*Index, error)
}

// TemperFinding is one issue found during validation.
type TemperFinding struct {
	Severity string // "error" today; reserved for future "warning"
	Path     string // dotted/bracketed path to the offending field
	Source   string // the source URL the finding refers to (mold or foundry)
	Err      error  // underlying cause
}

// TemperResult is the aggregate output of a Temper run.
type TemperResult struct {
	Index    *Index // parsed root index, even when findings are present
	Findings []TemperFinding
}

// HasErrors reports whether any finding is severity "error".
func (r *TemperResult) HasErrors() bool {
	for _, f := range r.Findings {
		if f.Severity == "error" {
			return true
		}
	}
	return false
}

// Temper reads, parses, and validates a foundry index file at path.
// See TemperBytes for the validation behaviour.
func Temper(path string, opts TemperOptions) (*TemperResult, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- caller-supplied foundry index path
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return TemperBytes(data, opts)
}

// TemperBytes validates an already-loaded foundry index.
//
// Validation order:
//  1. Parse YAML.
//  2. Schema validation via Index.Validate(). Failures here short-circuit
//     remaining checks (returning a TemperResult with one finding).
//  3. (network) Per direct mold: parse the source ref and resolve it on the
//     remote via git ls-remote. Each failure is reported as a finding.
//  4. (network, unless NoRecurse) Per nested foundry: fetch the child index,
//     run the same checks recursively. Cycles are silently broken via a
//     visited-set keyed on canonical source URL.
//
// Network steps are skipped entirely when opts.Offline is true.
func TemperBytes(data []byte, opts TemperOptions) (*TemperResult, error) {
	idx, err := ParseIndex(data)
	if err != nil {
		return nil, err
	}

	res := &TemperResult{Index: idx}
	if err := idx.Validate(); err != nil {
		res.Findings = append(res.Findings, TemperFinding{Severity: "error", Err: err})
		return res, nil
	}

	if opts.Offline {
		return res, nil
	}

	moldVerifier := opts.MoldVerifier
	if moldVerifier == nil {
		git := foundry.DefaultGitRunner()
		moldVerifier = func(source string) error {
			ref, err := foundry.ParseReference(source)
			if err != nil {
				return err
			}
			_, err = foundry.ResolveVersion(ref, git)
			return err
		}
	}

	foundryFetcher := opts.FoundryFetcher
	if foundryFetcher == nil {
		fetcher, err := NewFetcher(defaultGitRunnerForSearch())
		if err != nil {
			return nil, fmt.Errorf("constructing fetcher: %w", err)
		}
		foundryFetcher = func(source string) (*Index, error) {
			url := NormalizeFoundryURL(source)
			entry := &FoundryEntry{URL: url, Type: DetectType(url)}
			return fetcher.FetchIndex(entry)
		}
	}

	// Verify direct molds.
	for i, m := range idx.Molds {
		if err := moldVerifier(m.Source); err != nil {
			res.Findings = append(res.Findings, TemperFinding{
				Severity: "error",
				Path:     fmt.Sprintf("molds[%d] (%s)", i, m.Name),
				Source:   m.Source,
				Err:      err,
			})
		}
	}

	// Recursively walk nested foundries.
	if !opts.NoRecurse {
		visited := map[string]bool{}
		// Mark the root visited (using its declared source-of-record: we don't
		// know the root's URL here, so visited[] starts empty and children
		// short-circuit each other.)
		for i, f := range idx.Foundries {
			res.Findings = append(res.Findings,
				temperNested(f.Source, fmt.Sprintf("foundries[%d] (%s)", i, f.Name),
					visited, moldVerifier, foundryFetcher, opts.NoRecurse)...)
		}
	}

	return res, nil
}

// temperNested validates a nested foundry by source URL: fetches the child
// index (which also schema-validates it), then verifies each of its molds and
// recurses into its own nested foundries. Cycles are short-circuited via the
// shared visited map.
func temperNested(
	source, pathLabel string,
	visited map[string]bool,
	moldVerifier func(string) error,
	foundryFetcher func(string) (*Index, error),
	noRecurse bool,
) []TemperFinding {
	key := canonicalizeSource(source)
	if visited[key] {
		return nil
	}
	visited[key] = true

	idx, err := foundryFetcher(source)
	if err != nil {
		return []TemperFinding{{
			Severity: "error",
			Path:     pathLabel,
			Source:   source,
			Err:      err,
		}}
	}

	var out []TemperFinding
	for i, m := range idx.Molds {
		if err := moldVerifier(m.Source); err != nil {
			out = append(out, TemperFinding{
				Severity: "error",
				Path:     fmt.Sprintf("%s -> molds[%d] (%s)", pathLabel, i, m.Name),
				Source:   m.Source,
				Err:      err,
			})
		}
	}
	if !noRecurse {
		for i, f := range idx.Foundries {
			child := fmt.Sprintf("%s -> foundries[%d] (%s)", pathLabel, i, f.Name)
			out = append(out, temperNested(f.Source, child, visited, moldVerifier, foundryFetcher, noRecurse)...)
		}
	}
	return out
}
