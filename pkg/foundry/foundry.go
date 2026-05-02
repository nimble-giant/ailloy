package foundry

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// InstalledFile carries the metadata recorded for one cast file.
type InstalledFile struct {
	RelPath string // path relative to the lockfile dir, forward-slash separated
	SHA256  string // hex-encoded sha256 of file content at install time
}

// RecordInstalledFiles backfills the Files list and FileHashes map on the
// lock entry whose Source matches. Hashes are sha256 hex of the file content
// at install time, used by UninstallMold to detect user modifications.
func RecordInstalledFiles(lockPath, source string, files []InstalledFile) error {
	lock, err := ReadLockFile(lockPath)
	if err != nil {
		return fmt.Errorf("reading lock file: %w", err)
	}
	if lock == nil {
		return fmt.Errorf("lock file %s does not exist", lockPath)
	}

	entry := lock.FindEntry(source)
	if entry == nil {
		return fmt.Errorf("no lock entry for source %q", source)
	}

	seen := make(map[string]struct{}, len(files))
	paths := make([]string, 0, len(files))
	hashes := make(map[string]string, len(files))
	for _, f := range files {
		s := filepath.ToSlash(f.RelPath)
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		paths = append(paths, s)
		if f.SHA256 != "" {
			hashes[s] = f.SHA256
		}
	}
	sort.Strings(paths)
	entry.Files = paths
	if len(hashes) > 0 {
		entry.FileHashes = hashes
	} else {
		entry.FileHashes = nil
	}

	return WriteLockFile(lockPath, lock)
}

// ResolveOption configures optional behaviour for Resolve.
type ResolveOption func(*resolveConfig)

type resolveConfig struct {
	// lockPath is the path to the lock file. The lock is *only* used (read &
	// updated) when a file already exists at this path. Resolve never creates
	// the lock — that is the job of `ailloy quench`.
	lockPath string
}

// applyResolveDefaults sets the default lockPath. Exposed for tests.
func applyResolveDefaults(c *resolveConfig) {
	if c.lockPath == "" {
		c.lockPath = LockFileName
	}
}

// WithLockPath overrides the lock file path used by Resolve. Use this for
// global installs (which use a path under ~/) or in tests.
func WithLockPath(path string) ResolveOption {
	return func(c *resolveConfig) {
		c.lockPath = path
	}
}

// shouldUseLock returns true when a lock file exists at the configured path.
// Lock reads/writes are gated on file presence — opt-in via `ailloy quench`.
func shouldUseLock(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

// Resolve is the main entry point for SCM-native mold resolution.
// It parses a raw reference, optionally consults an existing ailloy.lock,
// resolves the version from remote tags when not locked, fetches/caches the
// mold, updates the lock if one already exists, and returns an fs.FS rooted
// at the mold directory.
func Resolve(rawRef string, opts ...ResolveOption) (fs.FS, error) {
	fsys, _, err := ResolveWithRoot(rawRef, opts...)
	return fsys, err
}

// ResolveWithRoot is like Resolve but also returns the on-disk root path of
// the resolved mold. The root path is needed by callers that resolve sibling
// directories (e.g., bundled ingots) during template rendering.
func ResolveWithRoot(rawRef string, opts ...ResolveOption) (fs.FS, string, error) {
	ref, err := ParseReference(rawRef)
	if err != nil {
		return nil, "", fmt.Errorf("parsing reference: %w", err)
	}
	git := DefaultGitRunner()
	return ResolveWith(ref, git, opts...)
}

// ResolveWith resolves a parsed reference using the given GitRunner.
func ResolveWith(ref *Reference, git GitRunner, opts ...ResolveOption) (fs.FS, string, error) {
	fsys, result, err := resolveWithMeta(ref, git, opts...)
	if err != nil {
		return nil, "", err
	}
	return fsys, result.Root, nil
}

// ResolveResult captures resolution outputs callers need to record provenance.
type ResolveResult struct {
	Ref      *Reference
	Resolved ResolvedVersion
	Root     string
}

// ResolveWithMetadata is like Resolve but returns provenance details for the
// caller to record in the installed manifest. On error, the returned
// *ResolveResult is always nil.
func ResolveWithMetadata(rawRef string, opts ...ResolveOption) (fs.FS, *ResolveResult, error) {
	ref, err := ParseReference(rawRef)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing reference: %w", err)
	}
	git := DefaultGitRunner()
	fsys, result, err := resolveWithMeta(ref, git, opts...)
	if err != nil {
		return nil, nil, err
	}
	return fsys, result, nil
}

// resolveWithMeta is the internal implementation; mirrors ResolveWith but also
// returns the ResolvedVersion alongside the fs.FS.
func resolveWithMeta(ref *Reference, git GitRunner, opts ...ResolveOption) (fs.FS, *ResolveResult, error) {
	var cfg resolveConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	applyResolveDefaults(&cfg)

	useLock := shouldUseLock(cfg.lockPath)

	var resolved *ResolvedVersion
	if useLock {
		lock, err := ReadLockFile(cfg.lockPath)
		if err != nil {
			log.Printf("warning: reading lock file: %v", err)
		}
		if entry := lock.FindEntry(ref.CacheKey()); entry != nil && ref.Type != Branch && ref.Type != SHA {
			if lockedSatisfies(ref, entry) {
				resolved = &ResolvedVersion{Tag: entry.Version, Commit: entry.Commit}
				log.Printf("using locked version %s@%s", ref.CacheKey(), entry.Version)
			}
		}
	}

	if resolved == nil {
		v, resolveErr := ResolveVersion(ref, git)
		if resolveErr != nil {
			return nil, nil, fmt.Errorf("resolving version: %w", resolveErr)
		}
		resolved = v
	}

	fetcher, err := NewFetcher(git)
	if err != nil {
		return nil, nil, fmt.Errorf("creating fetcher: %w", err)
	}
	fsys, root, err := fetcher.Fetch(ref, resolved)
	if err != nil {
		return nil, nil, fmt.Errorf("fetching mold: %w", err)
	}

	if useLock {
		if err := updateLockAt(cfg.lockPath, ref, resolved); err != nil {
			log.Printf("warning: updating lock file: %v", err)
		}
	}

	return fsys, &ResolveResult{Ref: ref, Resolved: *resolved, Root: root}, nil
}

func lockedSatisfies(ref *Reference, entry *LockEntry) bool {
	switch ref.Type {
	case Latest, Constraint:
		return true
	case Exact:
		return entry.Version == ref.Version || entry.Version == "v"+ref.Version
	default:
		return false
	}
}

// updateLockAt reads, upserts, and writes the lock at the given path.
func updateLockAt(path string, ref *Reference, resolved *ResolvedVersion) error {
	lock, err := ReadLockFile(path)
	if err != nil {
		lock = nil
	}
	if lock == nil {
		lock = &LockFile{APIVersion: "v1"}
	}
	lock.UpsertEntry(LockEntry{
		Name:      ref.Repo,
		Source:    ref.CacheKey(),
		Version:   resolved.Tag,
		Commit:    resolved.Commit,
		Subpath:   ref.Subpath,
		Timestamp: time.Now().UTC(),
	})
	return WriteLockFile(path, lock)
}
