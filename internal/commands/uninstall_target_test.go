package commands

import (
	"path/filepath"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/foundry"
)

// owner/repo shorthand must resolve to the canonical github.com source that
// cast records, while a local-path entry recorded verbatim stays addressable.
func TestResolveUninstallTarget_Shorthand(t *testing.T) {
	path := filepath.Join(t.TempDir(), "installed.yaml")
	m := &foundry.InstalledManifest{}
	m.UpsertEntry(foundry.InstalledEntry{Name: "remote", Source: "github.com/acme/mold"})
	m.UpsertEntry(foundry.InstalledEntry{Name: "local", Source: "molds/local"})
	if err := foundry.WriteInstalledManifest(path, m); err != nil {
		t.Fatal(err)
	}

	tests := map[string]string{
		"acme/mold":            "github.com/acme/mold",
		"github.com/acme/mold": "github.com/acme/mold",
		"molds/local":          "molds/local",
	}
	for raw, want := range tests {
		got, _, err := resolveUninstallTarget(path, raw)
		if err != nil {
			t.Fatalf("%s: %v", raw, err)
		}
		if got != want {
			t.Errorf("resolveUninstallTarget(%q) source = %q, want %q", raw, got, want)
		}
	}
}
