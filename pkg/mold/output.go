package mold

import (
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// reservedDirs are top-level directories excluded from auto-discovery
// when output is absent or uses the string (parent) form.
var reservedDirs = map[string]bool{
	"ingots": true,
}

// dirMapping represents a normalized directory-to-directory output mapping.
type dirMapping struct {
	src    string // source directory in the mold fs
	target OutputTarget
}

// fileMapping represents a normalized single-file output mapping.
type fileMapping struct {
	src     string // source file path in the mold fs
	dest    string // destination file path
	process bool
}

// ResolveFiles walks the mold filesystem and resolves all files according
// to the output mapping.
//
// If output is nil, all top-level directories (excluding reserved names
// like "ingots") are walked with identity mapping (src path = dest path).
//
// If output is a string, it's treated as a parent directory — all top-level
// directories are mapped under it.
//
// If output is a map, each entry maps a source directory or file to a
// destination. Values can be strings (simple dest path) or maps with
// "dest" and optional "process" fields.
func ResolveFiles(output any, moldFS fs.FS) ([]ResolvedFile, error) {
	if output == nil {
		return resolveIdentity(moldFS)
	}

	dirs, files, err := parseOutput(output, moldFS)
	if err != nil {
		return nil, fmt.Errorf("parsing output mapping: %w", err)
	}

	return resolveFromMappings(dirs, files, moldFS)
}

// resolveIdentity walks all top-level directories (excluding reserved ones)
// and returns files with identity mapping (src = dest, process = true).
func resolveIdentity(moldFS fs.FS) ([]ResolvedFile, error) {
	topDirs, err := discoverTopLevelDirs(moldFS)
	if err != nil {
		return nil, err
	}

	var dirs []dirMapping
	for _, d := range topDirs {
		dirs = append(dirs, dirMapping{
			src:    d,
			target: OutputTarget{Dest: d},
		})
	}

	return resolveFromMappings(dirs, nil, moldFS)
}

// discoverTopLevelDirs returns all non-reserved top-level directories in the mold.
func discoverTopLevelDirs(moldFS fs.FS) ([]string, error) {
	entries, err := fs.ReadDir(moldFS, ".")
	if err != nil {
		return nil, fmt.Errorf("reading mold root: %w", err)
	}

	var dirs []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if reservedDirs[name] || strings.HasPrefix(name, ".") {
			continue
		}
		dirs = append(dirs, name)
	}
	return dirs, nil
}

// parseOutput normalizes the raw output YAML value into directory and file mappings.
func parseOutput(raw any, moldFS fs.FS) ([]dirMapping, []fileMapping, error) {
	switch v := raw.(type) {
	case string:
		return parseStringOutput(v, moldFS)
	case map[string]any:
		return parseMapOutput(v, moldFS)
	default:
		return nil, nil, fmt.Errorf("output must be a string or map, got %T", raw)
	}
}

// parseStringOutput handles `output: .claude` — all top-level dirs go under the parent.
func parseStringOutput(parent string, moldFS fs.FS) ([]dirMapping, []fileMapping, error) {
	topDirs, err := discoverTopLevelDirs(moldFS)
	if err != nil {
		return nil, nil, err
	}

	var dirs []dirMapping
	for _, d := range topDirs {
		dirs = append(dirs, dirMapping{
			src:    d,
			target: OutputTarget{Dest: path.Join(parent, d)},
		})
	}
	return dirs, nil, nil
}

// parseMapOutput handles the map form of output.
func parseMapOutput(m map[string]any, moldFS fs.FS) ([]dirMapping, []fileMapping, error) {
	var dirs []dirMapping
	var files []fileMapping

	for src, val := range m {
		isDir := isDirectory(moldFS, src)

		target, err := parseOutputValue(val)
		if err != nil {
			return nil, nil, fmt.Errorf("output key %q: %w", src, err)
		}

		if isDir {
			dirs = append(dirs, dirMapping{src: src, target: target})
		} else {
			files = append(files, fileMapping{
				src:     src,
				dest:    target.Dest,
				process: target.ShouldProcess(),
			})
		}
	}

	// Sort for deterministic output.
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].src < dirs[j].src })
	sort.Slice(files, func(i, j int) bool { return files[i].src < files[j].src })

	return dirs, files, nil
}

// parseOutputValue normalizes a single output value (string or map).
func parseOutputValue(val any) (OutputTarget, error) {
	switch v := val.(type) {
	case string:
		return OutputTarget{Dest: v}, nil
	case map[string]any:
		t := OutputTarget{}
		if dest, ok := v["dest"]; ok {
			d, ok := dest.(string)
			if !ok {
				return t, fmt.Errorf("dest must be a string")
			}
			t.Dest = d
		}
		if proc, ok := v["process"]; ok {
			b, ok := proc.(bool)
			if !ok {
				return t, fmt.Errorf("process must be a boolean")
			}
			t.Process = &b
		}
		return t, nil
	default:
		return OutputTarget{}, fmt.Errorf("value must be a string or map, got %T", val)
	}
}

// resolveFromMappings walks directories and applies mappings to produce resolved files.
func resolveFromMappings(dirs []dirMapping, files []fileMapping, moldFS fs.FS) ([]ResolvedFile, error) {
	// Build file override set for quick lookup.
	fileOverrides := make(map[string]fileMapping)
	for _, f := range files {
		fileOverrides[f.src] = f
	}

	var resolved []ResolvedFile

	// Walk each mapped directory.
	for _, dm := range dirs {
		err := fs.WalkDir(moldFS, dm.src, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}

			// Check for file-level override.
			if fo, ok := fileOverrides[p]; ok {
				resolved = append(resolved, ResolvedFile{
					SrcPath:  p,
					DestPath: fo.dest,
					Process:  fo.process,
				})
				delete(fileOverrides, p) // consumed
				return nil
			}

			// Apply directory mapping: replace source prefix with dest prefix.
			rel := strings.TrimPrefix(p, dm.src)
			rel = strings.TrimPrefix(rel, "/")
			destPath := path.Join(dm.target.Dest, rel)

			resolved = append(resolved, ResolvedFile{
				SrcPath:  p,
				DestPath: destPath,
				Process:  dm.target.ShouldProcess(),
			})
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("walking source directory %q: %w", dm.src, err)
		}
	}

	// Add remaining file-level mappings (files not inside any mapped directory).
	for _, f := range files {
		if _, consumed := fileOverrides[f.src]; !consumed {
			continue
		}
		// Read the file to verify it exists.
		if _, err := fs.Stat(moldFS, f.src); err != nil {
			return nil, fmt.Errorf("file mapping %q: %w", f.src, err)
		}
		resolved = append(resolved, ResolvedFile{
			SrcPath:  f.src,
			DestPath: f.dest,
			Process:  f.process,
		})
	}

	// Sort for deterministic output.
	sort.Slice(resolved, func(i, j int) bool {
		return resolved[i].SrcPath < resolved[j].SrcPath
	})

	return resolved, nil
}

// isDirectory checks if a path is a directory in the given filesystem.
func isDirectory(moldFS fs.FS, name string) bool {
	info, err := fs.Stat(moldFS, name)
	if err != nil {
		return false
	}
	return info.IsDir()
}
