package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/foundry"
)

// writeIngotFixture writes a minimal single-ingot repo at dir.
func writeIngotFixture(t *testing.T, dir, name, version string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	manifest := "apiVersion: v1\nkind: ingot\nname: " + name + "\nversion: " + version + "\nfiles: [content.md]\n"
	if err := os.WriteFile(filepath.Join(dir, "ingot.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write ingot.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "content.md"), []byte("# "+name+"\n"), 0o644); err != nil {
		t.Fatalf("write content.md: %v", err)
	}
}

// writeMultiIngotFixture writes a multi-ingot repo with N named ingots under ingots/<name>/.
func writeMultiIngotFixture(t *testing.T, dir string, names ...string) {
	t.Helper()
	for _, n := range names {
		sub := filepath.Join(dir, "ingots", n)
		writeIngotFixture(t, sub, n, "1.0.0")
	}
}

func TestRunIngotAdd_SingleIngot_RecordsArtifactAndLock(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "src")
	writeIngotFixture(t, srcDir, "solo", "0.1.0")

	projectDir := filepath.Join(tmp, "proj")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(projectDir, foundry.LockFileName)
	if err := os.WriteFile(lockPath, []byte("apiVersion: v1\nmolds: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	chdir(t, projectDir)
	if err := runIngotAddFromLocal(srcDir, false); err != nil {
		t.Fatalf("runIngotAddFromLocal: %v", err)
	}

	if _, err := os.Stat(filepath.Join(projectDir, ".ailloy", "ingots", "solo", "ingot.yaml")); err != nil {
		t.Fatalf("expected ingot.yaml under .ailloy/ingots/solo/, got: %v", err)
	}

	im, err := foundry.ReadInstalledManifest(filepath.Join(projectDir, ".ailloy", "installed.yaml"))
	if err != nil || im == nil {
		t.Fatalf("read installed manifest: im=%v err=%v", im, err)
	}
	if len(im.Ingots) != 1 || im.Ingots[0].Name != "solo" {
		t.Fatalf("expected 1 ingot entry named solo, got %+v", im.Ingots)
	}
	if im.Ingots[0].Subpath != "" {
		t.Errorf("expected empty subpath for root layout, got %q", im.Ingots[0].Subpath)
	}
	if len(im.Ingots[0].Dependents) != 1 || im.Ingots[0].Dependents[0] != "user" {
		t.Errorf("expected dependents=[user], got %v", im.Ingots[0].Dependents)
	}

	lf, err := foundry.ReadLockFile(lockPath)
	if err != nil || lf == nil {
		t.Fatalf("read lock: lf=%v err=%v", lf, err)
	}
	if len(lf.Ingots) != 1 || lf.Ingots[0].Name != "solo" {
		t.Fatalf("expected 1 ingot in lock named solo, got %+v", lf.Ingots)
	}
}

func TestRunIngotAdd_MultiIngot_InstallsAll(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "src")
	writeMultiIngotFixture(t, srcDir, "header", "footer")

	projectDir := filepath.Join(tmp, "proj")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(projectDir, foundry.LockFileName)
	if err := os.WriteFile(lockPath, []byte("apiVersion: v1\nmolds: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	chdir(t, projectDir)
	if err := runIngotAddFromLocal(srcDir, false); err != nil {
		t.Fatalf("runIngotAddFromLocal: %v", err)
	}

	for _, name := range []string{"header", "footer"} {
		if _, err := os.Stat(filepath.Join(projectDir, ".ailloy", "ingots", name, "ingot.yaml")); err != nil {
			t.Errorf("missing %s: %v", name, err)
		}
	}

	im, err := foundry.ReadInstalledManifest(filepath.Join(projectDir, ".ailloy", "installed.yaml"))
	if err != nil || im == nil {
		t.Fatalf("read installed manifest: im=%v err=%v", im, err)
	}
	if len(im.Ingots) != 2 {
		t.Fatalf("expected 2 ingots, got %d (%+v)", len(im.Ingots), im.Ingots)
	}
	bySubpath := map[string]string{}
	for _, e := range im.Ingots {
		bySubpath[e.Subpath] = e.Name
	}
	if bySubpath["ingots/footer"] != "footer" || bySubpath["ingots/header"] != "header" {
		t.Errorf("expected per-package subpaths, got %v", bySubpath)
	}

	lf, err := foundry.ReadLockFile(lockPath)
	if err != nil || lf == nil {
		t.Fatalf("read lock: lf=%v err=%v", lf, err)
	}
	if len(lf.Ingots) != 2 {
		t.Fatalf("expected 2 ingots in lock, got %d (%+v)", len(lf.Ingots), lf.Ingots)
	}
}

func TestRunIngotAdd_NoManifest_Errors(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "README.md"), []byte("not an ingot"), 0o644); err != nil {
		t.Fatal(err)
	}

	projectDir := filepath.Join(tmp, "proj")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	chdir(t, projectDir)

	err := runIngotAddFromLocal(srcDir, false)
	if err == nil {
		t.Fatal("expected error for repo with no ingot manifests, got nil")
	}
}
