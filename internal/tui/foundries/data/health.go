package data

import (
	"os"

	"github.com/goccy/go-yaml"
	"github.com/nimble-giant/ailloy/pkg/assay"
	"github.com/nimble-giant/ailloy/pkg/foundry/index"
	"github.com/nimble-giant/ailloy/pkg/mold"
)

// Severity grades a finding for display order and styling.
type Severity int

const (
	SevError Severity = iota
	SevWarn
	SevInfo
)

// Finding is one row in the Health tab.
type Finding struct {
	Severity Severity
	Source   string // mold source or scope
	Title    string
	Detail   string
}

// DriftFindings inspects the inventory for orphaned and legacy entries.
// `catalog` should be the result of LoadCatalog(cfg).
func DriftFindings(_ *index.Config, items []InventoryItem, catalog []CatalogEntry) []Finding {
	bySource := map[string]struct{}{}
	for _, c := range catalog {
		bySource[c.Source] = struct{}{}
	}
	var out []Finding
	for _, it := range items {
		if it.Entry.Files == nil {
			out = append(out, Finding{
				Severity: SevWarn,
				Source:   it.Entry.Source,
				Title:    "Legacy install manifest",
				Detail:   "Re-cast to enable safe uninstall.",
			})
		}
		if _, ok := bySource[it.Entry.Source]; !ok {
			out = append(out, Finding{
				Severity: SevWarn,
				Source:   it.Entry.Source,
				Title:    "Orphaned mold",
				Detail:   "No registered foundry indexes this mold anymore.",
			})
		}
	}
	return out
}

// AssayFindings runs `assay` over each rendered blank dir and surfaces its
// diagnostics as findings. Errors from individual runs become findings;
// we never propagate.
func AssayFindings(blankDirs []string) []Finding {
	var out []Finding
	for _, dir := range blankDirs {
		res, err := assay.Assay(dir, nil)
		if err != nil {
			out = append(out, Finding{Severity: SevWarn, Source: dir, Title: "assay error", Detail: err.Error()})
			continue
		}
		for _, d := range res.Diagnostics {
			sev := SevInfo
			switch d.Severity {
			case mold.SeverityError:
				sev = SevError
			case mold.SeverityWarning:
				sev = SevWarn
			}
			out = append(out, Finding{Severity: sev, Source: d.File, Title: d.Rule, Detail: d.Message})
		}
	}
	return out
}

// stateFile mirrors the .ailloy/state.yaml format written by `cast`.
type stateFile struct {
	BlankDirs    []string `yaml:"blankDirs,omitempty"`
	WorkflowDirs []string `yaml:"workflowDirs,omitempty"`
}

// ReadBlankDirs returns the BlankDirs field from a state file at stateYAMLPath,
// or nil if missing/unreadable.
func ReadBlankDirs(stateYAMLPath string) []string {
	data, err := os.ReadFile(stateYAMLPath) // #nosec G304 -- path under user control by design
	if err != nil {
		return nil
	}
	var s stateFile
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil
	}
	return s.BlankDirs
}
