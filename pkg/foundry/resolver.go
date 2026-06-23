package foundry

import (
	"errors"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// ErrNoSemverTags is returned by resolveLatest when the foundry has no semver
// tags. Callers can detect this with errors.Is to offer a fallback (e.g. cast
// from the default branch HEAD).
var ErrNoSemverTags = errors.New("no semver tags found")

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
	// MoldVersion is the version declared in the mold's mold.yaml at Tag.
	// Empty when no MoldVersionReader was supplied or the mold declared no
	// parseable version (in which case the constraint was matched against
	// the tag-embedded semver).
	MoldVersion string
}

// MoldVersionReader reports the version declared in a mold's mold.yaml at a
// given git tag. found=false means no mold manifest exists at that tag (the
// candidate tag is then excluded). found=true with an empty version means the
// manifest exists but declares no parseable version, so callers fall back to
// the tag-embedded semver.
//
// Release-train monorepos tag every mold with a shared train version
// (`launch-v0.7.1`, `ce-v0.7.1`, ...) while each mold carries its own semver
// in mold.yaml. A reader lets the resolver rank candidates by the mold's own
// version rather than the train version baked into the tag name.
type MoldVersionReader func(tag string) (version string, found bool)

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

// RankVersion returns the semver used to rank a candidate tag against a
// version constraint. When moldVersion is non-empty it is authoritative — the
// mold declares its own version in mold.yaml, which on a release-train
// monorepo differs from the version baked into the tag name. Otherwise the
// version embedded in the tag name is used as a fallback (single-mold repos,
// or molds with no parseable version field).
func RankVersion(tag, moldVersion string) (*semver.Version, bool) {
	if moldVersion != "" {
		if v, err := semver.NewVersion(moldVersion); err == nil {
			return v, true
		}
	}
	_, normalized, ok := parseSemverTag(tag)
	if !ok {
		return nil, false
	}
	v, err := semver.NewVersion(normalized)
	if err != nil {
		return nil, false
	}
	return v, true
}

// ResolveVersion resolves a Reference to a concrete tag + commit SHA using
// git ls-remote. It does not require a local clone, and ranks candidate tags
// by their tag-embedded semver. For release-train monorepos use
// ResolveVersionWithMoldReader.
func ResolveVersion(ref *Reference, git GitRunner) (*ResolvedVersion, error) {
	return ResolveVersionWithMoldReader(ref, git, nil)
}

// ResolveVersionWithMoldReader is like ResolveVersion but ranks candidate
// tags by the dependency mold's declared mold.yaml version (supplied by
// reader) instead of the tag-embedded semver. A nil reader behaves exactly
// like ResolveVersion.
func ResolveVersionWithMoldReader(ref *Reference, git GitRunner, reader MoldVersionReader) (*ResolvedVersion, error) {
	switch ref.Type {
	case Latest:
		return resolveLatest(ref, git, reader)
	case Exact:
		return resolveExact(ref, git, reader)
	case Constraint:
		return resolveConstraint(ref, git, reader)
	case Branch:
		return resolveBranch(ref, git)
	case SHA:
		return &ResolvedVersion{Tag: ref.Version, Commit: ref.Version}, nil
	default:
		return nil, fmt.Errorf("unsupported ref type: %v", ref.Type)
	}
}

// RemoteTags lists the semver tags (tag → commit SHA) for the given remote
// URL, optionally narrowed to a monorepo subpath via the prefix selection
// rules (`<subpath>-v*` tags when present, falling back to plain tags).
//
// Used by graph-aware resolvers (transitive mold deps) that need to intersect
// constraints from multiple parents and pick the highest-compatible version
// without re-issuing one git ls-remote per constraint.
func RemoteTags(url, subpath string, git GitRunner) (map[string]string, error) {
	all, err := remoteTags(url, git)
	if err != nil {
		return nil, err
	}
	prefix := ""
	if s := strings.Trim(subpath, "/"); s != "" {
		if i := strings.LastIndex(s, "/"); i != -1 {
			prefix = s[i+1:]
		} else {
			prefix = s
		}
	}
	return selectTagsForPrefix(all, prefix), nil
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

// resolveLatest picks the highest-versioned tag, preferring monorepo-prefixed
// tags (`<subpath>-v*`) when the reference has a Subpath. With a reader,
// candidates are ranked by their mold.yaml version.
func resolveLatest(ref *Reference, git GitRunner, reader MoldVersionReader) (*ResolvedVersion, error) {
	all, err := remoteTags(ref.CloneURL(), git)
	if err != nil {
		return nil, err
	}
	tags := selectTagsForPrefix(all, ref.ReleasePrefix())
	tag, sha, moldVersion, err := highestVersion(tags, nil, reader)
	if err != nil {
		return nil, fmt.Errorf("%w for %s", ErrNoSemverTags, ref.CacheKey())
	}
	return &ResolvedVersion{Tag: tag, Commit: sha, MoldVersion: moldVersion}, nil
}

// resolveExact finds the exact tag matching the specified version. When the
// reference has a Subpath, prefixed candidates (`<prefix>-v1.2.3`) are tried
// before plain ones. With a reader, an exact version that matches no literal
// tag name is matched against the declared mold.yaml versions instead — on a
// release-train monorepo `@0.2.1` is the mold's own version, which lives at a
// differently-named tag (e.g. `launch-v0.7.1`).
func resolveExact(ref *Reference, git GitRunner, reader MoldVersionReader) (*ResolvedVersion, error) {
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

	if reader != nil {
		if rv, ok := resolveExactByMoldVersion(ref, tags, reader); ok {
			return rv, nil
		}
	}
	return nil, fmt.Errorf("tag %q not found in %s", ref.Version, ref.CacheKey())
}

// resolveExactByMoldVersion finds the tag whose mold.yaml version equals the
// reference's exact version. When several tags carry the same mold version
// (the mold was unchanged across release-train releases) the one with the
// highest tag-embedded semver wins, so the newest checkout pointer is used.
func resolveExactByMoldVersion(ref *Reference, all map[string]string, reader MoldVersionReader) (*ResolvedVersion, bool) {
	want, err := semver.NewVersion(strings.TrimPrefix(ref.Version, "v"))
	if err != nil {
		return nil, false
	}
	tags := selectTagsForPrefix(all, ref.ReleasePrefix())
	var best *ResolvedVersion
	var bestRank *semver.Version
	for tag, sha := range tags {
		mv, found := reader(tag)
		if !found || mv == "" {
			continue
		}
		v, err := semver.NewVersion(mv)
		if err != nil || !v.Equal(want) {
			continue
		}
		rank, ok := RankVersion(tag, "")
		if best == nil || (ok && bestRank != nil && bestRank.LessThan(rank)) {
			best = &ResolvedVersion{Tag: tag, Commit: sha, MoldVersion: mv}
			bestRank = rank
		}
	}
	return best, best != nil
}

// resolveConstraint matches a semver constraint against available tags. When
// the reference has a Subpath, the constraint is evaluated against the
// monorepo-prefixed tags for that subpath when any exist.
func resolveConstraint(ref *Reference, git GitRunner, reader MoldVersionReader) (*ResolvedVersion, error) {
	c, err := semver.NewConstraint(ref.Version)
	if err != nil {
		return nil, fmt.Errorf("invalid semver constraint %q: %w", ref.Version, err)
	}

	all, err := remoteTags(ref.CloneURL(), git)
	if err != nil {
		return nil, err
	}
	tags := selectTagsForPrefix(all, ref.ReleasePrefix())

	tag, sha, moldVersion, err := highestVersion(tags, c, reader)
	if err != nil {
		return nil, fmt.Errorf("no tag matching %q for %s", ref.Version, ref.CacheKey())
	}
	return &ResolvedVersion{Tag: tag, Commit: sha, MoldVersion: moldVersion}, nil
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

// ResolveDefaultBranchHead resolves the HEAD commit on the default branch of a
// foundry. It is used as a fallback when no semver tags are found. The
// ResolvedVersion.Tag is set to the full commit SHA so the fetcher caches it
// under a stable, content-addressed path.
func ResolveDefaultBranchHead(ref *Reference, git GitRunner) (*ResolvedVersion, error) {
	out, err := git("ls-remote", ref.CloneURL(), "HEAD")
	if err != nil {
		return nil, fmt.Errorf("git ls-remote %s HEAD: %w\n%s", ref.CloneURL(), err, out)
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[1] == "HEAD" {
			return &ResolvedVersion{Tag: parts[0], Commit: parts[0]}, nil
		}
	}
	return nil, fmt.Errorf("could not resolve HEAD for %s", ref.CacheKey())
}

// highestVersion picks the highest-versioned tag from a tag map, optionally
// filtered by a constraint. Candidates are ranked by their mold.yaml version
// when reader is non-nil; tags whose mold manifest is absent (reader reports
// found=false) are excluded. Returns the tag name, SHA, the mold version used
// for ranking (empty when ranked by the tag-embedded semver), and a nil error
// on success.
func highestVersion(tags map[string]string, c *semver.Constraints, reader MoldVersionReader) (string, string, string, error) {
	type entry struct {
		tag         string
		ver         *semver.Version // rank version (mold version when known)
		tagVer      *semver.Version // tag-embedded version, for tie-breaking
		sha         string
		moldVersion string
	}

	var entries []entry
	for tag, sha := range tags {
		var moldVersion string
		if reader != nil {
			v, found := reader(tag)
			if !found {
				continue // mold manifest absent at this tag
			}
			moldVersion = v
		}
		v, ok := RankVersion(tag, moldVersion)
		if !ok {
			continue
		}
		if c != nil && !c.Check(v) {
			continue
		}
		tagVer, _ := RankVersion(tag, "")
		entries = append(entries, entry{tag: tag, ver: v, tagVer: tagVer, sha: sha, moldVersion: moldVersion})
	}

	if len(entries) == 0 {
		return "", "", "", fmt.Errorf("no matching versions")
	}

	sort.Slice(entries, func(i, j int) bool {
		return lessEntry(entries[i].ver, entries[i].tagVer, entries[i].tag,
			entries[j].ver, entries[j].tagVer, entries[j].tag)
	})

	best := entries[len(entries)-1]
	return best.tag, best.sha, best.moldVersion, nil
}

// lessEntry orders candidates by rank version, then by the tag-embedded
// version as a tie-breaker (release-train monorepos share one mold version
// across many tags — the newest tag is the right checkout pointer), then by
// tag name for total determinism.
func lessEntry(vi, ti *semver.Version, tagI string, vj, tj *semver.Version, tagJ string) bool {
	if !vi.Equal(vj) {
		return vi.LessThan(vj)
	}
	if ti != nil && tj != nil && !ti.Equal(tj) {
		return ti.LessThan(tj)
	}
	return tagI < tagJ
}
