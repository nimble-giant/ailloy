package foundry

import (
	"fmt"
	"io/fs"
	"log"
	"time"
)

// Resolve is the main entry point for SCM-native mold resolution.
// It parses a raw reference, checks the lock file, resolves the version
// from remote tags, fetches/caches the mold, updates the lock, and returns
// an fs.FS rooted at the mold directory.
func Resolve(rawRef string) (fs.FS, error) {
	ref, err := ParseReference(rawRef)
	if err != nil {
		return nil, fmt.Errorf("parsing reference: %w", err)
	}

	git := DefaultGitRunner()
	return ResolveWith(ref, git)
}

// ResolveWith is like Resolve but accepts an injectable GitRunner (for testing).
func ResolveWith(ref *Reference, git GitRunner) (fs.FS, error) {
	// Read existing lock file.
	lock, err := ReadLockFile(LockFileName)
	if err != nil {
		log.Printf("warning: reading lock file: %v", err)
	}

	// Check lock for a pinned version.
	var resolved *ResolvedVersion
	if entry := lock.FindEntry(ref.CacheKey()); entry != nil && ref.Type != Branch && ref.Type != SHA {
		// Use locked version if it satisfies the reference.
		if lockedSatisfies(ref, entry) {
			resolved = &ResolvedVersion{Tag: entry.Version, Commit: entry.Commit}
			log.Printf("using locked version %s@%s", ref.CacheKey(), entry.Version)
		}
	}

	// Resolve version from remote if not locked.
	if resolved == nil {
		resolved, err = ResolveVersion(ref, git)
		if err != nil {
			return nil, fmt.Errorf("resolving version: %w", err)
		}
	}

	// Fetch (clone/cache).
	fetcher, err := NewFetcher(git)
	if err != nil {
		return nil, fmt.Errorf("creating fetcher: %w", err)
	}

	fsys, err := fetcher.Fetch(ref, resolved)
	if err != nil {
		return nil, fmt.Errorf("fetching mold: %w", err)
	}

	// Update lock file.
	if err := updateLock(ref, resolved); err != nil {
		log.Printf("warning: updating lock file: %v", err)
	}

	return fsys, nil
}

// lockedSatisfies checks whether the locked entry still satisfies the
// requested reference. For Latest and Constraint types, the lock is always
// used (user must delete lock to get newer versions). For Exact, the locked
// version must match.
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

// updateLock reads the current lock file, upserts the resolved entry, and
// writes it back.
func updateLock(ref *Reference, resolved *ResolvedVersion) error {
	lock, err := ReadLockFile(LockFileName)
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

	return WriteLockFile(LockFileName, lock)
}
