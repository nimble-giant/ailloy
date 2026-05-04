package discover

import (
	"strings"
	"testing"

	"github.com/nimble-giant/ailloy/internal/tui/foundries/data"
	"github.com/nimble-giant/ailloy/pkg/foundry"
)

func TestCurrentMold_NoSelection(t *testing.T) {
	m := Model{}
	if _, _, ok := m.CurrentMold(); ok {
		t.Fatalf("expected ok=false on empty model")
	}
}

func TestCurrentMold_HighlightedEntry(t *testing.T) {
	m := Model{
		filtered: []data.CatalogEntry{
			{Name: "agents", Source: "official/agents"},
		},
		cursor: 0,
	}
	ref, scope, ok := m.CurrentMold()
	if !ok {
		t.Fatalf("expected ok=true with one filtered entry")
	}
	if ref != "official/agents" {
		t.Fatalf("ref = %q want %q", ref, "official/agents")
	}
	if scope != data.ScopeProject {
		t.Fatalf("scope = %v want ScopeProject (default)", scope)
	}
}

// TestRenderCastBadge_NotInstalled — molds the user has never cast get no badge,
// so the row stays clean.
func TestRenderCastBadge_NotInstalled(t *testing.T) {
	m := Model{
		installed: map[string]installedInfo{},
		latest:    map[string]foundry.ResolvedVersion{},
		resolving: map[string]bool{},
	}
	e := data.CatalogEntry{Source: "github.com/x/y"}
	if got := renderCastBadge(m, e); got != "" {
		t.Fatalf("expected empty badge for uninstalled mold, got %q", got)
	}
}

// TestRenderCastBadge_InstalledNoLatest — the user cast it but we haven't
// resolved upstream yet (e.g. they haven't pressed `r`). Show "● installed".
func TestRenderCastBadge_InstalledNoLatest(t *testing.T) {
	key := data.MoldIdentity("github.com/x/y", "")
	m := Model{
		installed: map[string]installedInfo{
			key: {Version: "v0.1.0", Source: "github.com/x/y"},
		},
		latest:    map[string]foundry.ResolvedVersion{},
		resolving: map[string]bool{},
	}
	e := data.CatalogEntry{Source: "github.com/x/y"}
	got := renderCastBadge(m, e)
	if !strings.Contains(got, "● installed") || !strings.Contains(got, "v0.1.0") {
		t.Fatalf("expected '● installed v0.1.0', got %q", got)
	}
	if strings.Contains(got, "checking") {
		t.Fatalf("did not expect 'checking' marker when resolve not in flight, got %q", got)
	}
}

// TestRenderCastBadge_InstalledChecking — we're actively resolving the latest
// version; show the in-flight indicator.
func TestRenderCastBadge_InstalledChecking(t *testing.T) {
	key := data.MoldIdentity("github.com/x/y", "")
	m := Model{
		installed: map[string]installedInfo{
			key: {Version: "v0.1.0", Source: "github.com/x/y"},
		},
		latest:    map[string]foundry.ResolvedVersion{},
		resolving: map[string]bool{key: true},
	}
	e := data.CatalogEntry{Source: "github.com/x/y"}
	got := renderCastBadge(m, e)
	if !strings.Contains(got, "checking") {
		t.Fatalf("expected 'checking' marker, got %q", got)
	}
}

// TestRenderCastBadge_UpToDate — upstream resolves to the same tag+commit as
// what's in the manifest; show the green "up to date" pill.
func TestRenderCastBadge_UpToDate(t *testing.T) {
	key := data.MoldIdentity("github.com/x/y", "")
	m := Model{
		installed: map[string]installedInfo{
			key: {Version: "v0.1.0", Commit: "abc123", Source: "github.com/x/y"},
		},
		latest: map[string]foundry.ResolvedVersion{
			key: {Tag: "v0.1.0", Commit: "abc123"},
		},
		resolving: map[string]bool{},
	}
	e := data.CatalogEntry{Source: "github.com/x/y"}
	got := renderCastBadge(m, e)
	if !strings.Contains(got, "up to date") {
		t.Fatalf("expected 'up to date', got %q", got)
	}
}

// TestRenderCastBadge_UpdateAvailable — upstream tag/commit differs; show the
// amber "update available v_old → v_new" pill.
func TestRenderCastBadge_UpdateAvailable(t *testing.T) {
	key := data.MoldIdentity("github.com/x/y", "")
	m := Model{
		installed: map[string]installedInfo{
			key: {Version: "v0.1.0", Commit: "abc123", Source: "github.com/x/y"},
		},
		latest: map[string]foundry.ResolvedVersion{
			key: {Tag: "v0.2.0", Commit: "def456"},
		},
		resolving: map[string]bool{},
	}
	e := data.CatalogEntry{Source: "github.com/x/y"}
	got := renderCastBadge(m, e)
	if !strings.Contains(got, "update available") {
		t.Fatalf("expected 'update available', got %q", got)
	}
	if !strings.Contains(got, "v0.1.0") || !strings.Contains(got, "v0.2.0") {
		t.Fatalf("expected old and new versions in badge, got %q", got)
	}
}

// TestRenderCastBadge_SubpathMoldMatches is the regression test for the bug
// behind this change. Catalog Source for subpath molds has the form
// "github.com/owner/repo//subpath" while installed manifests store the same
// mold as Source="github.com/owner/repo" + Subpath="subpath". Before the
// MoldIdentity fix, the badge silently dropped off after a TUI restart because
// the lookup keys didn't match.
func TestRenderCastBadge_SubpathMoldMatches(t *testing.T) {
	catalogSource := "github.com/kriscoleman/replicated-foundry//molds/launch"
	key := data.MoldIdentity("github.com/kriscoleman/replicated-foundry", "molds/launch")

	m := Model{
		installed: map[string]installedInfo{
			key: {
				Version: "v0.2.0",
				Source:  "github.com/kriscoleman/replicated-foundry",
				Subpath: "molds/launch",
			},
		},
		latest:    map[string]foundry.ResolvedVersion{},
		resolving: map[string]bool{},
	}
	e := data.CatalogEntry{Source: catalogSource}
	got := renderCastBadge(m, e)
	if !strings.Contains(got, "● installed") {
		t.Fatalf("expected subpath mold to render as installed, got %q", got)
	}
}

func TestApplySessionOverrides_StoresEncoded(t *testing.T) {
	m := Model{}
	m = m.ApplySessionOverrides("official/x", map[string]any{
		"agents.targets":  []string{"opencode", "claude"},
		"agents.parallel": true,
	})
	got := m.pending["official/x"]
	if len(got) != 2 {
		t.Fatalf("len = %d want 2; got %v", len(got), got)
	}
	// Sorted: "agents.parallel" < "agents.targets".
	if got[0] != "agents.parallel=true" {
		t.Fatalf("got[0] = %q", got[0])
	}
	// Slices are emitted as YAML flow sequences so the --set parser produces
	// a real list, not a single space-separated string.
	if got[1] != "agents.targets=[opencode,claude]" {
		t.Fatalf("got[1] = %q want agents.targets=[opencode,claude]", got[1])
	}
}
