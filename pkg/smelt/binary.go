package smelt

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/knadh/stuffbin"
	"github.com/nimble-giant/ailloy/pkg/mold"
	"github.com/nimble-giant/ailloy/pkg/safepath"
	"golang.org/x/sync/errgroup"
)

// PackageBinary packages a mold into a self-contained binary by collecting
// all mold files and appending them to the current ailloy binary using stuffbin.
// The output binary can be distributed and run directly: ./my-mold cast.
func PackageBinary(moldDir, outputDir string) (string, int64, error) {
	cleanDir, err := safepath.Clean(moldDir)
	if err != nil {
		return "", 0, fmt.Errorf("invalid mold directory: %w", err)
	}

	moldPath := filepath.Join(cleanDir, "mold.yaml")
	m, err := mold.LoadMold(moldPath)
	if err != nil {
		return "", 0, fmt.Errorf("loading mold: %w", err)
	}

	if err := mold.ValidateMold(m); err != nil {
		return "", 0, fmt.Errorf("validating mold: %w", err)
	}

	moldFS := os.DirFS(cleanDir)

	// Collect files to include in the binary.
	files, hasFluxYAML, err := collectMoldFiles(moldFS, cleanDir)
	if err != nil {
		return "", 0, fmt.Errorf("collecting files: %w", err)
	}

	// Generate flux.yaml defaults only if no source flux.yaml was found.
	if !hasFluxYAML {
		fluxData, err := generateFluxDefaults(m.Flux)
		if err != nil {
			return "", 0, fmt.Errorf("generating flux defaults: %w", err)
		}
		if fluxData != nil {
			files = append(files, archiveFile{path: "flux.yaml", data: fluxData})
		}
	}

	// Write collected files to a temp staging directory in parallel.
	stagingDir, err := os.MkdirTemp("", "ailloy-smelt-*")
	if err != nil {
		return "", 0, fmt.Errorf("creating staging directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(stagingDir) }()

	stuffPaths, err := stageFiles(stagingDir, files)
	if err != nil {
		return "", 0, fmt.Errorf("staging files: %w", err)
	}

	// Resolve current executable.
	execPath, err := os.Executable()
	if err != nil {
		return "", 0, fmt.Errorf("resolving executable: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return "", 0, fmt.Errorf("resolving executable symlinks: %w", err)
	}

	// Determine output path.
	outputName := fmt.Sprintf("%s-%s", m.Name, m.Version)
	if outputDir == "" {
		outputDir = "."
	}
	outputPath := filepath.Join(outputDir, outputName)

	if err := os.MkdirAll(filepath.Dir(outputPath), 0750); err != nil { // #nosec G301
		return "", 0, fmt.Errorf("creating output directory: %w", err)
	}

	// Stuff the binary with mold files using alias format for clean zip paths.
	_, _, err = stuffbin.Stuff(execPath, outputPath, "/", stuffPaths...)
	if err != nil {
		return "", 0, fmt.Errorf("stuffing binary: %w", err)
	}

	// Make output executable.
	if err := os.Chmod(outputPath, 0755); err != nil { // #nosec G302 -- binary must be executable
		return "", 0, fmt.Errorf("making binary executable: %w", err)
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		return "", 0, fmt.Errorf("stating output: %w", err)
	}

	return outputPath, info.Size(), nil
}

// stageFiles writes archiveFiles to a staging directory in parallel using
// goroutines. Returns stuffbin alias-format paths ("disk-path:/zip-path").
func stageFiles(stagingDir string, files []archiveFile) ([]string, error) {
	stuffPaths := make([]string, len(files))
	for i, f := range files {
		diskPath := filepath.Join(stagingDir, f.path)
		stuffPaths[i] = diskPath + ":/" + f.path
	}

	// Create all needed directories first (sequential to avoid races on mkdirall).
	dirs := make(map[string]bool)
	for _, f := range files {
		dir := filepath.Dir(filepath.Join(stagingDir, f.path))
		dirs[dir] = true
	}
	for dir := range dirs {
		if err := os.MkdirAll(dir, 0750); err != nil { // #nosec G301
			return nil, fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}

	// Write files in parallel.
	var g errgroup.Group
	for _, f := range files {
		g.Go(func() error {
			dest := filepath.Join(stagingDir, f.path)
			//#nosec G306 -- staging files are temporary
			return os.WriteFile(dest, f.data, 0644)
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return stuffPaths, nil
}

// UnstuffFS extracts the stuffed mold files from a binary and returns an fs.FS.
func UnstuffFS(binPath string) (fs.FS, error) {
	sfs, err := stuffbin.UnStuff(binPath)
	if err != nil {
		return nil, err
	}
	return NewStuffFS(sfs), nil
}
