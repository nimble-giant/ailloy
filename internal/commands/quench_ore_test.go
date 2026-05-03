package commands

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nimble-giant/ailloy/pkg/foundry"
)

func TestQuench_PinsOresAndIngots(t *testing.T) {
	tmp := t.TempDir()
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	manifest := &foundry.InstalledManifest{
		APIVersion: "v1",
		Molds:      []foundry.InstalledEntry{{Name: "m", Source: "g/m", Version: "1.0.0", Commit: "abc", CastAt: time.Now().UTC()}},
		Ingots: []foundry.ArtifactEntry{
			{Name: "ig", Source: "g/ig", Version: "1.0.0", Commit: "def", InstalledAt: time.Now().UTC(), Dependents: []string{"g/m"}},
		},
		Ores: []foundry.ArtifactEntry{
			{Name: "or", Source: "g/or-source", Version: "1.0.0", Commit: "ghi", InstalledAt: time.Now().UTC(), Alias: "aliased", Dependents: []string{"g/m"}},
		},
	}
	if err := foundry.WriteInstalledManifest(filepath.Join(".ailloy", "installed.yaml"), manifest); err != nil {
		t.Fatal(err)
	}

	if err := runQuench(quenchCmd, nil); err != nil {
		t.Fatalf("runQuench: %v", err)
	}

	lock, err := foundry.ReadLockFile(foundry.LockFileName)
	if err != nil {
		t.Fatal(err)
	}
	if lock == nil {
		t.Fatal("lock file not written")
	}
	if len(lock.Ingots) != 1 || lock.Ingots[0].Name != "ig" {
		t.Errorf("ingots: %+v", lock.Ingots)
	}
	if len(lock.Ores) != 1 || lock.Ores[0].Name != "or" || lock.Ores[0].Alias != "aliased" {
		t.Errorf("ores: %+v", lock.Ores)
	}
}

func TestQuenchVerify_DetectsOreDrift(t *testing.T) {
	t.Skip("requires --verify mode plumbing for ingots/ores — Phase 12 follow-up")
}
