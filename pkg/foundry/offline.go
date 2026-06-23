package foundry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// NewOfflineGitRunner wraps a GitRunner so that network-requiring git commands
// are intercepted and served from the local bare-clone cache instead. Commands
// that already operate on local data (git archive, git show, git for-each-ref
// with -C, etc.) pass through unchanged.
//
// Two commands are intercepted:
//   - git ls-remote --tags <url>   → reads tags from the local bare clone
//   - git -C <dir> fetch ...       → no-op if the bare clone exists, error if not
//
// Any other network-requiring command (ls-remote for a branch, clone --bare)
// returns an actionable error that names --offline as the cause.
func NewOfflineGitRunner(git GitRunner, cacheDir string) GitRunner {
	return func(args ...string) ([]byte, error) {
		// git ls-remote --tags <url>
		if len(args) >= 3 && args[0] == "ls-remote" && args[1] == "--tags" {
			url := args[2]
			bareDir := bareDirForURL(url, cacheDir)
			return localTagsOutput(bareDir, git)
		}

		// git -C <dir> fetch ...  →  no-op (skip network fetch of bare clone)
		if len(args) >= 3 && args[0] == "-C" && args[2] == "fetch" {
			bareDir := args[1]
			if _, err := os.Stat(filepath.Join(bareDir, "HEAD")); err != nil {
				return nil, fmt.Errorf("offline mode: no cached clone at %s; run without --offline to fetch it", bareDir)
			}
			return nil, nil
		}

		// git clone --bare <url> <dir>  →  error (can't create new bare clone offline)
		if len(args) >= 2 && args[0] == "clone" {
			for _, a := range args {
				if a == "--bare" {
					url := ""
					for i, a2 := range args {
						if a2 == "--bare" && i+1 < len(args) {
							url = args[i+1]
						}
					}
					return nil, fmt.Errorf("offline mode: no cached clone for %q; run without --offline to fetch it", url)
				}
			}
		}

		// git ls-remote <url> refs/heads/<branch>  →  error (branch resolution needs network)
		if len(args) >= 1 && args[0] == "ls-remote" {
			return nil, fmt.Errorf("offline mode: cannot fetch remote refs; run without --offline")
		}

		// All other commands (git show, git archive, git for-each-ref, etc.) pass through.
		return git(args...)
	}
}

// bareDirForURL derives the local bare-clone path for a remote clone URL.
// "https://github.com/owner/repo.git" → "<cacheDir>/github.com/owner/repo/git"
func bareDirForURL(url, cacheDir string) string {
	key := strings.TrimSuffix(strings.TrimPrefix(url, "https://"), ".git")
	return filepath.Join(cacheDir, filepath.FromSlash(key), "git")
}

// localTagsOutput reads semver tags from a local bare clone and returns bytes
// in the same format as "git ls-remote --tags", suitable for parseLsRemoteTags.
//
// Two for-each-ref calls are combined:
//  1. Plain lines:  <sha>	refs/tags/<name>
//  2. Deref lines: <sha>	refs/tags/<name>^{}   (non-empty only for annotated tags)
//
// The deref lines let parseLsRemoteTags correctly resolve annotated tags to
// their underlying commit SHA (matching the ls-remote behaviour exactly).
func localTagsOutput(bareDir string, git GitRunner) ([]byte, error) {
	if _, err := os.Stat(filepath.Join(bareDir, "HEAD")); err != nil {
		return nil, fmt.Errorf("offline mode: no cached clone at %s; run without --offline to fetch it", bareDir)
	}

	plain, err := git("-C", bareDir, "for-each-ref", "refs/tags",
		"--format=%(objectname)\trefs/tags/%(refname:short)")
	if err != nil {
		return nil, fmt.Errorf("offline mode: listing local tags: %w", err)
	}

	// %(*objectname) is empty for lightweight tags; parseLsRemoteTags skips
	// those blank-SHA lines, so combining them here is safe.
	deref, err := git("-C", bareDir, "for-each-ref", "refs/tags",
		"--format=%(*objectname)\trefs/tags/%(refname:short)^{}")
	if err != nil {
		return nil, fmt.Errorf("offline mode: listing local tag derefs: %w", err)
	}

	combined := append(plain, '\n')
	combined = append(combined, deref...)
	return combined, nil
}
