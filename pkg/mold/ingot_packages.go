package mold

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// IngotPackage describes one ingot package discovered inside an fs.FS. Root is
// the package's path within the FS (".", or "ingots/<name>"). Subpath is the
// "//subpath" component used for refs and lock identity — empty for the
// single-at-root layout, "ingots/<name>" for multi-package layout.
type IngotPackage struct {
	Name    string
	Version string
	Root    string
	Subpath string
	Ingot   *Ingot
}

// DiscoverIngotPackages returns every ingot package present in fsys. Layout
// rules (mirrors ore PR #192):
//
//   - If a top-level ingot.yaml exists, return a single package rooted at "."
//     with empty Subpath. (Single-at-root layout.)
//   - Else if "ingots/" exists, scan each subdirectory <name> for
//     <name>/ingot.yaml. Return one package per subdir; Root="ingots/<name>",
//     Subpath="ingots/<name>". Subdirectories without an ingot.yaml are skipped.
//   - Else: empty result, no error. The caller decides whether emptiness is
//     fatal (e.g. `ailloy ingot add` should reject; temper does too).
//
// Manifest parse errors are returned as errors — a partially-valid multi-ingot
// repo should not silently succeed.
func DiscoverIngotPackages(fsys fs.FS) ([]IngotPackage, error) {
	if _, err := fs.Stat(fsys, "ingot.yaml"); err == nil {
		ingot, perr := LoadIngotFromFS(fsys, "ingot.yaml")
		if perr != nil {
			return nil, fmt.Errorf("loading root ingot manifest: %w", perr)
		}
		return []IngotPackage{{
			Name:    ingot.Name,
			Version: ingot.Version,
			Root:    ".",
			Subpath: "",
			Ingot:   ingot,
		}}, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("stat ingot.yaml: %w", err)
	}

	entries, err := fs.ReadDir(fsys, "ingots")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading ingots/: %w", err)
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	var pkgs []IngotPackage
	for _, name := range names {
		root := path.Join("ingots", name)
		manifestPath := path.Join(root, "ingot.yaml")
		ingot, perr := LoadIngotFromFS(fsys, manifestPath)
		if perr != nil {
			if errors.Is(perr, fs.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("loading ingot at %s: %w", manifestPath, perr)
		}
		pkgs = append(pkgs, IngotPackage{
			Name:    ingot.Name,
			Version: ingot.Version,
			Root:    root,
			Subpath: root,
			Ingot:   ingot,
		})
	}
	return pkgs, nil
}
