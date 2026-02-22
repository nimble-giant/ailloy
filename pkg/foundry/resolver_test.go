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

	tag, sha, err := highestVersion(tags, nil)
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
	_, _, err := highestVersion(map[string]string{}, nil)
	if err == nil {
		t.Fatal("expected error for empty tags")
	}
}
