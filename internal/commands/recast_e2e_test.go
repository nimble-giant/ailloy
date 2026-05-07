package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/foundry"
)

// recastE2EEnv wires a synthetic remote into a self-contained sandbox so the
// e2e tests can exercise `cast` and `recast` without ever touching the user's
// real ~/.ailloy cache or hitting the network.
//
// The sandbox is built on two git mechanisms:
//
//  1. A bare git repository created on disk plays the role of the "remote".
//     Tags pushed to it (v1.0.0, v2.0.0, ...) become the versions ailloy will
//     resolve against.
//  2. A scratch HOME directory + GIT_CONFIG_GLOBAL pointing at a temp gitconfig
//     with `url.<file://...>.insteadOf = https://<host>/<owner>/<repo>` makes
//     every `git ls-remote` / `git clone` / `git fetch` ailloy issues against
//     the synthetic CloneURL transparently redirect to the local bare repo.
//
// Why this shape: the foundry package only treats `host/owner/repo`-style refs
// as remote (see foundry.IsRemoteReference), and ParseReference + CloneURL
// hardcode `https://<host>/<owner>/<repo>.git`. There is no `file://` shortcut
// in the parser, and recast specifically requires entry.Source to be in
// `host/owner/repo` form (referenceFromInstalledEntry in quench.go). The
// insteadOf trick is the cleanest way to satisfy both constraints without
// patching the parser or pre-populating the cache by hand.
type recastE2EEnv struct {
	bin       string // ailloy binary path
	repoDir   string // working tree of the synthetic remote
	bareDir   string // bare git repo ailloy will clone from (via insteadOf)
	homeDir   string // sandboxed HOME so ~/.ailloy/cache stays out of $HOME
	gitConfig string // path to the temp gitconfig with insteadOf
	host      string // synthetic host segment, e.g. "localhost.test"
	owner     string // synthetic owner segment, e.g. "ailloy-test"
	repo      string // synthetic repo segment, e.g. "recast-fixture"
}

// refString returns the canonical ailloy ref for the synthetic repo.
func (e *recastE2EEnv) refString() string {
	return fmt.Sprintf("%s/%s/%s", e.host, e.owner, e.repo)
}

// extraEnv returns the env additions every spawned ailloy process needs so
// HOME points at the sandbox and git applies the insteadOf substitution.
func (e *recastE2EEnv) extraEnv() []string {
	return []string{
		"HOME=" + e.homeDir,
		"GIT_CONFIG_GLOBAL=" + e.gitConfig,
		// Defang any user/system gitconfig that might inject conflicting
		// insteadOf rules or credential helpers.
		"GIT_CONFIG_NOSYSTEM=1",
	}
}

// sandboxEnv returns os.Environ() with all GIT_* variables stripped, then the
// sandbox additions appended. This matters when the test runs inside a git
// hook (e.g. lefthook's pre-push) because git sets GIT_DIR / GIT_INDEX_FILE /
// GIT_WORK_TREE in the hook environment, and any subprocess that inherits
// those would operate on the OUTER repo regardless of cmd.Dir — causing the
// test's git operations to mutate the developer's repo instead of the
// per-test sandbox.
func (e *recastE2EEnv) sandboxEnv() []string {
	out := make([]string, 0, len(os.Environ()))
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "GIT_") {
			continue
		}
		out = append(out, kv)
	}
	return append(out, e.extraEnv()...)
}

// run invokes the ailloy binary with the sandbox env applied. dir is the
// working directory (typically the project under test).
func (e *recastE2EEnv) run(t *testing.T, dir string, args ...string) ([]byte, error) {
	t.Helper()
	cmd := exec.Command(e.bin, args...)
	cmd.Dir = dir
	cmd.Env = e.sandboxEnv()
	return cmd.CombinedOutput()
}

// gitInRepo runs a git command inside the synthetic remote's working tree
// (e.repoDir) with the sandbox gitconfig applied. Used to add commits and
// tags that ailloy will later resolve.
func (e *recastE2EEnv) gitInRepo(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = e.repoDir
	cmd.Env = e.sandboxEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// writeMoldFiles populates the synthetic remote's working tree with a minimal
// mold whose rendered README echoes the value of flux variable `foo`.
// versionMarker is appended to the template so v1 vs v2 renders are
// distinguishable.
func (e *recastE2EEnv) writeMoldFiles(t *testing.T, version, versionMarker string) {
	t.Helper()

	moldYAML := fmt.Sprintf("apiVersion: v1\nkind: Mold\nname: e2e-recast\nversion: %s\n", version)
	if err := os.WriteFile(filepath.Join(e.repoDir, "mold.yaml"), []byte(moldYAML), 0644); err != nil {
		t.Fatal(err)
	}
	// flux.yaml maps the README template as an output. Note: keys under
	// `output:` are SOURCE files in the mold; they are processed as Go
	// templates by default.
	fluxYAML := "output:\n  README.md: README.md\n"
	if err := os.WriteFile(filepath.Join(e.repoDir, "flux.yaml"), []byte(fluxYAML), 0644); err != nil {
		t.Fatal(err)
	}
	tmpl := fmt.Sprintf("foo={{ .foo }}%s\n", versionMarker)
	if err := os.WriteFile(filepath.Join(e.repoDir, "README.md"), []byte(tmpl), 0644); err != nil {
		t.Fatal(err)
	}
}

// commitAndTag stages all changes in the synthetic remote, commits them with
// the given message, tags the commit, and pushes the commit + tag to the bare
// repo so ailloy's `git ls-remote --tags` against the synthetic CloneURL
// (redirected via insteadOf) sees the new version.
func (e *recastE2EEnv) commitAndTag(t *testing.T, message, tag string) {
	t.Helper()
	e.gitInRepo(t, "add", ".")
	e.gitInRepo(t, "commit", "-m", message)
	e.gitInRepo(t, "tag", tag)
	// Push to the bare repo so future ls-remote/fetch calls against the
	// synthetic remote URL find the new tag.
	e.gitInRepo(t, "push", "origin", "HEAD:refs/heads/main", "--tags", "--force")
}

// setupRecastE2EEnv builds the ailloy binary, prepares the synthetic remote
// with an initial v1.0.0 mold, and returns an env ready for cast/recast.
func setupRecastE2EEnv(t *testing.T) *recastE2EEnv {
	t.Helper()

	bin := buildAilloyBinary(t)

	root := t.TempDir()
	env := &recastE2EEnv{
		bin:       bin,
		repoDir:   filepath.Join(root, "remote-worktree"),
		bareDir:   filepath.Join(root, "remote.git"),
		homeDir:   filepath.Join(root, "home"),
		gitConfig: filepath.Join(root, "gitconfig"),
		host:      "localhost.test",
		owner:     "ailloy-test",
		repo:      "recast-fixture",
	}

	for _, dir := range []string{env.repoDir, env.bareDir, env.homeDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Write the gitconfig that makes git redirect the synthetic CloneURL
	// (https://localhost.test/ailloy-test/recast-fixture[.git]) to the local
	// bare repo. Two insteadOf entries handle both the with-.git and
	// without-.git URL shapes that ailloy may construct.
	cloneURL := fmt.Sprintf("https://%s/%s/%s", env.host, env.owner, env.repo)
	gitconfigContents := fmt.Sprintf(
		"[url \"file://%s\"]\n\tinsteadOf = %s.git\n[url \"file://%s\"]\n\tinsteadOf = %s\n",
		env.bareDir, cloneURL,
		env.bareDir, cloneURL,
	)
	if err := os.WriteFile(env.gitConfig, []byte(gitconfigContents), 0644); err != nil {
		t.Fatal(err)
	}

	// Initialize the bare "remote" repo (where tags will live).
	bareInit := exec.Command("git", "init", "--bare", "--initial-branch=main", env.bareDir)
	bareInit.Env = env.sandboxEnv()
	if out, err := bareInit.CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v\n%s", err, out)
	}

	// Initialize the working tree, point its origin at the bare repo, and
	// configure a committer identity scoped to this repo (so the test does
	// not depend on the developer's global git identity).
	wtInit := exec.Command("git", "init", "--initial-branch=main", env.repoDir)
	wtInit.Env = env.sandboxEnv()
	if out, err := wtInit.CombinedOutput(); err != nil {
		t.Fatalf("git init worktree: %v\n%s", err, out)
	}
	env.gitInRepo(t, "config", "user.email", "test@example.com")
	env.gitInRepo(t, "config", "user.name", "ailloy-test")
	env.gitInRepo(t, "config", "commit.gpgsign", "false")
	env.gitInRepo(t, "remote", "add", "origin", env.bareDir)

	// Seed the v1 mold and publish it.
	env.writeMoldFiles(t, "1.0.0", "")
	env.commitAndTag(t, "initial", "v1.0.0")

	return env
}

// TestE2E_Recast_RerenderWithRecordedOptions is the happy-path test: cast
// with --set, bump the synthetic remote to v2, run `recast` with no flags and
// assert files reflect the new version while the recorded --set value is
// still applied. Then `recast --set foo=updated` to confirm CLI overrides
// replace the recorded entry rather than appending.
func TestE2E_Recast_RerenderWithRecordedOptions(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e binary build is slow; skipping in -short mode")
	}

	env := setupRecastE2EEnv(t)
	project := t.TempDir()

	// Initial cast pinned to v1.0.0 with a recorded --set override.
	refV1 := env.refString() + "@v1.0.0"
	if out, err := env.run(t, project, "cast", refV1, "--set", "foo=cli-value"); err != nil {
		t.Fatalf("cast failed: %v\n%s", err, out)
	}

	got, _ := os.ReadFile(filepath.Join(project, "README.md"))
	if !strings.Contains(string(got), "foo=cli-value") {
		t.Fatalf("after initial cast README.md did not reflect --set value:\n%s", got)
	}

	// Manifest must record the --set override so a no-flag recast can replay it.
	manifest, err := foundry.ReadInstalledManifest(filepath.Join(project, ".ailloy", "installed.yaml"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if len(manifest.Molds) != 1 || manifest.Molds[0].CastOptions == nil {
		t.Fatalf("expected manifest to record cast options, got %+v", manifest)
	}
	if got := manifest.Molds[0].CastOptions.SetOverrides; len(got) != 1 || got[0] != "foo=cli-value" {
		t.Errorf("recorded SetOverrides = %v, want [foo=cli-value]", got)
	}

	// Publish a v2 commit that materially changes the rendered README so we
	// can detect that recast actually re-rendered against the new tag.
	env.writeMoldFiles(t, "2.0.0", " (v2)")
	env.commitAndTag(t, "v2", "v2.0.0")

	// Recast with no flags. Because the manifest entry is unversioned (Latest)
	// recast should resolve to v2.0.0, re-render, and replay the recorded
	// --set foo=cli-value.
	if out, err := env.run(t, project, "recast"); err != nil {
		t.Fatalf("recast failed: %v\n%s", err, out)
	}

	got, _ = os.ReadFile(filepath.Join(project, "README.md"))
	gs := string(got)
	if !strings.Contains(gs, "foo=cli-value") {
		t.Errorf("expected recorded --set foo=cli-value to still apply, got:\n%s", gs)
	}
	if !strings.Contains(gs, "(v2)") {
		t.Errorf("expected recast to bump the rendered template to v2, got:\n%s", gs)
	}

	// recast --set foo=updated should replace the recorded entry (same key),
	// not append a duplicate.
	if out, err := env.run(t, project, "recast", "--set", "foo=updated"); err != nil {
		t.Fatalf("recast --set failed: %v\n%s", err, out)
	}
	got, _ = os.ReadFile(filepath.Join(project, "README.md"))
	if !strings.Contains(string(got), "foo=updated") {
		t.Errorf("expected recast --set to update render, got:\n%s", got)
	}

	manifest, err = foundry.ReadInstalledManifest(filepath.Join(project, ".ailloy", "installed.yaml"))
	if err != nil {
		t.Fatalf("re-read manifest: %v", err)
	}
	if got := manifest.Molds[0].CastOptions.SetOverrides; len(got) != 1 || got[0] != "foo=updated" {
		t.Errorf("expected SetOverrides=[foo=updated] (replaced, not appended), got %v", got)
	}
}

// TestE2E_Recast_DryRunWritesNothing proves --dry-run is non-mutating: even
// with CLI overrides supplied that would otherwise change the render, neither
// the rendered files nor the installed manifest may change.
func TestE2E_Recast_DryRunWritesNothing(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e binary build is slow; skipping in -short mode")
	}

	env := setupRecastE2EEnv(t)
	project := t.TempDir()

	refV1 := env.refString() + "@v1.0.0"
	if out, err := env.run(t, project, "cast", refV1); err != nil {
		t.Fatalf("cast failed: %v\n%s", err, out)
	}

	// Snapshot README and manifest contents so we can assert byte-for-byte
	// equality after the dry run.
	readmeBefore, err := os.ReadFile(filepath.Join(project, "README.md"))
	if err != nil {
		t.Fatalf("read README before: %v", err)
	}
	manifestPath := filepath.Join(project, ".ailloy", "installed.yaml")
	manBefore, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest before: %v", err)
	}

	// Publish v2 so a real recast WOULD pick it up; --dry-run must still not
	// mutate disk even though the version moved AND a CLI override was given.
	env.writeMoldFiles(t, "2.0.0", " (v2)")
	env.commitAndTag(t, "v2", "v2.0.0")

	if out, err := env.run(t, project, "recast", "--dry-run", "--set", "foo=updated"); err != nil {
		t.Fatalf("recast --dry-run failed: %v\n%s", err, out)
	}

	readmeAfter, _ := os.ReadFile(filepath.Join(project, "README.md"))
	manAfter, _ := os.ReadFile(manifestPath)
	if string(readmeBefore) != string(readmeAfter) {
		t.Errorf("README.md changed under --dry-run\nbefore:\n%s\nafter:\n%s", readmeBefore, readmeAfter)
	}
	if string(manBefore) != string(manAfter) {
		t.Errorf("installed.yaml changed under --dry-run\nbefore:\n%s\nafter:\n%s", manBefore, manAfter)
	}
}

// TestE2E_Recast_PartialFailureExitsNonZero is intentionally stubbed.
//
// The happy-path harness above uses a single synthetic remote per test; to
// prove "one mold fails, others still re-render, exit code is non-zero" we
// need either two remotes (one healthy, one whose ls-remote/clone is wired to
// fail deterministically) or a way to inject a transient failure mid-loop.
// Both are bigger than a single test-file change and orthogonal to the ref-
// scheme question this task was scoped around — so leaving as a marker for a
// follow-up rather than smuggling a flaky network failure into CI.
func TestE2E_Recast_PartialFailureExitsNonZero(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e binary build is slow; skipping in -short mode")
	}
	t.Skip("TODO: requires multi-remote sandbox; tracked as follow-up to the recast e2e harness")
}
