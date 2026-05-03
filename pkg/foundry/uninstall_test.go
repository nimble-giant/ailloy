package foundry

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeFileT(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func sha256Hex(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}

func setupManifest(t *testing.T, files []string, hashes map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	manifestPath := filepath.Join(dir, ".ailloy", "installed.yaml")
	m := &InstalledManifest{
		APIVersion: "v1",
		Molds: []InstalledEntry{
			{
				Name:       "test-mold",
				Source:     "github.com/x/y",
				Version:    "v1",
				Commit:     "c1",
				CastAt:     time.Now().UTC(),
				Files:      files,
				FileHashes: hashes,
			},
		},
	}
	if err := WriteInstalledManifest(manifestPath, m); err != nil {
		t.Fatal(err)
	}
	return manifestPath
}

func TestUninstallMold_Clean(t *testing.T) {
	hashes := map[string]string{
		"agents.md":     sha256Hex("hello"),
		"skills/x/y.md": sha256Hex("world"),
	}
	manifestPath := setupManifest(t, []string{"agents.md", "skills/x/y.md"}, hashes)
	writeFileT(t, "agents.md", "hello")
	writeFileT(t, "skills/x/y.md", "world")

	res, err := UninstallMold(manifestPath, "github.com/x/y", "", UninstallOptions{})
	if err != nil {
		t.Fatalf("UninstallMold: %v", err)
	}
	if len(res.Deleted) != 2 {
		t.Errorf("Deleted = %v, want 2", res.Deleted)
	}
	if _, err := os.Stat("agents.md"); !os.IsNotExist(err) {
		t.Error("agents.md not removed")
	}
	if _, err := os.Stat("skills"); !os.IsNotExist(err) {
		t.Error("empty skills/ dir not pruned")
	}

	loaded, _ := ReadInstalledManifest(manifestPath)
	if len(loaded.Molds) != 0 {
		t.Errorf("entry not removed from manifest, got %d", len(loaded.Molds))
	}
}

func TestUninstallMold_ModifiedFile_Skipped(t *testing.T) {
	hashes := map[string]string{"agents.md": sha256Hex("original content")}
	manifestPath := setupManifest(t, []string{"agents.md"}, hashes)
	writeFileT(t, "agents.md", "modified content")

	res, err := UninstallMold(manifestPath, "github.com/x/y", "", UninstallOptions{})
	if err != nil {
		t.Fatalf("UninstallMold: %v", err)
	}
	if len(res.SkippedModified) != 1 {
		t.Errorf("expected SkippedModified=[agents.md], got %v", res.SkippedModified)
	}
	if _, err := os.Stat("agents.md"); err != nil {
		t.Error("modified file should not be removed")
	}
}

func TestUninstallMold_ModifiedFile_Force(t *testing.T) {
	hashes := map[string]string{"agents.md": sha256Hex("original")}
	manifestPath := setupManifest(t, []string{"agents.md"}, hashes)
	writeFileT(t, "agents.md", "modified")

	res, err := UninstallMold(manifestPath, "github.com/x/y", "", UninstallOptions{Force: true})
	if err != nil {
		t.Fatalf("UninstallMold: %v", err)
	}
	if len(res.Deleted) != 1 {
		t.Errorf("Deleted = %v, want [agents.md]", res.Deleted)
	}
}

func TestUninstallMold_Missing(t *testing.T) {
	hashes := map[string]string{"agents.md": sha256Hex("anything")}
	manifestPath := setupManifest(t, []string{"agents.md"}, hashes)

	res, err := UninstallMold(manifestPath, "github.com/x/y", "", UninstallOptions{})
	if err != nil {
		t.Fatalf("UninstallMold: %v", err)
	}
	if len(res.NotFound) != 1 || res.NotFound[0] != "agents.md" {
		t.Errorf("expected NotFound=[agents.md], got %v", res.NotFound)
	}
}

func TestUninstallMold_DryRun(t *testing.T) {
	hashes := map[string]string{"agents.md": sha256Hex("x")}
	manifestPath := setupManifest(t, []string{"agents.md"}, hashes)
	writeFileT(t, "agents.md", "x")

	res, err := UninstallMold(manifestPath, "github.com/x/y", "", UninstallOptions{DryRun: true})
	if err != nil {
		t.Fatalf("UninstallMold: %v", err)
	}
	if len(res.Deleted) != 1 {
		t.Errorf("dry-run should report Deleted=[agents.md], got %v", res.Deleted)
	}
	if _, err := os.Stat("agents.md"); err != nil {
		t.Error("dry-run should NOT remove the file")
	}
	loaded, _ := ReadInstalledManifest(manifestPath)
	if len(loaded.Molds) != 1 {
		t.Error("dry-run should NOT modify the manifest")
	}
}

func TestUninstallMold_SharedFile_Retained(t *testing.T) {
	hash := sha256Hex("shared")
	hashes := map[string]string{"agents.md": hash}
	manifestPath := setupManifest(t, []string{"agents.md"}, hashes)
	writeFileT(t, "agents.md", "shared")

	m, _ := ReadInstalledManifest(manifestPath)
	m.Molds = append(m.Molds, InstalledEntry{
		Name:       "other",
		Source:     "github.com/x/other",
		Files:      []string{"agents.md"},
		FileHashes: map[string]string{"agents.md": hash},
	})
	if err := WriteInstalledManifest(manifestPath, m); err != nil {
		t.Fatal(err)
	}

	res, err := UninstallMold(manifestPath, "github.com/x/y", "", UninstallOptions{})
	if err != nil {
		t.Fatalf("UninstallMold: %v", err)
	}
	if len(res.Retained) != 1 {
		t.Errorf("expected Retained=[agents.md], got %v", res.Retained)
	}
	if _, err := os.Stat("agents.md"); err != nil {
		t.Error("shared file should be retained on disk")
	}
}

func TestUninstallMold_LegacyEntry_NoFiles(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	manifestPath := filepath.Join(dir, ".ailloy", "installed.yaml")
	m := &InstalledManifest{
		APIVersion: "v1",
		Molds: []InstalledEntry{
			{Name: "legacy", Source: "github.com/x/y", Version: "v1", Commit: "c1"},
		},
	}
	if err := WriteInstalledManifest(manifestPath, m); err != nil {
		t.Fatal(err)
	}

	res, err := UninstallMold(manifestPath, "github.com/x/y", "", UninstallOptions{})
	if err == nil {
		t.Fatal("expected ErrLegacyEntry")
	}
	if !res.LegacyManifest {
		t.Errorf("expected LegacyManifest=true, got %+v", res)
	}
}

// Pre-existing project with both manifest and lock — uninstall should remove
// the entry from both.
func TestUninstallMold_DropsLockEntryToo(t *testing.T) {
	hashes := map[string]string{"agents.md": sha256Hex("x")}
	manifestPath := setupManifest(t, []string{"agents.md"}, hashes)
	writeFileT(t, "agents.md", "x")

	// Cwd is the project root; LockFileName resolves there. Seed the lock.
	lock := &LockFile{
		APIVersion: "v1",
		Molds: []LockEntry{
			{Name: "test-mold", Source: "github.com/x/y", Version: "v1", Commit: "c1", Timestamp: time.Now().UTC()},
			{Name: "other", Source: "github.com/x/other", Version: "v1", Commit: "c2", Timestamp: time.Now().UTC()},
		},
	}
	if err := WriteLockFile(LockFileName, lock); err != nil {
		t.Fatal(err)
	}

	if _, err := UninstallMold(manifestPath, "github.com/x/y", "", UninstallOptions{}); err != nil {
		t.Fatalf("UninstallMold: %v", err)
	}
	loaded, _ := ReadLockFile(LockFileName)
	if loaded == nil || len(loaded.Molds) != 1 || loaded.Molds[0].Source != "github.com/x/other" {
		t.Errorf("lock entry not cleaned up: %+v", loaded)
	}
}
