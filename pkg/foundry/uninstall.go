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
	DryRun bool // report what would happen, don't touch disk or manifest
}

// UninstallResult summarizes the outcome of an uninstall.
type UninstallResult struct {
	Deleted         []string // files removed (or that would be removed in dry-run)
	SkippedModified []string // files retained because user-modified (no --force)
	NotFound        []string // files in manifest that didn't exist on disk
	Retained        []string // files retained because shared with another manifest entry
	LegacyManifest  bool     // entry had no Files manifest (legacy install)
}

// ErrLegacyEntry is returned when the manifest entry has no Files list.
// Caller should re-cast the mold to backfill the Files list before retrying.
var ErrLegacyEntry = errors.New("manifest entry has no install file list (legacy); re-cast the mold to enable safe uninstall")

// UninstallMold removes the files listed in a mold's installed-manifest entry,
// prunes any now-empty parent directories, and removes the entry from the
// manifest. If a lock file exists alongside the manifest, the matching lock
// entry is also removed. Files modified since cast (content hash differs) are
// skipped unless opts.Force is set. Files claimed by another manifest entry
// are retained.
//
// manifestPath is the path to .ailloy/installed.yaml (project) or
// ~/.ailloy/installed.yaml (global). The lock that mirrors it is derived by
// looking for ailloy.lock alongside the manifest's containing directory.
func UninstallMold(manifestPath, source string, opts UninstallOptions) (UninstallResult, error) {
	var res UninstallResult

	m, err := ReadInstalledManifest(manifestPath)
	if err != nil {
		return res, fmt.Errorf("reading installed manifest: %w", err)
	}
	if m == nil {
		return res, fmt.Errorf("installed manifest %s does not exist", manifestPath)
	}

	entry := m.FindBySource(source)
	if entry == nil {
		return res, fmt.Errorf("no installed manifest entry for source %q", source)
	}

	if entry.Files == nil {
		res.LegacyManifest = true
		return res, ErrLegacyEntry
	}

	// Build a set of paths claimed by other entries.
	otherClaims := make(map[string]struct{})
	for i := range m.Molds {
		if m.Molds[i].Source == source {
			continue
		}
		for _, f := range m.Molds[i].Files {
			otherClaims[f] = struct{}{}
		}
	}

	// Files in the manifest are stored relative to the manifest's *containing*
	// project root — i.e., the directory above .ailloy/. This matches what
	// cast.go records (rf.DestPath is project-relative).
	rootDir := projectRootForManifest(manifestPath)
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

		abs := filepath.Join(rootDir, filepath.FromSlash(rel))

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
		pruneEmptyDirs(d, rootDir)
	}

	for i := range m.Molds {
		if m.Molds[i].Source == source {
			m.Molds = append(m.Molds[:i], m.Molds[i+1:]...)
			break
		}
	}
	if err := WriteInstalledManifest(manifestPath, m); err != nil {
		return res, fmt.Errorf("writing installed manifest: %w", err)
	}

	// Best-effort: also drop the matching lock entry if a lock exists alongside.
	lockPath := filepath.Join(rootDir, LockFileName)
	if lock, lerr := ReadLockFile(lockPath); lerr == nil && lock != nil {
		dropped := false
		for i := range lock.Molds {
			if lock.Molds[i].Source == source {
				lock.Molds = append(lock.Molds[:i], lock.Molds[i+1:]...)
				dropped = true
				break
			}
		}
		if dropped {
			_ = WriteLockFile(lockPath, lock)
		}
	}

	return res, nil
}

// projectRootForManifest returns the directory that contains the manifest's
// .ailloy/ subdirectory. Cast records files relative to this root.
//
// For ".ailloy/installed.yaml" we want "."; for "/home/u/.ailloy/installed.yaml"
// we want "/home/u". The rule is: take the parent of the directory holding the
// manifest file. If the manifest path doesn't follow the .ailloy/ convention,
// fall back to the manifest's directory.
func projectRootForManifest(manifestPath string) string {
	dir := filepath.Dir(manifestPath)
	if filepath.Base(dir) == ".ailloy" {
		return filepath.Dir(dir)
	}
	return dir
}

func fileModifiedSinceCast(path, expectedHash string) (bool, error) {
	if expectedHash == "" {
		return true, nil
	}
	f, err := os.Open(path) // #nosec G304 -- uninstall reads files from a user-controlled manifest by design
	if err != nil {
		return false, err
	}
	defer func() { _ = f.Close() }()
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
