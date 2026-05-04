package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nimble-giant/ailloy/pkg/foundry"
)

// TestE2E_PublishOreThenCastConsumerMold exercises the full Phase 1-11
// pipeline end-to-end on a single test:
//
//  1. Build a fake "status" ore foundry (ore.yaml + flux.schema.yaml +
//     flux.yaml; git init + tag v1.0.0). See helpers in
//     test_helpers_ore_test.go for the (documented) reason this uses local
//     paths instead of routing through ResolveWithMetadata.
//  2. Build a fake consumer mold that declares `dependencies: - ore: <ore>`
//     and has a blank that templates against the ore namespace.
//  3. Cast the mold into a project dir via the programmatic CastMold API.
//  4. Assert the ore was auto-installed under .ailloy/ores/status/.
//  5. Assert the rendered output references the ore (default `enabled: false`
//     branch is taken).
//  6. Manually record the mold in installed.yaml (CastMold only does this
//     for remote refs; for local-path molds the test seam is necessary so
//     the cascade-uninstall path has something to walk). Then trigger the
//     cascade.
//  7. Assert .ailloy/ores/status/ is gone (cascade-removed alongside the
//     mold).
func TestE2E_PublishOreThenCastConsumerMold(t *testing.T) {
	// Disabled: gitInitTagAndCommit in test_helpers_ore_test.go has been
	// observed to leak commits onto the parent worktree's branch under
	// `go test -race` during pre-push. Re-enable once the helper is hardened
	// (e.g. by passing `git -C <dir>` plus an explicit GIT_DIR/GIT_WORK_TREE
	// pair, or by replacing the local-path harness with a real foundry
	// resolver injection).
	t.Skip("disabled pending hermetic fake-foundry harness; see helper note")

	tmp := t.TempDir()

	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	// 1. Fake status-ore v1.0.0.
	oreRef, oreCleanup := buildFakeOreFoundry(t, tmp, "status", "v1.0.0")
	t.Cleanup(oreCleanup)

	// 2. Fake consumer mold declaring the ore.
	moldRef, moldCleanup := buildFakeMoldFoundryWithOreDep(t, tmp, oreRef)
	t.Cleanup(moldCleanup)

	// 3. Run cast against the mold from a fresh project dir.
	projectDir := filepath.Join(tmp, "project")
	if err := os.MkdirAll(projectDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}

	res, err := CastMold(context.Background(), moldRef, CastOptions{})
	if err != nil {
		t.Fatalf("CastMold: %v", err)
	}
	if res.MoldName != "status-consumer" {
		t.Errorf("MoldName = %q, want %q", res.MoldName, "status-consumer")
	}

	// 4. Ore was installed alongside the mold.
	oreDir := filepath.Join(projectDir, ".ailloy", "ores", "status")
	if _, err := os.Stat(filepath.Join(oreDir, "ore.yaml")); err != nil {
		t.Errorf("ore should be installed at %s: %v", oreDir, err)
	}
	if _, err := os.Stat(filepath.Join(oreDir, "flux.schema.yaml")); err != nil {
		t.Errorf("ore flux.schema.yaml should be present: %v", err)
	}

	// installed.yaml records the ore with the consumer mold listed as
	// dependent. (moldKey is "" for local-path molds, so the dependents list
	// is empty — but the entry itself must exist so the cascade walker has
	// state to act on later.)
	im, _ := foundry.ReadInstalledManifest(filepath.Join(projectDir, ".ailloy", "installed.yaml"))
	if im == nil {
		t.Fatal("installed manifest missing after cast")
	}
	oreEntry := im.FindArtifact("ore", "status")
	if oreEntry == nil {
		t.Fatalf("status ore not recorded in installed manifest; ores=%+v", im.Ores)
	}

	// 5. Rendered output references the ore namespace and produced the
	// `enabled: false` branch (the default).
	rendered, err := os.ReadFile(filepath.Join(projectDir, ".claude", "commands", "status.md"))
	if err != nil {
		t.Fatalf("rendered status blank not found: %v", err)
	}
	rs := string(rendered)
	if !strings.Contains(rs, "Status reporting disabled") {
		t.Errorf("expected disabled branch (default ore.status.enabled=false); got:\n%s", rs)
	}
	if !strings.Contains(rs, "hello") {
		t.Errorf("expected greeting=hello from ore default; got:\n%s", rs)
	}
	if strings.Contains(rs, "Status reporting enabled") {
		t.Errorf("did not expect enabled branch with default flux; got:\n%s", rs)
	}

	// 6. CastMold writes a mold entry to installed.yaml only for remote refs.
	// To exercise the cascade-uninstall path, manually upsert the mold entry
	// using a synthetic moldKey (mirroring what a remote-resolved cast would
	// have recorded), then attach that key to the ore's Dependents so the
	// cascade has something concrete to drop.
	moldKey := "github.com/test/status-consumer"
	im.UpsertEntry(foundry.InstalledEntry{
		Name:    "status-consumer",
		Source:  moldKey,
		Version: "0.1.0",
		CastAt:  time.Now().UTC(),
	})
	// Attach the mold as a dependent so the cascade GC drops the ore.
	if oreEntry := im.FindArtifact("ore", "status"); oreEntry != nil {
		oreEntry.Dependents = append(oreEntry.Dependents, moldKey)
	}
	if err := foundry.WriteInstalledManifest(filepath.Join(projectDir, ".ailloy", "installed.yaml"), im); err != nil {
		t.Fatalf("rewrite manifest with mold entry: %v", err)
	}

	// Cascade: drop the consumer mold and verify the ore is GC'd.
	if err := cascadeUninstallArtifacts(filepath.Join(projectDir, ".ailloy", "installed.yaml"), moldKey, false); err != nil {
		t.Fatalf("cascadeUninstallArtifacts: %v", err)
	}

	// 7. Ore install dir is gone; manifest entry is gone.
	if _, err := os.Stat(oreDir); !os.IsNotExist(err) {
		t.Errorf("ore install dir should be cascade-removed: %v", err)
	}
	postIM, _ := foundry.ReadInstalledManifest(filepath.Join(projectDir, ".ailloy", "installed.yaml"))
	if postIM != nil && postIM.FindArtifact("ore", "status") != nil {
		t.Errorf("ore manifest entry should be cascade-removed: %+v", postIM.Ores)
	}
}

// TestE2E_TwoMoldsShareOre_FirstUninstallKeepsOre verifies the shared-ore
// scenario end-to-end: two molds each declare the same ore, both cast, and
// uninstalling one leaves the ore in place for the surviving consumer.
//
// Like the publish-then-cast test above, this routes via local-path
// dependencies (the foundry resolver isn't easily mockable from a public
// API) but exercises the full installDeclaredDeps + cascade pipeline that
// remote-foundry casts share.
func TestE2E_TwoMoldsShareOre_FirstUninstallKeepsOre(t *testing.T) {
	// See TestE2E_PublishOreThenCastConsumerMold above for the disable
	// rationale. Same helper, same hazard.
	t.Skip("disabled pending hermetic fake-foundry harness; see helper note")

	tmp := t.TempDir()
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	oreRef, oreCleanup := buildFakeOreFoundry(t, tmp, "status", "v1.0.0")
	t.Cleanup(oreCleanup)

	// Build the first mold, then relocate it so the second helper call
	// (which always names its dir "fake-mold") doesn't clobber it.
	moldFirst, _ := buildFakeMoldFoundryWithOreDep(t, tmp, oreRef)
	moldB := filepath.Join(tmp, "fake-mold-b")
	if err := os.Rename(moldFirst, moldB); err != nil {
		t.Fatalf("relocate moldB: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(moldB) })

	moldA, cleanupA := buildFakeMoldFoundryWithOreDep(t, tmp, oreRef)
	t.Cleanup(cleanupA)

	projectDir := filepath.Join(tmp, "project")
	if err := os.MkdirAll(projectDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}

	// Cast both molds.
	if _, err := CastMold(context.Background(), moldA, CastOptions{}); err != nil {
		t.Fatalf("CastMold A: %v", err)
	}
	if _, err := CastMold(context.Background(), moldB, CastOptions{}); err != nil {
		t.Fatalf("CastMold B: %v", err)
	}

	manifestPath := filepath.Join(projectDir, ".ailloy", "installed.yaml")
	im, _ := foundry.ReadInstalledManifest(manifestPath)
	if im == nil {
		t.Fatal("installed manifest missing")
	}

	// Manually attach two distinct mold keys as dependents on the ore (cast
	// of local-path molds doesn't populate the mold table; this is the same
	// test seam used in the single-mold e2e test).
	moldKeyA := "github.com/test/consumer-a"
	moldKeyB := "github.com/test/consumer-b"
	for _, k := range []string{moldKeyA, moldKeyB} {
		im.UpsertEntry(foundry.InstalledEntry{
			Name: "status-consumer", Source: k, Version: "0.1.0", CastAt: time.Now().UTC(),
		})
	}
	if e := im.FindArtifact("ore", "status"); e != nil {
		e.Dependents = []string{moldKeyA, moldKeyB}
	}
	if err := foundry.WriteInstalledManifest(manifestPath, im); err != nil {
		t.Fatal(err)
	}

	// Uninstall mold A's cascade; ore should remain because mold B still depends on it.
	if err := cascadeUninstallArtifacts(manifestPath, moldKeyA, false); err != nil {
		t.Fatalf("cascade A: %v", err)
	}
	postIM, _ := foundry.ReadInstalledManifest(manifestPath)
	if postIM == nil || postIM.FindArtifact("ore", "status") == nil {
		t.Fatalf("ore should remain after first uninstall: %+v", postIM)
	}
	if _, err := os.Stat(filepath.Join(projectDir, ".ailloy", "ores", "status")); err != nil {
		t.Errorf("ore install dir should still exist: %v", err)
	}

	// Now uninstall mold B's cascade; ore should disappear.
	if err := cascadeUninstallArtifacts(manifestPath, moldKeyB, false); err != nil {
		t.Fatalf("cascade B: %v", err)
	}
	postIM2, _ := foundry.ReadInstalledManifest(manifestPath)
	if postIM2 != nil && postIM2.FindArtifact("ore", "status") != nil {
		t.Errorf("ore should be cascade-removed after second uninstall: %+v", postIM2.Ores)
	}
	if _, err := os.Stat(filepath.Join(projectDir, ".ailloy", "ores", "status")); !os.IsNotExist(err) {
		t.Errorf("ore install dir should be gone after second uninstall: %v", err)
	}
}
