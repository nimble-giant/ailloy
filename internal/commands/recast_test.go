package commands

import (
	"strings"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/foundry"
)

// resolveRecastTarget must reject ambiguous bare names (same name across two
// subpaths in one foundry) with a disambiguation list, rather than silently
// operating on whichever entry happens to come first.
func TestResolveRecastTarget_AmbiguousNameErrors(t *testing.T) {
	manifest := &foundry.InstalledManifest{
		APIVersion: "v1",
		Molds: []foundry.InstalledEntry{
			{Name: "shortcut", Source: "github.com/k/foundry", Subpath: "molds/a", Version: "v1.0.0"},
			{Name: "shortcut", Source: "github.com/k/foundry", Subpath: "molds/b", Version: "v1.0.0"},
		},
	}

	_, err := resolveRecastTarget(manifest, "shortcut")
	if err == nil {
		t.Fatal("expected ambiguity error for bare name with multiple matches, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "multiple installed molds") {
		t.Errorf("error missing disambiguation phrasing: %q", msg)
	}
	if !strings.Contains(msg, "github.com/k/foundry//molds/a") || !strings.Contains(msg, "github.com/k/foundry//molds/b") {
		t.Errorf("error missing both candidate refs: %q", msg)
	}
}

func TestResolveRecastTarget_UniqueName(t *testing.T) {
	manifest := &foundry.InstalledManifest{
		APIVersion: "v1",
		Molds: []foundry.InstalledEntry{
			{Name: "shortcut", Source: "github.com/k/foundry", Subpath: "molds/a"},
			{Name: "linear", Source: "github.com/k/foundry", Subpath: "molds/b"},
		},
	}

	got, err := resolveRecastTarget(manifest, "linear")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Subpath != "molds/b" {
		t.Errorf("resolved entry = %+v, want subpath molds/b", got)
	}
}

func TestResolveRecastTarget_FullRefDisambiguates(t *testing.T) {
	manifest := &foundry.InstalledManifest{
		APIVersion: "v1",
		Molds: []foundry.InstalledEntry{
			{Name: "shortcut", Source: "github.com/k/foundry", Subpath: "molds/a", Version: "v1.0.0"},
			{Name: "shortcut", Source: "github.com/k/foundry", Subpath: "molds/b", Version: "v2.0.0"},
		},
	}

	got, err := resolveRecastTarget(manifest, "github.com/k/foundry//molds/b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Subpath != "molds/b" || got.Version != "v2.0.0" {
		t.Errorf("resolved entry = %+v, want subpath molds/b version v2.0.0", got)
	}
}

func TestResolveRecastTarget_NotFound(t *testing.T) {
	manifest := &foundry.InstalledManifest{APIVersion: "v1"}

	if _, err := resolveRecastTarget(manifest, "missing"); err == nil {
		t.Fatal("expected not-found error for empty manifest, got nil")
	}
	if _, err := resolveRecastTarget(manifest, "github.com/k/foundry//molds/x"); err == nil {
		t.Fatal("expected not-found error for missing full ref, got nil")
	}
}
