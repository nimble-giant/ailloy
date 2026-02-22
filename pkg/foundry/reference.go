package foundry

import (
	"fmt"
	"regexp"
	"strings"
)

// RefType classifies the version specifier in a mold reference.
type RefType int

const (
	// Latest means no version was specified; resolve to the newest semver tag.
	Latest RefType = iota
	// Constraint is a semver range (^, ~, >=, <, etc.).
	Constraint
	// Exact is a specific semver version (e.g. 1.2.3 or v1.2.3).
	Exact
	// Branch pins to a branch name.
	Branch
	// SHA pins to a commit hash.
	SHA
)

// Reference is a parsed mold reference in the format:
//
//	<host>/<owner>/<repo>[@<version>][//<subpath>]
type Reference struct {
	Host    string
	Owner   string
	Repo    string
	Version string
	Subpath string
	Type    RefType
}

var (
	shaPattern    = regexp.MustCompile(`^[0-9a-f]{7,40}$`)
	semverPattern = regexp.MustCompile(`^v?\d+\.\d+\.\d+`)
	constraintPre = regexp.MustCompile(`^[~^>=<!]`)
)

// ParseReference parses a raw mold reference string into a Reference.
//
// Accepted formats:
//
//	github.com/owner/repo
//	github.com/owner/repo@v1.2.3
//	github.com/owner/repo@^1.0.0//subpath
//	https://github.com/owner/repo@v1.0.0
//	git@github.com:owner/repo@v1.0.0
func ParseReference(raw string) (*Reference, error) {
	if raw == "" {
		return nil, fmt.Errorf("empty reference")
	}

	s := raw

	// Strip URL schemes.
	if after, ok := strings.CutPrefix(s, "https://"); ok {
		s = after
	} else if after, ok := strings.CutPrefix(s, "http://"); ok {
		s = after
	}

	// Normalise SSH shorthand: git@github.com:owner/repo → github.com/owner/repo
	if after, ok := strings.CutPrefix(s, "git@"); ok {
		s = strings.Replace(after, ":", "/", 1) // first colon only
	}

	// Split off //subpath (must come before @version split).
	var subpath string
	if idx := strings.Index(s, "//"); idx != -1 {
		subpath = s[idx+2:]
		s = s[:idx]
	}

	// Split off @version.
	var version string
	if idx := strings.LastIndex(s, "@"); idx != -1 {
		version = s[idx+1:]
		s = s[:idx]
	}

	// Strip trailing .git (after version/subpath extraction so it doesn't
	// interfere with patterns like repo.git@v1.0.0).
	s = strings.TrimSuffix(s, ".git")

	// Now s should be host/owner/repo.
	parts := strings.Split(s, "/")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid reference %q: expected <host>/<owner>/<repo>", raw)
	}

	host := parts[0]
	owner := parts[1]
	repo := strings.Join(parts[2:], "/") // allow nested paths in repo segment

	if host == "" || owner == "" || repo == "" {
		return nil, fmt.Errorf("invalid reference %q: host, owner, and repo must be non-empty", raw)
	}

	ref := &Reference{
		Host:    host,
		Owner:   owner,
		Repo:    repo,
		Version: version,
		Subpath: subpath,
		Type:    classifyVersion(version),
	}
	return ref, nil
}

// classifyVersion determines the RefType from a version string.
func classifyVersion(v string) RefType {
	if v == "" || v == "latest" {
		return Latest
	}
	if shaPattern.MatchString(v) {
		return SHA
	}
	if constraintPre.MatchString(v) {
		return Constraint
	}
	if semverPattern.MatchString(v) {
		return Exact
	}
	return Branch
}

// IsRemoteReference returns true when the string looks like a remote mold
// reference rather than a local filesystem path.
//
// Heuristic: true when the first path segment contains a dot (e.g.
// "github.com/..."), or when the string starts with "https://", "http://",
// or "git@". False for relative ("./local") or absolute ("/abs/path") paths.
func IsRemoteReference(s string) bool {
	if s == "" {
		return false
	}
	if strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "git@") {
		return true
	}
	// Local paths: start with / . ~ or have no dot in the first segment.
	if s[0] == '/' || s[0] == '.' || s[0] == '~' {
		return false
	}
	// First segment (before /) must contain a dot to look like a hostname.
	firstSeg, _, _ := strings.Cut(s, "/")
	return strings.Contains(firstSeg, ".")
}

// CloneURL returns the HTTPS clone URL for the reference.
func (r *Reference) CloneURL() string {
	return fmt.Sprintf("https://%s/%s/%s.git", r.Host, r.Owner, r.Repo)
}

// CacheKey returns the cache directory key: host/owner/repo.
func (r *Reference) CacheKey() string {
	return fmt.Sprintf("%s/%s/%s", r.Host, r.Owner, r.Repo)
}

// String returns a human-readable representation of the reference.
func (r *Reference) String() string {
	s := r.CacheKey()
	if r.Version != "" {
		s += "@" + r.Version
	}
	if r.Subpath != "" {
		s += "//" + r.Subpath
	}
	return s
}
