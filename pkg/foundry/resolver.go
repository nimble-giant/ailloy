package foundry

import (
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// GitRunner executes a git command and returns its combined output.
// It is injectable for testing.
type GitRunner func(args ...string) ([]byte, error)

// DefaultGitRunner returns a GitRunner that shells out to git.
func DefaultGitRunner() GitRunner {
	return func(args ...string) ([]byte, error) {
		cmd := exec.Command("git", args...) //#nosec G204 -- args are constructed internally, not user-supplied
		return cmd.CombinedOutput()
	}
}

// ResolvedVersion holds the resolved tag and commit SHA for a reference.
type ResolvedVersion struct {
	Tag    string // semver tag (e.g. "v1.2.3"), or branch name for branch pins
	Commit string // full commit SHA
}

// tagRef matches a line from git ls-remote --tags, e.g.:
// abc123\trefs/tags/v1.0.0
// abc123\trefs/tags/v1.0.0^{}
var tagRef = regexp.MustCompile(`^([0-9a-f]+)\trefs/tags/(.+)$`)

// prefixedTagRe matches monorepo-style per-component tags like `wiki-v0.4.0`
// or `my-mold-1.2.3-rc1`. Group 1 is the prefix (component name); group 2 is
// the semver remainder (with or without leading `v`).
var prefixedTagRe = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9_-]*)-v?(\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.+-]+)?)$`)

// parseSemverTag classifies a tag as plain semver (`v1.2.3`, `1.2.3`) or
// prefixed semver (`wiki-v0.4.0`). Returns the prefix ("" for plain), the
// normalised semver string (no leading v), and ok=true on success.
func parseSemverTag(tag string) (prefix, normalized string, ok bool) {
	plain := strings.TrimPrefix(tag, "v")
	if _, err := semver.NewVersion(plain); err == nil {
		return "", plain, true
	}
	if m := prefixedTagRe.FindStringSubmatch(tag); m != nil {
		if _, err := semver.NewVersion(m[2]); err == nil {
			return m[1], m[2], true
		}
	}
	return "", "", false
}

// ResolveVersion resolves a Reference to a concrete tag + commit SHA using
// git ls-remote. It does not require a local clone.
func ResolveVersion(ref *Reference, git GitRunner) (*ResolvedVersion, error) {
	switch ref.Type {
	case Latest:
		return resolveLatest(ref, git)
	case Exact:
		return resolveExact(ref, git)
	case Constraint:
		return resolveConstraint(ref, git)
	case Branch:
		return resolveBranch(ref, git)
	case SHA:
		return &ResolvedVersion{Tag: ref.Version, Commit: ref.Version}, nil
	default:
		return nil, fmt.Errorf("unsupported ref type: %v", ref.Type)
	}
}

// remoteTags fetches all semver tags from the remote and returns a map of
// version → commit SHA. Annotated tags (^{}) override lightweight tag SHAs.
func remoteTags(url string, git GitRunner) (map[string]string, error) {
	out, err := git("ls-remote", "--tags", url)
	if err != nil {
		return nil, fmt.Errorf("git ls-remote --tags %s: %w\n%s", url, err, out)
	}
	return parseLsRemoteTags(string(out))
}

// parseLsRemoteTags parses the output of git ls-remote --tags into a map of
// raw tag name → commit SHA. Includes both plain (`v1.2.3`) and monorepo-
// prefixed (`wiki-v0.4.0`) semver tags. Non-semver tags are excluded.
func parseLsRemoteTags(output string) (map[string]string, error) {
	tags := make(map[string]string)

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := tagRef.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		sha := m[1]
		tagName := m[2]

		// Annotated tags have a ^{} dereference line pointing to the actual commit.
		isDeref := strings.HasSuffix(tagName, "^{}")
		if isDeref {
			tagName = strings.TrimSuffix(tagName, "^{}")
		}

		// Only keep tags that look like semver (plain or prefixed).
		if _, _, ok := parseSemverTag(tagName); !ok {
			continue
		}

		// Deref entries override lightweight entries.
		if isDeref || tags[tagName] == "" {
			tags[tagName] = sha
		}
	}
	return tags, nil
}

// selectTagsForPrefix narrows a tag map down to those eligible for ranking
// against a particular release prefix.
//
//   - prefix == "":       only plain (un-prefixed) semver tags.
//   - prefix == "wiki":   only `wiki-v*` tags. If none exist, falls back to
//     plain tags so single-mold repos using plain tags still work.
func selectTagsForPrefix(all map[string]string, prefix string) map[string]string {
	plain := map[string]string{}
	prefixed := map[string]string{}

	for tag, sha := range all {
		p, _, ok := parseSemverTag(tag)
		if !ok {
			continue
		}
		switch {
		case p == "":
			plain[tag] = sha
		case prefix != "" && p == prefix:
			prefixed[tag] = sha
		}
	}

	if prefix == "" {
		return plain
	}
	if len(prefixed) > 0 {
		return prefixed
	}
	return plain
}

// resolveLatest picks the highest semver tag, preferring monorepo-prefixed
// tags (`<subpath>-v*`) when the reference has a Subpath.
func resolveLatest(ref *Reference, git GitRunner) (*ResolvedVersion, error) {
	all, err := remoteTags(ref.CloneURL(), git)
	if err != nil {
		return nil, err
	}
	tags := selectTagsForPrefix(all, ref.ReleasePrefix())
	tag, sha, err := highestVersion(tags, nil)
	if err != nil {
		return nil, fmt.Errorf("no semver tags found for %s", ref.CacheKey())
	}
	return &ResolvedVersion{Tag: tag, Commit: sha}, nil
}

// resolveExact finds the exact tag matching the specified version. When the
// reference has a Subpath, prefixed candidates (`<prefix>-v1.2.3`) are tried
// before plain ones.
func resolveExact(ref *Reference, git GitRunner) (*ResolvedVersion, error) {
	tags, err := remoteTags(ref.CloneURL(), git)
	if err != nil {
		return nil, err
	}

	version := ref.Version
	bare := strings.TrimPrefix(version, "v")
	candidates := []string{version, "v" + bare, bare}
	if prefix := ref.ReleasePrefix(); prefix != "" {
		prefixed := []string{
			prefix + "-" + version,
			prefix + "-v" + bare,
			prefix + "-" + bare,
		}
		candidates = append(prefixed, candidates...)
	}

	for _, tag := range candidates {
		if sha, ok := tags[tag]; ok {
			return &ResolvedVersion{Tag: tag, Commit: sha}, nil
		}
	}
	return nil, fmt.Errorf("tag %q not found in %s", ref.Version, ref.CacheKey())
}

// resolveConstraint matches a semver constraint against available tags. When
// the reference has a Subpath, the constraint is evaluated against the
// monorepo-prefixed tags for that subpath when any exist.
func resolveConstraint(ref *Reference, git GitRunner) (*ResolvedVersion, error) {
	c, err := semver.NewConstraint(ref.Version)
	if err != nil {
		return nil, fmt.Errorf("invalid semver constraint %q: %w", ref.Version, err)
	}

	all, err := remoteTags(ref.CloneURL(), git)
	if err != nil {
		return nil, err
	}
	tags := selectTagsForPrefix(all, ref.ReleasePrefix())

	tag, sha, err := highestVersion(tags, c)
	if err != nil {
		return nil, fmt.Errorf("no tag matching %q for %s", ref.Version, ref.CacheKey())
	}
	return &ResolvedVersion{Tag: tag, Commit: sha}, nil
}

// resolveBranch resolves a branch pin to its HEAD commit.
func resolveBranch(ref *Reference, git GitRunner) (*ResolvedVersion, error) {
	log.Printf("warning: branch pin %q is mutable; consider using a semver tag", ref.Version)

	out, err := git("ls-remote", ref.CloneURL(), "refs/heads/"+ref.Version)
	if err != nil {
		return nil, fmt.Errorf("git ls-remote %s refs/heads/%s: %w\n%s", ref.CloneURL(), ref.Version, err, out)
	}

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			return &ResolvedVersion{Tag: ref.Version, Commit: parts[0]}, nil
		}
	}
	return nil, fmt.Errorf("branch %q not found in %s", ref.Version, ref.CacheKey())
}

// highestVersion picks the highest semver version from a tag map, optionally
// filtered by a constraint. Returns the tag name, SHA, and nil error on success.
func highestVersion(tags map[string]string, c *semver.Constraints) (string, string, error) {
	type entry struct {
		tag string
		ver *semver.Version
		sha string
	}

	var entries []entry
	for tag, sha := range tags {
		_, normalized, ok := parseSemverTag(tag)
		if !ok {
			continue
		}
		v, err := semver.NewVersion(normalized)
		if err != nil {
			continue
		}
		if c != nil && !c.Check(v) {
			continue
		}
		entries = append(entries, entry{tag: tag, ver: v, sha: sha})
	}

	if len(entries) == 0 {
		return "", "", fmt.Errorf("no matching versions")
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ver.LessThan(entries[j].ver)
	})

	best := entries[len(entries)-1]
	return best.tag, best.sha, nil
}
