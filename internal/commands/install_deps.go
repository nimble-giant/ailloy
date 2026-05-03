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
// `allowLocalDeps` controls whether mold.yaml may declare dependencies
// pointing at local filesystem paths (absolute or file:// refs). Should be
// true only when the parent mold itself was loaded from a local source —
// refusing local-path deps inside a remotely-resolved mold prevents a
// malicious foundry from exfiltrating files via the cast pipeline.
//
// Pre-collision check: ore deps with explicit aliases that resolve to the
// same install-dir name fail BEFORE any download.
func installDeclaredDeps(manifest *mold.Mold, moldKey string, global, allowLocalDeps bool) error {
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

		if !allowLocalDeps && !foundry.IsRemoteReference(ref) {
			return fmt.Errorf("dependency %q is a local path, but the mold was resolved from a remote source; refusing to copy local files into the project", ref)
		}

		// Pre-compute provenance identity (source, subpath) from the ref so
		// the skip-check below uses the same (Source, Subpath, Alias) key
		// that UpsertArtifact later writes. Avoids a network round-trip for
		// already-installed deps.
		sourceID, subpath := depIdentity(ref)

		// Skip if already installed (any version) for this (source, subpath, alias).
		// Constraint solving is deferred to issue #185 (transitive deps).
		if existing := findArtifactBySource(im, kind, sourceID, subpath, d.As); existing != nil {
			if moldKey != "" && !containsString(existing.Dependents, moldKey) {
				existing.Dependents = append(existing.Dependents, moldKey)
				_ = foundry.WriteInstalledManifest(manifestPath, im)
			}
			continue
		}

		if !castSilent.Load() {
			fmt.Println(styles.WorkingBanner(fmt.Sprintf("Installing %s %s...", kind, ref)))
		}
		fsys, resolvedSource, resolvedSubpath, version, commit, err := resolveDepFS(ref, d.Version, global)
		if err != nil {
			return fmt.Errorf("resolving %s %s: %w", kind, ref, err)
		}
		// Trust the resolver's view of (source, subpath) once we have it;
		// pre-parse can't see e.g. lock-file rewrites, but practically these
		// match for both the remote and local branches.
		sourceID = resolvedSource
		subpath = resolvedSubpath

		// Validate manifest matches kind.
		manifestName := ""
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
		baseDir, derr := artifactInstallDir(kind, installName, global)
		if derr != nil {
			return fmt.Errorf("determining install dir for %s %s: %w", kind, installName, derr)
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

		if !castSilent.Load() {
			fmt.Println(styles.SuccessStyle.Render("  Installed: ") + styles.AccentStyle.Render(manifestName+" "+version))
		}
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
// fields suitable for InstalledManifest: source (cache key for remote, path
// for local), subpath (only set for remote refs that include //subpath),
// resolved version, and resolved commit. Remote refs go through the foundry
// resolver; local paths (absolute, ./relative, or file://...) are loaded
// directly from disk so cast-time auto-install works in dev workflows
// without a network round-trip. The dep's declared `version` is reused as
// the recorded version for local refs (they have no resolved tag).
func resolveDepFS(ref, declaredVersion string, global bool) (fs.FS, string, string, string, string, error) {
	if foundry.IsRemoteReference(ref) {
		var resolveOpts []foundry.ResolveOption
		if global {
			resolveOpts = append(resolveOpts, foundry.WithLockPath(globalLockPath()))
		}
		fsys, result, err := foundry.ResolveWithMetadata(ref, resolveOpts...)
		if err != nil {
			return nil, "", "", "", "", err
		}
		return fsys, result.Ref.CacheKey(), result.Ref.Subpath, result.Resolved.Tag, result.Resolved.Commit, nil
	}

	path := strings.TrimPrefix(ref, "file://")
	info, err := os.Stat(path)
	if err != nil {
		return nil, "", "", "", "", fmt.Errorf("stat %q: %w", path, err)
	}
	if !info.IsDir() {
		return nil, "", "", "", "", fmt.Errorf("%q is not a directory", path)
	}
	return os.DirFS(path), path, "", declaredVersion, "", nil
}

// depIdentity returns the (source, subpath) identity tuple for a dependency
// reference without doing a full network/disk resolve. Used by the
// installed-manifest skip check so already-installed deps don't trigger a
// foundry round-trip just to discover they're a no-op.
//
// For remote refs, this parses the reference to extract CacheKey + Subpath.
// For local refs, source is the path and subpath is empty (matching what
// resolveDepFS reports). Parse failures fall back to (ref, "") — the
// downstream resolve will surface the real error.
func depIdentity(ref string) (string, string) {
	if foundry.IsRemoteReference(ref) {
		parsed, err := foundry.ParseReference(ref)
		if err != nil {
			return ref, ""
		}
		return parsed.CacheKey(), parsed.Subpath
	}
	return strings.TrimPrefix(ref, "file://"), ""
}

func findArtifactBySource(m *foundry.InstalledManifest, kind, source, subpath, alias string) *foundry.ArtifactEntry {
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
		e := &(*list)[i]
		if e.Source == source && e.Subpath == subpath && e.Alias == alias {
			return e
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

// pruneRemovedDeps reconciles the dependents graph for a mold whose
// declared dependency list has changed. It walks the installed manifest at
// manifestPath and, for every ingot/ore entry whose source is NOT in
// currentDeps but which still lists moldKey as a dependent, strips moldKey
// from that entry's Dependents. Entries whose Dependents become empty after
// stripping are treated as orphans: the manifest entry is dropped and the
// install directory under .ailloy/<kind>s/<install-name> is removed from
// disk.
//
// Used by recast (to drop deps the mold no longer declares) and reused in
// later phases by uninstall cascade. Entries whose source is still declared
// are left untouched even if their Dependents would otherwise change.
//
// A nil/missing manifest is a no-op (nothing to prune).
func pruneRemovedDeps(manifestPath, moldKey string, currentDeps []mold.Dependency, global bool) error {
	if manifestPath == "" || moldKey == "" {
		return nil
	}
	im, err := foundry.ReadInstalledManifest(manifestPath)
	if err != nil {
		return err
	}
	if im == nil {
		return nil
	}

	// Build the set of sources still declared by the mold (post-update).
	declaredSources := make(map[string]struct{}, len(currentDeps))
	for _, d := range currentDeps {
		if src := d.Source(); src != "" {
			declaredSources[src] = struct{}{}
		}
	}

	type prunePlan struct {
		kind        string
		installName string
	}
	var orphanPlans []prunePlan
	mutated := false

	// Walk ingots + ores. Strip moldKey from any entry whose source is no
	// longer declared. Track orphans (Dependents drained to empty) so we can
	// also drop them from disk + manifest.
	for _, kind := range []string{"ingot", "ore"} {
		var src *[]foundry.ArtifactEntry
		switch kind {
		case "ingot":
			src = &im.Ingots
		case "ore":
			src = &im.Ores
		}
		kept := (*src)[:0]
		for _, e := range *src {
			if _, declared := declaredSources[e.Source]; declared {
				kept = append(kept, e)
				continue
			}
			if !containsString(e.Dependents, moldKey) {
				kept = append(kept, e)
				continue
			}
			// Strip moldKey from this entry.
			e.Dependents = stripDependent(e.Dependents, moldKey)
			mutated = true
			if len(e.Dependents) == 0 {
				installName := e.Name
				if e.Alias != "" {
					installName = e.Alias
				}
				orphanPlans = append(orphanPlans, prunePlan{kind: kind, installName: installName})
				continue // drop from manifest
			}
			kept = append(kept, e)
		}
		*src = kept
	}

	if !mutated {
		return nil
	}

	// Remove orphan install directories. Project scope lives under
	// .ailloy/<kind>s/<name>; global lives under ~/.ailloy/<kind>s/<name>.
	for _, p := range orphanPlans {
		dir, derr := artifactInstallDir(p.kind, p.installName, global)
		if derr != nil {
			log.Printf("warning: determining install dir for %s %s: %v", p.kind, p.installName, derr)
			continue
		}
		if err := os.RemoveAll(dir); err != nil {
			log.Printf("warning: removing %s: %v", dir, err)
		}
	}

	if err := foundry.WriteInstalledManifest(manifestPath, im); err != nil {
		return fmt.Errorf("writing pruned manifest: %w", err)
	}
	return nil
}

// stripDependent returns a copy of s with every occurrence of target removed,
// or nil if no entries remain. Mirrors foundry.stripString without exporting
// it.
func stripDependent(s []string, target string) []string {
	if len(s) == 0 {
		return nil
	}
	out := make([]string, 0, len(s))
	for _, v := range s {
		if v != target {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// artifactInstallDir returns the on-disk install directory for an artifact of
// the given kind and install-name. Project scope returns ".ailloy/<kind>s/<name>";
// global scope returns "~/.ailloy/<kind>s/<name>". Returns an error if the home
// directory cannot be resolved while in global mode.
func artifactInstallDir(kind, name string, global bool) (string, error) {
	if !global {
		return filepath.Join(".ailloy", kind+"s", name), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ailloy", kind+"s", name), nil
}

// cascadeUninstallArtifacts strips moldKey from every ingot/ore Dependents
// list. Any artifact whose Dependents drains to empty is treated as an orphan:
// the install directory is removed from disk, the manifest entry is dropped,
// and the matching lock entry (if any, project-scope only) is removed.
//
// Used by uninstall after the mold's own files have been removed. The
// "user" sentinel is preserved automatically — RemoveDependent only matches
// the supplied moldKey.
func cascadeUninstallArtifacts(manifestPath, moldKey string, global bool) error {
	if manifestPath == "" || moldKey == "" {
		return nil
	}
	im, err := foundry.ReadInstalledManifest(manifestPath)
	if err != nil {
		return err
	}
	if im == nil {
		return nil
	}

	// Tag every artifact's identity with its kind up-front so we can recover
	// the kind for orphans (RemoveDependent doesn't return that info).
	type kindKey struct{ source, subpath, alias string }
	kindByKey := map[kindKey]string{}
	for _, e := range im.Ingots {
		kindByKey[kindKey{e.Source, e.Subpath, e.Alias}] = "ingot"
	}
	for _, e := range im.Ores {
		kindByKey[kindKey{e.Source, e.Subpath, e.Alias}] = "ore"
	}

	orphans := im.RemoveDependent(moldKey)

	for _, o := range orphans {
		kind, ok := kindByKey[kindKey{o.Source, o.Subpath, o.Alias}]
		if !ok {
			continue
		}
		installName := o.Name
		if o.Alias != "" {
			installName = o.Alias
		}
		baseDir, derr := artifactInstallDir(kind, installName, global)
		if derr != nil {
			log.Printf("warning: determining install dir for orphaned %s %s: %v", kind, installName, derr)
			continue
		}
		if err := os.RemoveAll(baseDir); err != nil {
			log.Printf("warning: removing %s: %v", baseDir, err)
		}
		if !castSilent.Load() {
			fmt.Println(styles.SuccessStyle.Render("  Cascade-removed: ") + styles.AccentStyle.Render(kind+" "+installName))
		}
	}

	if err := foundry.WriteInstalledManifest(manifestPath, im); err != nil {
		return fmt.Errorf("writing manifest after cascade: %w", err)
	}

	// Drop orphans from the lock as well (project-scope only — global has no
	// lock convention today).
	if !global && len(orphans) > 0 {
		if lock, _ := foundry.ReadLockFile(foundry.LockFileName); lock != nil {
			for _, o := range orphans {
				kind, ok := kindByKey[kindKey{o.Source, o.Subpath, o.Alias}]
				if !ok {
					continue
				}
				installName := o.Name
				if o.Alias != "" {
					installName = o.Alias
				}
				dropArtifactLockEntry(lock, kind, installName)
			}
			_ = foundry.WriteLockFile(foundry.LockFileName, lock)
		}
	}

	return nil
}

// dropArtifactLockEntry removes a lock entry for the given (kind, install-name).
// install-name is the alias-applied name (the on-disk dir name).
func dropArtifactLockEntry(lock *foundry.LockFile, kind, name string) {
	if lock == nil {
		return
	}
	var list *[]foundry.LockEntry
	switch kind {
	case "ingot":
		list = &lock.Ingots
	case "ore":
		list = &lock.Ores
	default:
		return
	}
	kept := (*list)[:0]
	for _, e := range *list {
		effective := e.Name
		if e.Alias != "" {
			effective = e.Alias
		}
		if effective != name {
			kept = append(kept, e)
		}
	}
	*list = kept
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
