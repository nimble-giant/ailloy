package commands

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nimble-giant/ailloy/pkg/blanks"
	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/mold"
	"github.com/nimble-giant/ailloy/pkg/styles"
)

// installDeclaredDeps walks manifest.Dependencies, installs any missing
// ingot/ore into the appropriate scope (project or global), and updates
// installed.yaml + ailloy.lock with provenance + Dependents += moldKey.
//
// `moldKey` is the requesting mold's source@subpath — used as the
// Dependents entry. May be "" for local-path molds (no dependents tracking
// happens then; the install still proceeds).
//
// Pre-collision check: ore deps with explicit aliases that resolve to the
// same install-dir name fail BEFORE any download.
func installDeclaredDeps(manifest *mold.Mold, moldKey string, global bool) error {
	if manifest == nil || len(manifest.Dependencies) == 0 {
		return nil
	}

	// Pre-validate kinds and pre-collide explicit aliases.
	oreNames := map[string]string{} // installDirName -> dep.Source
	for _, d := range manifest.Dependencies {
		kind, kerr := d.Kind()
		if kerr != nil {
			return fmt.Errorf("invalid dependency: %w", kerr)
		}
		if kind != "ore" {
			continue
		}
		if d.As != "" {
			if prev, dup := oreNames[d.As]; dup {
				return fmt.Errorf("ore install-dir collision on alias %q: %s and %s; resolve with distinct 'as:' values", d.As, prev, d.Source())
			}
			oreNames[d.As] = d.Source()
		}
	}

	manifestPath := manifestPathFor(global)
	im, _ := foundry.ReadInstalledManifest(manifestPath)
	if im == nil {
		im = &foundry.InstalledManifest{APIVersion: "v1"}
	}

	for _, d := range manifest.Dependencies {
		kind, _ := d.Kind() // already validated above
		ref := d.Source()

		// Skip if already installed (any version) for this source.
		// Constraint solving is deferred to issue #185 (transitive deps).
		if existing := findArtifactBySource(im, kind, ref); existing != nil {
			if moldKey != "" && !containsString(existing.Dependents, moldKey) {
				existing.Dependents = append(existing.Dependents, moldKey)
				_ = foundry.WriteInstalledManifest(manifestPath, im)
			}
			continue
		}

		fmt.Println(styles.WorkingBanner(fmt.Sprintf("Installing %s %s...", kind, ref)))
		fsys, sourceID, version, commit, err := resolveDepFS(ref, d.Version, global)
		if err != nil {
			return fmt.Errorf("resolving %s %s: %w", kind, ref, err)
		}

		// Validate manifest matches kind.
		manifestName := ""
		subpath := ""
		switch kind {
		case "ingot":
			ingot, ierr := mold.LoadIngotFromFS(fsys, "ingot.yaml")
			if ierr != nil {
				return fmt.Errorf("invalid ingot manifest at %s: %w", ref, ierr)
			}
			manifestName = ingot.Name
		case "ore":
			ore, oerr := mold.LoadOreFromFS(fsys, "ore.yaml")
			if oerr != nil {
				return fmt.Errorf("invalid ore manifest at %s: %w", ref, oerr)
			}
			if ore.Kind != "ore" {
				return fmt.Errorf("manifest at %s has kind=%q, expected 'ore'", ref, ore.Kind)
			}
			manifestName = ore.Name
		}

		// Install-dir name resolution (alias if set, else manifest name).
		installName := manifestName
		if d.As != "" {
			installName = d.As
		}
		// Late collision check for ores: net-new vs already-noted alias / vs another net-new.
		if kind == "ore" {
			if prev, dup := oreNames[installName]; dup && prev != ref {
				return fmt.Errorf("ore install-dir collision on name %q: %s and %s; resolve with 'as:'", installName, prev, ref)
			}
			oreNames[installName] = ref
		}

		// Compute install dir.
		baseDir := filepath.Join(".ailloy", kind+"s", installName)
		if global {
			home, herr := os.UserHomeDir()
			if herr != nil {
				return fmt.Errorf("determining home directory: %w", herr)
			}
			baseDir = filepath.Join(home, ".ailloy", kind+"s", installName)
		}
		if err := copyFromFS(fsys, baseDir); err != nil {
			return fmt.Errorf("copying %s to %s: %w", kind, baseDir, err)
		}

		// Record in manifest.
		entry := foundry.ArtifactEntry{
			Name:        manifestName,
			Source:      sourceID,
			Subpath:     subpath,
			Version:     version,
			Commit:      commit,
			InstalledAt: time.Now().UTC(),
		}
		if d.As != "" {
			entry.Alias = d.As
		}
		if moldKey != "" {
			entry.Dependents = []string{moldKey}
		}
		im.UpsertArtifact(kind, entry)

		fmt.Println(styles.SuccessStyle.Render("  Installed: ") + styles.AccentStyle.Render(manifestName+" "+version))
	}

	if err := foundry.WriteInstalledManifest(manifestPath, im); err != nil {
		log.Printf("warning: writing installed manifest: %v", err)
	}

	// Update ailloy.lock if it exists (project-scope only).
	if !global {
		if lock, _ := foundry.ReadLockFile(foundry.LockFileName); lock != nil {
			for _, e := range im.Ingots {
				lock.UpsertArtifactLock("ingot", artifactToLock(e))
			}
			for _, e := range im.Ores {
				lock.UpsertArtifactLock("ore", artifactToLock(e))
			}
			_ = foundry.WriteLockFile(foundry.LockFileName, lock)
		}
	}

	return nil
}

// resolveDepFS returns an fs.FS for an ore/ingot dep along with provenance
// fields suitable for InstalledManifest. Remote refs go through the foundry
// resolver; local paths (absolute, ./relative, or file://...) are loaded
// directly from disk so cast-time auto-install works in dev workflows
// without a network round-trip. The dep's declared `version` is reused as
// the recorded version for local refs (they have no resolved tag).
func resolveDepFS(ref, declaredVersion string, global bool) (fs.FS, string, string, string, error) {
	if foundry.IsRemoteReference(ref) {
		var resolveOpts []foundry.ResolveOption
		if global {
			resolveOpts = append(resolveOpts, foundry.WithLockPath(globalLockPath()))
		}
		fsys, result, err := foundry.ResolveWithMetadata(ref, resolveOpts...)
		if err != nil {
			return nil, "", "", "", err
		}
		return fsys, result.Ref.CacheKey(), result.Resolved.Tag, result.Resolved.Commit, nil
	}

	path := strings.TrimPrefix(ref, "file://")
	info, err := os.Stat(path)
	if err != nil {
		return nil, "", "", "", fmt.Errorf("stat %q: %w", path, err)
	}
	if !info.IsDir() {
		return nil, "", "", "", fmt.Errorf("%q is not a directory", path)
	}
	return os.DirFS(path), path, declaredVersion, "", nil
}

func findArtifactBySource(m *foundry.InstalledManifest, kind, source string) *foundry.ArtifactEntry {
	if m == nil {
		return nil
	}
	var list *[]foundry.ArtifactEntry
	switch kind {
	case "ingot":
		list = &m.Ingots
	case "ore":
		list = &m.Ores
	default:
		return nil
	}
	for i := range *list {
		if (*list)[i].Source == source {
			return &(*list)[i]
		}
	}
	return nil
}

func containsString(s []string, target string) bool {
	for _, v := range s {
		if v == target {
			return true
		}
	}
	return false
}

func copyFromFS(fsys fs.FS, dest string) error {
	if err := os.MkdirAll(dest, 0750); err != nil { // #nosec G301
		return err
	}
	return fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		out := filepath.Join(dest, path)
		if d.IsDir() {
			return os.MkdirAll(out, 0750) // #nosec G301
		}
		body, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		return os.WriteFile(out, body, 0644) // #nosec G306
	})
}

func artifactToLock(e foundry.ArtifactEntry) foundry.LockEntry {
	return foundry.LockEntry{
		Name:      e.Name,
		Source:    e.Source,
		Subpath:   e.Subpath,
		Version:   e.Version,
		Commit:    e.Commit,
		Alias:     e.Alias,
		Timestamp: e.InstalledAt,
	}
}

// buildOreSearchPaths returns the ore-search-path order for cast-time merge:
// mold-local → project (./.ailloy/ores) → global (~/.ailloy/ores). Lower
// priority paths only contribute ore namespaces not already seen, mirroring
// how flux defaults are layered.
func buildOreSearchPaths(moldFS fs.FS, global bool) []mold.OreSearchPath {
	paths := []mold.OreSearchPath{
		{Name: "mold-local", FS: moldFS, Root: "ores"},
	}
	// Project scope is meaningful even on a global cast — the user may have
	// project-installed ores they want to layer in. Global cast users who
	// want strict isolation can run from a clean cwd.
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, mold.OreSearchPath{
			Name: "project",
			FS:   os.DirFS(cwd),
			Root: ".ailloy/ores",
		})
	}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, mold.OreSearchPath{
			Name: "global",
			FS:   os.DirFS(home),
			Root: ".ailloy/ores",
		})
	}
	_ = global // currently only affects install-dir, not search-path order
	return paths
}

// readerSearchPaths is a thin convenience wrapper for callers that have a
// MoldReader in hand (cast.go / cast_core.go). It threads the reader's FS
// into buildOreSearchPaths so mold-local ores under <mold>/ores/ are picked
// up first.
func readerSearchPaths(reader *blanks.MoldReader, global bool) []mold.OreSearchPath {
	if reader == nil {
		return buildOreSearchPaths(nil, global)
	}
	return buildOreSearchPaths(reader.FS(), global)
}
