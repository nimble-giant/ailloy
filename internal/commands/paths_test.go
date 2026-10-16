package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Project bookkeeping lives in the project's state dir under
// ~/.ailloy/projects/, never in the project tree.
func TestProjectPaths(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home dir: %v", err)
	}
	stateRoot := filepath.Join(home, ".ailloy", "projects") + string(filepath.Separator)
	if got := projectLockPath(); got != filepath.Join(projectStateDir(), "ailloy.lock") || !strings.HasPrefix(got, stateRoot) {
		t.Errorf("projectLockPath() = %q, want <state dir>/ailloy.lock under %q", got, stateRoot)
	}
	if got := projectManifestPath(); got != filepath.Join(projectStateDir(), "installed.yaml") || !strings.HasPrefix(got, stateRoot) {
		t.Errorf("projectManifestPath() = %q, want <state dir>/installed.yaml under %q", got, stateRoot)
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
	if got := lockPathFor(false); got != projectLockPath() {
		t.Errorf("lockPathFor(false) = %q", got)
	}
	if got := manifestPathFor(false); got != projectManifestPath() {
		t.Errorf("manifestPathFor(false) = %q", got)
	}
	if _, err := os.UserHomeDir(); err != nil {
		t.Skipf("no home dir, skipping global-flag assertions: %v", err)
	}
	if got := lockPathFor(true); !filepath.IsAbs(got) {
		t.Errorf("lockPathFor(true) = %q, want absolute path", got)
	}
	if got := manifestPathFor(true); !filepath.IsAbs(got) {
		t.Errorf("manifestPathFor(true) = %q, want absolute path", got)
	}
}
