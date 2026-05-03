package extensions

import (
	"path/filepath"
	"testing"
)

func TestNewManager_RespectsConfigDirEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AILLOY_CONFIG_DIR", dir)
	m, err := NewManager("v0.43.0")
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	if got, want := m.host.ManifestPath(), filepath.Join(dir, "extensions", "manifest.yaml"); got != want {
		t.Errorf("ManifestPath = %q, want %q", got, want)
	}
}

func TestList_IncludesRegisteredEvenWhenNotInstalled(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AILLOY_CONFIG_DIR", dir)
	m, err := NewManager("v0.43.0")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := m.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	var found bool
	for _, e := range entries {
		if e.Name == "docs" {
			found = true
			if e.Installed {
				t.Errorf("docs should not be installed in fresh config dir")
			}
			if e.Description == "" {
				t.Errorf("docs entry should have a description")
			}
		}
	}
	if !found {
		t.Errorf("docs extension should appear in List() even when uninstalled")
	}
}

func TestIsInstalled_FalseOnFreshConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AILLOY_CONFIG_DIR", dir)
	m, err := NewManager("v0.43.0")
	if err != nil {
		t.Fatal(err)
	}
	if m.IsInstalled("docs") {
		t.Errorf("docs should not be installed in fresh config dir")
	}
	if m.IsDeclined("docs") {
		t.Errorf("docs should not be declined in fresh config dir")
	}
}

func TestRecordDeclined_PersistsAndReadsBack(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AILLOY_CONFIG_DIR", dir)
	m, err := NewManager("v0.43.0")
	if err != nil {
		t.Fatal(err)
	}
	if err := m.recordDeclined("docs"); err != nil {
		t.Fatalf("recordDeclined: %v", err)
	}
	if !m.IsDeclined("docs") {
		t.Errorf("after recordDeclined, IsDeclined should be true")
	}
}

func TestReset_ConsentOnlyClearsDeclinedFlag(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AILLOY_CONFIG_DIR", dir)
	m, err := NewManager("v0.43.0")
	if err != nil {
		t.Fatal(err)
	}
	if err := m.recordDeclined("docs"); err != nil {
		t.Fatal(err)
	}
	if err := m.Reset(true); err != nil {
		t.Fatalf("Reset(consentOnly): %v", err)
	}
	if m.IsDeclined("docs") {
		t.Errorf("after Reset(consentOnly), declined flag should clear")
	}
}
