// Package projectstate locates ailloy's per-project bookkeeping.
//
// Ailloy tracks what it cast into a project (installed manifest, lock file,
// persisted flux, installed ores/ingots, install state). None of that is part
// of a mold's distributable output, so it lives outside the project tree, in
// the user's global ailloy directory:
//
//	~/.ailloy/projects/<project-name>-<hash>/
//
// <hash> is derived from the project's absolute path, so each checkout gets
// its own state dir and the same checkout always maps to the same one. A
// project.yaml file inside records the project root so the dirs stay
// discoverable (`ailloy state list`).
//
// Set AILLOY_PROJECT_STATE_DIR to override the location. A relative value
// resolves against the project root, so ".ailloy" keeps all state (lock
// included, as .ailloy/ailloy.lock) inside the project, for teams that want
// it committed.
package projectstate

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/goccy/go-yaml"
)

const (
	// EnvDir overrides the per-project state directory.
	EnvDir = "AILLOY_PROJECT_STATE_DIR"
	// LegacyDirName is the in-project directory ailloy used before state
	// moved to the user's global ailloy directory.
	LegacyDirName = ".ailloy"
	// LegacyLockFileName is the lock file ailloy used to write at the
	// project root.
	LegacyLockFileName = "ailloy.lock"

	projectsSubdir = "projects"
	metaFileName   = "project.yaml"
)

// legacyItems are the bookkeeping entries under the legacy .ailloy/ dir that
// MigrateLegacy moves. Anything else in .ailloy/ is left alone.
var legacyItems = []string{"installed.yaml", "state.yaml", "flux", "ores", "ingots"}

// Dir returns the state dir for the project rooted at the current working
// directory.
func Dir() string { return DirFor(".") }

// DirFor returns the state dir for the project rooted at projectRoot. When
// the home directory cannot be resolved it falls back to the legacy
// in-project dir so ailloy keeps working.
func DirFor(projectRoot string) string {
	if v := os.Getenv(EnvDir); v != "" {
		if filepath.IsAbs(v) {
			return v
		}
		return filepath.Join(projectRoot, v)
	}
	base, err := ProjectsRoot()
	if err != nil {
		return filepath.Join(projectRoot, LegacyDirName)
	}
	return filepath.Join(base, Key(canonical(projectRoot)))
}

// ProjectsRoot returns ~/.ailloy/projects, the parent of every per-project
// state dir.
func ProjectsRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, LegacyDirName, projectsSubdir), nil
}

// Key returns the state-dir name for an absolute project root: a readable
// project name plus a short hash of the full path for uniqueness.
func Key(absRoot string) string {
	sum := sha256.Sum256([]byte(absRoot))
	return sanitize(filepath.Base(absRoot)) + "-" + hex.EncodeToString(sum[:6])
}

func sanitize(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	s := strings.Trim(b.String(), ".")
	if s == "" {
		return "project"
	}
	return s
}

// canonical returns the absolute, symlink-resolved form of path so the same
// checkout always hashes to the same key.
func canonical(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	if real, err := filepath.EvalSymlinks(abs); err == nil {
		return real
	}
	return abs
}

type meta struct {
	Root string `yaml:"root"`
}

// Record writes project.yaml (the project root) into the project's state dir
// when that dir exists. It never creates a state dir on its own.
func Record(projectRoot string) error {
	dir := DirFor(projectRoot)
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return nil
	}
	return writeMeta(dir, canonical(projectRoot))
}

func writeMeta(dir, root string) error {
	path := filepath.Join(dir, metaFileName)
	if existing, ok := readMeta(dir); ok && existing == root {
		return nil
	}
	data, err := yaml.Marshal(meta{Root: root})
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644) // #nosec G306 -- non-secret bookkeeping
}

func readMeta(dir string) (string, bool) {
	data, err := os.ReadFile(filepath.Join(dir, metaFileName)) // #nosec G304 -- fixed name under the state dir
	if err != nil {
		return "", false
	}
	var m meta
	if err := yaml.Unmarshal(data, &m); err != nil || m.Root == "" {
		return "", false
	}
	return m.Root, true
}

// RootFor returns the project root recorded in stateDir's project.yaml.
func RootFor(stateDir string) (string, bool) { return readMeta(stateDir) }

// Project is one known per-project state dir.
type Project struct {
	Root string // project root recorded in project.yaml ("" if unknown)
	Dir  string // the state dir
}

// List returns every per-project state dir under ProjectsRoot, sorted by root.
func List() ([]Project, error) {
	base, err := ProjectsRoot()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(base)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var out []Project
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(base, e.Name())
		root, _ := readMeta(dir)
		out = append(out, Project{Root: root, Dir: dir})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Root != out[j].Root {
			return out[i].Root < out[j].Root
		}
		return out[i].Dir < out[j].Dir
	})
	return out, nil
}

// MigrationResult reports what MigrateLegacy did, as project-relative paths.
type MigrationResult struct {
	Dir       string   // destination state dir
	Moved     []string // legacy paths moved into Dir
	Conflicts []string // legacy paths left in place because Dir already had them
}

// MigrateLegacy moves ailloy bookkeeping out of the project tree
// (.ailloy/{installed.yaml,state.yaml,flux,ores,ingots} and ailloy.lock) into
// the project's state dir. Existing destination entries are never
// overwritten; those legacy entries stay put and are reported as conflicts.
// The legacy .ailloy/ dir is removed once empty. Entries already at their
// destination (AILLOY_PROJECT_STATE_DIR=.ailloy) are skipped.
func MigrateLegacy(projectRoot string) (MigrationResult, error) {
	dest := DirFor(projectRoot)
	res := MigrationResult{Dir: dest}
	legacyDir := filepath.Join(projectRoot, LegacyDirName)
	// In the home directory, ~/.ailloy and ~/ailloy.lock are the GLOBAL
	// state, not a legacy project layout. Never migrate them.
	if home, err := os.UserHomeDir(); err == nil && canonical(home) == canonical(projectRoot) {
		return res, nil
	}

	type item struct{ rel, src, dst string }
	var items []item
	for _, name := range legacyItems {
		items = append(items, item{
			rel: filepath.Join(LegacyDirName, name),
			src: filepath.Join(legacyDir, name),
			dst: filepath.Join(dest, name),
		})
	}
	items = append(items, item{
		rel: LegacyLockFileName,
		src: filepath.Join(projectRoot, LegacyLockFileName),
		dst: filepath.Join(dest, LegacyLockFileName),
	})

	for _, it := range items {
		if _, err := os.Lstat(it.src); err != nil {
			continue
		}
		if canonical(it.src) == canonical(it.dst) {
			continue
		}
		if _, err := os.Lstat(it.dst); err == nil {
			res.Conflicts = append(res.Conflicts, it.rel)
			continue
		}
		if err := os.MkdirAll(dest, 0o750); err != nil { // #nosec G301
			return res, fmt.Errorf("creating state dir: %w", err)
		}
		if err := move(it.src, it.dst); err != nil {
			return res, fmt.Errorf("moving %s: %w", it.rel, err)
		}
		res.Moved = append(res.Moved, it.rel)
	}

	if len(res.Moved) > 0 {
		if err := writeMeta(dest, canonical(projectRoot)); err != nil {
			return res, fmt.Errorf("recording project root: %w", err)
		}
		_ = os.Remove(legacyDir) // only succeeds when empty
	}
	return res, nil
}

// move renames src to dst, falling back to copy+remove across filesystems.
func move(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if err := copyTree(src, dst); err != nil {
		_ = os.RemoveAll(dst)
		return err
	}
	return os.RemoveAll(src)
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o750) // #nosec G301
		}
		if !d.Type().IsRegular() {
			return nil
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src) // #nosec G304 -- migrating ailloy's own files
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_EXCL, info.Mode().Perm()) // #nosec G304
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
