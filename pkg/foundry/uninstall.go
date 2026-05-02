package foundry

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// UninstallOptions controls UninstallMold behavior.
type UninstallOptions struct {
	Force  bool // delete even if file content has been modified since cast
	DryRun bool // report what would happen, don't touch disk or lockfile
}

// UninstallResult summarizes the outcome of an uninstall.
type UninstallResult struct {
	Deleted         []string // files removed (or that would be removed in dry-run)
	SkippedModified []string // files retained because user-modified (no --force)
	NotFound        []string // files in manifest that didn't exist on disk
	Retained        []string // files retained because shared with another lock entry
	LegacyManifest  bool     // entry had no Files manifest (legacy lockfile)
}

// ErrLegacyEntry is returned when the lock entry has no Files manifest.
// Caller should re-cast the mold to backfill the manifest before retrying.
var ErrLegacyEntry = errors.New("lock entry has no install manifest (legacy); re-cast the mold to enable safe uninstall")

// UninstallMold removes the files listed in a mold's lock entry, prunes any
// now-empty parent directories, and removes the entry from the lockfile.
// Files modified since cast (content hash differs) are skipped unless
// opts.Force is set. Files claimed by another lock entry are retained.
func UninstallMold(lockPath, source string, opts UninstallOptions) (UninstallResult, error) {
	var res UninstallResult

	lock, err := ReadLockFile(lockPath)
	if err != nil {
		return res, fmt.Errorf("reading lock file: %w", err)
	}
	if lock == nil {
		return res, fmt.Errorf("lock file %s does not exist", lockPath)
	}

	entry := lock.FindEntry(source)
	if entry == nil {
		return res, fmt.Errorf("no lock entry for source %q", source)
	}

	if entry.Files == nil {
		res.LegacyManifest = true
		return res, ErrLegacyEntry
	}

	// Build a set of paths claimed by other entries.
	otherClaims := make(map[string]struct{})
	for i := range lock.Molds {
		if lock.Molds[i].Source == source {
			continue
		}
		for _, f := range lock.Molds[i].Files {
			otherClaims[f] = struct{}{}
		}
	}

	lockDir := filepath.Dir(lockPath)
	dirsTouched := make(map[string]struct{})

	// Process files in reverse path order so deeper paths come first
	// (helps with empty-dir pruning later).
	files := append([]string(nil), entry.Files...)
	sort.Sort(sort.Reverse(sort.StringSlice(files)))

	for _, rel := range files {
		if _, shared := otherClaims[rel]; shared {
			res.Retained = append(res.Retained, rel)
			continue
		}

		abs := filepath.Join(lockDir, filepath.FromSlash(rel))

		st, err := os.Stat(abs)
		if err != nil {
			if os.IsNotExist(err) {
				res.NotFound = append(res.NotFound, rel)
				continue
			}
			return res, fmt.Errorf("stat %s: %w", rel, err)
		}
		if st.IsDir() {
			// Manifests should never list directories; skip defensively.
			res.NotFound = append(res.NotFound, rel)
			continue
		}

		if !opts.Force {
			modified, err := fileModifiedSinceCast(abs, entry.FileHashes[rel])
			if err != nil {
				return res, fmt.Errorf("checking %s: %w", rel, err)
			}
			if modified {
				res.SkippedModified = append(res.SkippedModified, rel)
				continue
			}
		}

		res.Deleted = append(res.Deleted, rel)
		if opts.DryRun {
			continue
		}
		if err := os.Remove(abs); err != nil {
			return res, fmt.Errorf("removing %s: %w", rel, err)
		}
		dirsTouched[filepath.Dir(abs)] = struct{}{}
	}

	sort.Strings(res.Deleted)
	sort.Strings(res.SkippedModified)
	sort.Strings(res.NotFound)
	sort.Strings(res.Retained)

	if opts.DryRun {
		return res, nil
	}

	dirs := make([]string, 0, len(dirsTouched))
	for d := range dirsTouched {
		dirs = append(dirs, d)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dirs)))
	for _, d := range dirs {
		pruneEmptyDirs(d, lockDir)
	}

	for i := range lock.Molds {
		if lock.Molds[i].Source == source {
			lock.Molds = append(lock.Molds[:i], lock.Molds[i+1:]...)
			break
		}
	}
	if err := WriteLockFile(lockPath, lock); err != nil {
		return res, fmt.Errorf("writing lock file: %w", err)
	}

	return res, nil
}

func fileModifiedSinceCast(path, expectedHash string) (bool, error) {
	if expectedHash == "" {
		return true, nil
	}
	f, err := os.Open(path) // #nosec G304 -- uninstall reads files from a user-controlled lock manifest by design
	if err != nil {
		return false, err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false, err
	}
	got := hex.EncodeToString(h.Sum(nil))
	return got != expectedHash, nil
}

// pruneEmptyDirs walks up from dir, removing each empty directory until it
// hits a non-empty dir or stopAt (exclusive).
func pruneEmptyDirs(dir, stopAt string) {
	for {
		if dir == stopAt || dir == filepath.Dir(dir) {
			return
		}
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			return
		}
		if err := os.Remove(dir); err != nil {
			return
		}
		dir = filepath.Dir(dir)
	}
}
