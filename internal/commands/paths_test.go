package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectPaths(t *testing.T) {
	if got := projectLockPath(); got != "ailloy.lock" {
		t.Errorf("projectLockPath() = %q, want ailloy.lock", got)
	}
	if got := projectManifestPath(); got != filepath.Join(".ailloy", "installed.yaml") {
		t.Errorf("projectManifestPath() = %q", got)
	}
}

func TestGlobalPaths(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home dir: %v", err)
	}
	wantLock := filepath.Join(home, "ailloy.lock")
	gotLock := globalLockPath()
	if gotLock != wantLock {
		t.Errorf("globalLockPath() = %q, want %q", gotLock, wantLock)
	}
	if !strings.HasPrefix(gotLock, home) {
		t.Errorf("globalLockPath() = %q should be under home %q", gotLock, home)
	}
	wantManifest := filepath.Join(home, ".ailloy", "installed.yaml")
	if got := globalManifestPath(); got != wantManifest {
		t.Errorf("globalManifestPath() = %q, want %q", got, wantManifest)
	}
}

func TestLockAndManifestPathsByGlobalFlag(t *testing.T) {
	if got := lockPathFor(false); got != "ailloy.lock" {
		t.Errorf("lockPathFor(false) = %q", got)
	}
	if got := manifestPathFor(false); got != filepath.Join(".ailloy", "installed.yaml") {
		t.Errorf("manifestPathFor(false) = %q", got)
	}
	if got := lockPathFor(true); !filepath.IsAbs(got) {
		t.Errorf("lockPathFor(true) = %q, want absolute path", got)
	}
	if got := manifestPathFor(true); !filepath.IsAbs(got) {
		t.Errorf("manifestPathFor(true) = %q, want absolute path", got)
	}
}
