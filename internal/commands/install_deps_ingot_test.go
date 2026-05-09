package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/mold"
)

func TestInstallDeclaredDeps_BareIngotRef_MultiLayout_InstallsAll(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "src")
	writeMultiIngotFixture(t, srcDir, "header", "footer")

	projectDir := filepath.Join(tmp, "proj")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	chdir(t, projectDir)

	manifest := &mold.Mold{
		APIVersion: "v1",
		Kind:       "mold",
		Name:       "consumer",
		Version:    "0.1.0",
		Dependencies: []mold.Dependency{
			{Ingot: srcDir, Version: "local"},
		},
	}

	if err := installDeclaredDeps(manifest, "test/consumer", false, true, false, true, nil); err != nil {
		t.Fatalf("installDeclaredDeps: %v", err)
	}

	im, err := foundry.ReadInstalledManifest(filepath.Join(projectDir, ".ailloy", "installed.yaml"))
	if err != nil || im == nil {
		t.Fatalf("read installed manifest: im=%v err=%v", im, err)
	}
	if len(im.Ingots) != 2 {
		t.Fatalf("expected 2 ingot entries, got %d (%+v)", len(im.Ingots), im.Ingots)
	}
	for _, e := range im.Ingots {
		if want := []string{"test/consumer"}; !equalStrings(e.Dependents, want) {
			t.Errorf("ingot %q: dependents=%v, want %v", e.Name, e.Dependents, want)
		}
	}
}

func TestInstallDeclaredDeps_SubpathIngotRef_InstallsOne(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "src")
	writeMultiIngotFixture(t, srcDir, "header", "footer")

	projectDir := filepath.Join(tmp, "proj")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	chdir(t, projectDir)

	// Local refs don't natively support //subpath, so point directly at the
	// header subdir. The pipeline treats this the same as a resolver-narrowed
	// remote ref: a single ingot at root.
	manifest := &mold.Mold{
		APIVersion: "v1",
		Kind:       "mold",
		Name:       "consumer",
		Version:    "0.1.0",
		Dependencies: []mold.Dependency{
			{Ingot: filepath.Join(srcDir, "ingots", "header"), Version: "local"},
		},
	}

	if err := installDeclaredDeps(manifest, "test/consumer", false, true, false, true, nil); err != nil {
		t.Fatalf("installDeclaredDeps: %v", err)
	}

	im, err := foundry.ReadInstalledManifest(filepath.Join(projectDir, ".ailloy", "installed.yaml"))
	if err != nil || im == nil {
		t.Fatalf("read installed manifest: im=%v err=%v", im, err)
	}
	if len(im.Ingots) != 1 || im.Ingots[0].Name != "header" {
		t.Fatalf("expected 1 ingot named header, got %+v", im.Ingots)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
