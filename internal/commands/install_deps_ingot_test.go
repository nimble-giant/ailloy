package commands

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

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
		if want := []string{"test/consumer"}; !slices.Equal(e.Dependents, want) {
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

func TestInstallDeclaredDeps_BareIngotRef_DoubleInvoke_NoReInstall(t *testing.T) {
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
		t.Fatalf("first install: %v", err)
	}

	im, err := foundry.ReadInstalledManifest(filepath.Join(projectDir, ".ailloy", "installed.yaml"))
	if err != nil || im == nil {
		t.Fatalf("read manifest: im=%v err=%v", im, err)
	}
	if len(im.Ingots) != 2 {
		t.Fatalf("expected 2 ingots after first install, got %d", len(im.Ingots))
	}
	firstStamps := map[string]time.Time{}
	for _, e := range im.Ingots {
		firstStamps[e.Subpath] = e.InstalledAt
	}

	// Second invocation should be a no-op for the install paths.
	if err := installDeclaredDeps(manifest, "test/consumer", false, true, false, true, nil); err != nil {
		t.Fatalf("second install: %v", err)
	}

	im2, err := foundry.ReadInstalledManifest(filepath.Join(projectDir, ".ailloy", "installed.yaml"))
	if err != nil || im2 == nil {
		t.Fatalf("read manifest after 2nd: im=%v err=%v", im2, err)
	}
	if len(im2.Ingots) != 2 {
		t.Fatalf("expected 2 ingots after second install, got %d", len(im2.Ingots))
	}
	for _, e := range im2.Ingots {
		if !e.InstalledAt.Equal(firstStamps[e.Subpath]) {
			t.Errorf("ingot %q (subpath %q): InstalledAt changed from %v to %v — re-install happened",
				e.Name, e.Subpath, firstStamps[e.Subpath], e.InstalledAt)
		}
		if !slices.Equal(e.Dependents, []string{"test/consumer"}) {
			t.Errorf("ingot %q: dependents=%v, expected [test/consumer]", e.Name, e.Dependents)
		}
	}
}
