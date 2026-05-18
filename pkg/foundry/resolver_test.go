package foundry

import (
	"fmt"
	"testing"
)

// mockGitRunner returns a GitRunner that returns canned output for specific
// argument patterns.
func mockGitRunner(responses map[string]string) GitRunner {
	return func(args ...string) ([]byte, error) {
		key := fmt.Sprintf("%v", args)
		if out, ok := responses[key]; ok {
			return []byte(out), nil
		}
		return nil, fmt.Errorf("unexpected git call: %v", args)
	}
}

const lsRemoteTagsOutput = `abc1234567890000000000000000000000000001	refs/tags/v0.1.0
abc1234567890000000000000000000000000002	refs/tags/v0.2.0
abc1234567890000000000000000000000000003	refs/tags/v1.0.0
abc1234567890000000000000000000000000003	refs/tags/v1.0.0^{}
abc1234567890000000000000000000000000004	refs/tags/v1.1.0
abc1234567890000000000000000000000000005	refs/tags/v1.2.3
abc1234567890000000000000000000000000006	refs/tags/v2.0.0
deadbeef00000000000000000000000000000000	refs/tags/not-semver
`

func TestParseLsRemoteTags(t *testing.T) {
	tags, err := parseLsRemoteTags(lsRemoteTagsOutput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// v1.0.0 should use the deref SHA (^{} overrides)
	if sha := tags["v1.0.0"]; sha != "abc1234567890000000000000000000000000003" {
		t.Errorf("v1.0.0 SHA = %q, want deref SHA", sha)
	}

	// non-semver tag should be excluded
	if _, ok := tags["not-semver"]; ok {
		t.Error("non-semver tag should be excluded")
	}

	// check total count (v0.1.0, v0.2.0, v1.0.0, v1.1.0, v1.2.3, v2.0.0)
	if len(tags) != 6 {
		t.Errorf("expected 6 tags, got %d: %v", len(tags), tags)
	}
}

func TestResolveVersion_Latest(t *testing.T) {
	git := mockGitRunner(map[string]string{
		"[ls-remote --tags https://github.com/owner/repo.git]": lsRemoteTagsOutput,
	})

	ref := &Reference{Host: "github.com", Owner: "owner", Repo: "repo", Type: Latest}
	resolved, err := ResolveVersion(ref, git)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Tag != "v2.0.0" {
		t.Errorf("Tag = %q, want v2.0.0", resolved.Tag)
	}
}

func TestResolveVersion_Exact(t *testing.T) {
	git := mockGitRunner(map[string]string{
		"[ls-remote --tags https://github.com/owner/repo.git]": lsRemoteTagsOutput,
	})

	ref := &Reference{Host: "github.com", Owner: "owner", Repo: "repo", Version: "v1.1.0", Type: Exact}
	resolved, err := ResolveVersion(ref, git)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Tag != "v1.1.0" {
		t.Errorf("Tag = %q, want v1.1.0", resolved.Tag)
	}
	if resolved.Commit != "abc1234567890000000000000000000000000004" {
		t.Errorf("Commit = %q, want abc...004", resolved.Commit)
	}
}

func TestResolveVersion_Exact_WithoutVPrefix(t *testing.T) {
	git := mockGitRunner(map[string]string{
		"[ls-remote --tags https://github.com/owner/repo.git]": lsRemoteTagsOutput,
	})

	ref := &Reference{Host: "github.com", Owner: "owner", Repo: "repo", Version: "1.1.0", Type: Exact}
	resolved, err := ResolveVersion(ref, git)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Tag != "v1.1.0" {
		t.Errorf("Tag = %q, want v1.1.0", resolved.Tag)
	}
}

func TestResolveVersion_Exact_NotFound(t *testing.T) {
	git := mockGitRunner(map[string]string{
		"[ls-remote --tags https://github.com/owner/repo.git]": lsRemoteTagsOutput,
	})

	ref := &Reference{Host: "github.com", Owner: "owner", Repo: "repo", Version: "v9.9.9", Type: Exact}
	_, err := ResolveVersion(ref, git)
	if err == nil {
		t.Fatal("expected error for missing tag")
	}
}

func TestResolveVersion_Constraint_Caret(t *testing.T) {
	git := mockGitRunner(map[string]string{
		"[ls-remote --tags https://github.com/owner/repo.git]": lsRemoteTagsOutput,
	})

	ref := &Reference{Host: "github.com", Owner: "owner", Repo: "repo", Version: "^1.0.0", Type: Constraint}
	resolved, err := ResolveVersion(ref, git)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// ^1.0.0 matches >=1.0.0 <2.0.0, highest is v1.2.3
	if resolved.Tag != "v1.2.3" {
		t.Errorf("Tag = %q, want v1.2.3", resolved.Tag)
	}
}

func TestResolveVersion_Constraint_Tilde(t *testing.T) {
	git := mockGitRunner(map[string]string{
		"[ls-remote --tags https://github.com/owner/repo.git]": lsRemoteTagsOutput,
	})

	ref := &Reference{Host: "github.com", Owner: "owner", Repo: "repo", Version: "~1.0.0", Type: Constraint}
	resolved, err := ResolveVersion(ref, git)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// ~1.0.0 matches >=1.0.0 <1.1.0, only v1.0.0
	if resolved.Tag != "v1.0.0" {
		t.Errorf("Tag = %q, want v1.0.0", resolved.Tag)
	}
}

func TestResolveVersion_Constraint_NoMatch(t *testing.T) {
	git := mockGitRunner(map[string]string{
		"[ls-remote --tags https://github.com/owner/repo.git]": lsRemoteTagsOutput,
	})

	ref := &Reference{Host: "github.com", Owner: "owner", Repo: "repo", Version: "^3.0.0", Type: Constraint}
	_, err := ResolveVersion(ref, git)
	if err == nil {
		t.Fatal("expected error for unmatched constraint")
	}
}

func TestResolveVersion_Branch(t *testing.T) {
	git := mockGitRunner(map[string]string{
		"[ls-remote https://github.com/owner/repo.git refs/heads/main]": "abc123def456\trefs/heads/main\n",
	})

	ref := &Reference{Host: "github.com", Owner: "owner", Repo: "repo", Version: "main", Type: Branch}
	resolved, err := ResolveVersion(ref, git)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Tag != "main" {
		t.Errorf("Tag = %q, want main", resolved.Tag)
	}
	if resolved.Commit != "abc123def456" {
		t.Errorf("Commit = %q, want abc123def456", resolved.Commit)
	}
}

func TestResolveVersion_Branch_NotFound(t *testing.T) {
	git := mockGitRunner(map[string]string{
		"[ls-remote https://github.com/owner/repo.git refs/heads/nonexistent]": "",
	})

	ref := &Reference{Host: "github.com", Owner: "owner", Repo: "repo", Version: "nonexistent", Type: Branch}
	_, err := ResolveVersion(ref, git)
	if err == nil {
		t.Fatal("expected error for missing branch")
	}
}

func TestResolveVersion_SHA(t *testing.T) {
	git := mockGitRunner(nil)

	ref := &Reference{Host: "github.com", Owner: "owner", Repo: "repo", Version: "abc1234", Type: SHA}
	resolved, err := ResolveVersion(ref, git)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Tag != "abc1234" {
		t.Errorf("Tag = %q, want abc1234", resolved.Tag)
	}
	if resolved.Commit != "abc1234" {
		t.Errorf("Commit = %q, want abc1234", resolved.Commit)
	}
}

func TestHighestVersion(t *testing.T) {
	tags := map[string]string{
		"v1.0.0": "sha1",
		"v1.2.0": "sha2",
		"v1.1.0": "sha3",
		"v0.9.0": "sha4",
	}

	tag, sha, _, err := highestVersion(tags, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tag != "v1.2.0" {
		t.Errorf("tag = %q, want v1.2.0", tag)
	}
	if sha != "sha2" {
		t.Errorf("sha = %q, want sha2", sha)
	}
}

func TestHighestVersion_Empty(t *testing.T) {
	_, _, _, err := highestVersion(map[string]string{}, nil, nil)
	if err == nil {
		t.Fatal("expected error for empty tags")
	}
}

// monorepoTagsOutput models a foundry repo (e.g. kriscoleman/replicated-foundry)
// that uses per-mold prefixed tags (`<mold>-v<semver>`) for current releases
// while still carrying older plain `v*` tags from before the split.
const monorepoTagsOutput = `1111111111111111111111111111111111111111	refs/tags/v0.1.0
2222222222222222222222222222222222222222	refs/tags/v0.2.0
3333333333333333333333333333333333333333	refs/tags/v0.3.0
aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa	refs/tags/wiki-v0.4.0
bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb	refs/tags/docs-v0.4.0
cccccccccccccccccccccccccccccccccccccccc	refs/tags/launch-v0.4.0
dddddddddddddddddddddddddddddddddddddddd	refs/tags/wiki-v0.5.0
`

func TestResolveVersion_Latest_PrefixedMonorepoTag(t *testing.T) {
	git := mockGitRunner(map[string]string{
		"[ls-remote --tags https://github.com/kriscoleman/replicated-foundry.git]": monorepoTagsOutput,
	})

	ref := &Reference{
		Host:    "github.com",
		Owner:   "kriscoleman",
		Repo:    "replicated-foundry",
		Subpath: "molds/wiki",
		Type:    Latest,
	}
	resolved, err := ResolveVersion(ref, git)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Tag != "wiki-v0.5.0" {
		t.Errorf("Tag = %q, want wiki-v0.5.0 (highest prefixed match for subpath)", resolved.Tag)
	}
	if resolved.Commit != "dddddddddddddddddddddddddddddddddddddddd" {
		t.Errorf("Commit = %q, want dddd... (wiki-v0.5.0 SHA)", resolved.Commit)
	}
}

func TestResolveVersion_Latest_PrefixedFallsBackToPlain(t *testing.T) {
	git := mockGitRunner(map[string]string{
		"[ls-remote --tags https://github.com/kriscoleman/replicated-foundry.git]": monorepoTagsOutput,
	})

	// Subpath has no matching prefixed tags → fall back to plain v*.
	ref := &Reference{
		Host:    "github.com",
		Owner:   "kriscoleman",
		Repo:    "replicated-foundry",
		Subpath: "molds/nonexistent",
		Type:    Latest,
	}
	resolved, err := ResolveVersion(ref, git)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Tag != "v0.3.0" {
		t.Errorf("Tag = %q, want v0.3.0 (fallback to plain semver)", resolved.Tag)
	}
}

func TestResolveVersion_Latest_NoSubpathIgnoresPrefixed(t *testing.T) {
	git := mockGitRunner(map[string]string{
		"[ls-remote --tags https://github.com/kriscoleman/replicated-foundry.git]": monorepoTagsOutput,
	})

	// No subpath → only plain v* tags are eligible (today's behaviour).
	ref := &Reference{
		Host:  "github.com",
		Owner: "kriscoleman",
		Repo:  "replicated-foundry",
		Type:  Latest,
	}
	resolved, err := ResolveVersion(ref, git)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Tag != "v0.3.0" {
		t.Errorf("Tag = %q, want v0.3.0 (no subpath → plain only)", resolved.Tag)
	}
}

func TestResolveVersion_Exact_PrefixedMonorepoTag(t *testing.T) {
	git := mockGitRunner(map[string]string{
		"[ls-remote --tags https://github.com/kriscoleman/replicated-foundry.git]": monorepoTagsOutput,
	})

	// User says @v0.4.0 with a wiki subpath → resolve to wiki-v0.4.0.
	ref := &Reference{
		Host:    "github.com",
		Owner:   "kriscoleman",
		Repo:    "replicated-foundry",
		Subpath: "molds/wiki",
		Version: "v0.4.0",
		Type:    Exact,
	}
	resolved, err := ResolveVersion(ref, git)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Tag != "wiki-v0.4.0" {
		t.Errorf("Tag = %q, want wiki-v0.4.0", resolved.Tag)
	}
	if resolved.Commit != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Errorf("Commit = %q, want aaaa... (wiki-v0.4.0 SHA)", resolved.Commit)
	}
}

func TestResolveVersion_Constraint_PrefixedMonorepoTag(t *testing.T) {
	git := mockGitRunner(map[string]string{
		"[ls-remote --tags https://github.com/kriscoleman/replicated-foundry.git]": monorepoTagsOutput,
	})

	ref := &Reference{
		Host:    "github.com",
		Owner:   "kriscoleman",
		Repo:    "replicated-foundry",
		Subpath: "molds/wiki",
		Version: ">=0.4.0",
		Type:    Constraint,
	}
	resolved, err := ResolveVersion(ref, git)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// >=0.4.0 against wiki-prefixed semvers (0.4.0, 0.5.0) → 0.5.0.
	if resolved.Tag != "wiki-v0.5.0" {
		t.Errorf("Tag = %q, want wiki-v0.5.0", resolved.Tag)
	}
}

func TestReference_ReleasePrefix(t *testing.T) {
	cases := []struct {
		subpath string
		want    string
	}{
		{"", ""},
		{"molds/wiki", "wiki"},
		{"molds/launch/", "launch"},
		{"wiki", "wiki"},
		{"a/b/c", "c"},
	}
	for _, tc := range cases {
		ref := &Reference{Subpath: tc.subpath}
		if got := ref.ReleasePrefix(); got != tc.want {
			t.Errorf("ReleasePrefix(subpath=%q) = %q, want %q", tc.subpath, got, tc.want)
		}
	}
}

// trainTagsOutput models a release-train monorepo: every mold is tagged with
// the foundry's shared train version on each release, so `launch-v*` tags do
// not encode the launch mold's own version (which lives in mold.yaml).
const trainTagsOutput = `1111111111111111111111111111111111111111	refs/tags/launch-v0.5.0
2222222222222222222222222222222222222222	refs/tags/launch-v0.6.0
3333333333333333333333333333333333333333	refs/tags/launch-v0.7.0
4444444444444444444444444444444444444444	refs/tags/launch-v0.7.1
5555555555555555555555555555555555555555	refs/tags/ce-v0.7.1
`

// launchMoldVersions is a MoldVersionReader for the launch mold: it exists
// from launch-v0.6.0 onward, always declaring version 0.2.1. launch-v0.5.0
// predates the mold and ce-v0.7.1 carries a different mold entirely.
func launchMoldVersions(tag string) (string, bool) {
	switch tag {
	case "launch-v0.6.0", "launch-v0.7.0", "launch-v0.7.1":
		return "0.2.1", true
	default:
		return "", false // mold manifest absent at this tag
	}
}

func trainGit(t *testing.T) GitRunner {
	t.Helper()
	return mockGitRunner(map[string]string{
		"[ls-remote --tags https://github.com/replicated-collab/foundry.git]": trainTagsOutput,
	})
}

func launchRef(version string, typ RefType) *Reference {
	return &Reference{
		Host: "github.com", Owner: "replicated-collab", Repo: "foundry",
		Subpath: "molds/launch", Version: version, Type: typ,
	}
}

// TestResolveConstraint_MoldVersion is the issue #228 scenario: `^0.2.0` is
// matched against the launch mold's declared version (0.2.1), not the version
// in the tag name. Three tags carry mold version 0.2.1 — the newest train tag
// wins the tie-break.
func TestResolveConstraint_MoldVersion(t *testing.T) {
	ref := launchRef("^0.2.0", Constraint)
	resolved, err := ResolveVersionWithMoldReader(ref, trainGit(t), launchMoldVersions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Tag != "launch-v0.7.1" {
		t.Errorf("Tag = %q, want launch-v0.7.1", resolved.Tag)
	}
	if resolved.MoldVersion != "0.2.1" {
		t.Errorf("MoldVersion = %q, want 0.2.1", resolved.MoldVersion)
	}
}

func TestResolveConstraint_MoldVersion_NoMatch(t *testing.T) {
	ref := launchRef("^0.3.0", Constraint)
	if _, err := ResolveVersionWithMoldReader(ref, trainGit(t), launchMoldVersions); err == nil {
		t.Fatal("expected error: no mold version satisfies ^0.3.0")
	}
}

func TestResolveLatest_MoldVersion(t *testing.T) {
	ref := launchRef("", Latest)
	resolved, err := ResolveVersionWithMoldReader(ref, trainGit(t), launchMoldVersions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// All present tags share mold version 0.2.1; the newest train tag wins.
	if resolved.Tag != "launch-v0.7.1" {
		t.Errorf("Tag = %q, want launch-v0.7.1", resolved.Tag)
	}
}

func TestResolveExact_ByMoldVersion(t *testing.T) {
	ref := launchRef("0.2.1", Exact)
	resolved, err := ResolveVersionWithMoldReader(ref, trainGit(t), launchMoldVersions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Tag != "launch-v0.7.1" {
		t.Errorf("Tag = %q, want launch-v0.7.1 (mold version 0.2.1)", resolved.Tag)
	}
	if resolved.MoldVersion != "0.2.1" {
		t.Errorf("MoldVersion = %q, want 0.2.1", resolved.MoldVersion)
	}
}

func TestResolveExact_ByMoldVersion_NoMatch(t *testing.T) {
	ref := launchRef("9.9.9", Exact)
	if _, err := ResolveVersionWithMoldReader(ref, trainGit(t), launchMoldVersions); err == nil {
		t.Fatal("expected error: no tag declares mold version 9.9.9")
	}
}

// TestResolveConstraint_FallbackEmptyMoldVersion verifies that when a mold
// manifest exists but declares no version, the resolver falls back to ranking
// by the tag-embedded semver.
func TestResolveConstraint_FallbackEmptyMoldVersion(t *testing.T) {
	emptyReader := func(tag string) (string, bool) { return "", true }
	ref := launchRef(">=0.6.0", Constraint)
	resolved, err := ResolveVersionWithMoldReader(ref, trainGit(t), emptyReader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Tag != "launch-v0.7.1" {
		t.Errorf("Tag = %q, want launch-v0.7.1 (ranked by tag-embedded version)", resolved.Tag)
	}
}

// TestResolveConstraint_NilReaderUsesTagVersion confirms the nil-reader path
// (plain ResolveVersion) is unchanged: constraints match the tag-embedded
// version, so a mold-version constraint like ^0.2.0 finds nothing here.
func TestResolveConstraint_NilReaderUsesTagVersion(t *testing.T) {
	ref := launchRef("^0.2.0", Constraint)
	if _, err := ResolveVersion(ref, trainGit(t)); err == nil {
		t.Fatal("expected error: tag-embedded versions are 0.5-0.7, not ^0.2.0")
	}
}
