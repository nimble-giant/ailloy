package smelt

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/nimble-giant/ailloy/pkg/mold"
	"github.com/nimble-giant/ailloy/pkg/safepath"
	"gopkg.in/yaml.v3"
)

// PackageTarball packages a mold directory into a .tar.gz archive.
// It validates the mold, collects all referenced files, includes or generates a
// flux.yaml defaults file, and writes the archive to outputDir (or the current
// directory if outputDir is empty). Returns the output file path and size.
func PackageTarball(moldDir, outputDir string) (string, int64, error) {
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

	// Determine output path
	archiveName := fmt.Sprintf("%s-%s.tar.gz", m.Name, m.Version)
	if outputDir == "" {
		outputDir = "."
	}
	outputPath := filepath.Join(outputDir, archiveName)

	// Collect files to include in the archive
	files, hasFluxYAML, err := collectMoldFiles(m, moldFS, cleanDir)
	if err != nil {
		return "", 0, fmt.Errorf("collecting files: %w", err)
	}

	// Generate flux.yaml defaults only if no source flux.yaml was found
	var fluxData []byte
	if !hasFluxYAML {
		fluxData, err = generateFluxDefaults(m.Flux)
		if err != nil {
			return "", 0, fmt.Errorf("generating flux defaults: %w", err)
		}
	}

	// Create the archive
	prefix := fmt.Sprintf("%s-%s", m.Name, m.Version)
	size, err := writeTarGz(outputPath, prefix, files, fluxData)
	if err != nil {
		return "", 0, fmt.Errorf("writing archive: %w", err)
	}

	return outputPath, size, nil
}

// archiveFile represents a file to include in the tarball.
type archiveFile struct {
	// path is the relative path within the archive (after prefix).
	path string
	// data is the file content.
	data []byte
}

// collectMoldFiles gathers all files referenced by the mold manifest.
// Returns the collected files and whether a source flux.yaml was found.
func collectMoldFiles(m *mold.Mold, moldFS fs.FS, moldDir string) ([]archiveFile, bool, error) {
	var files []archiveFile

	// Include mold.yaml itself
	moldYAML, err := fs.ReadFile(moldFS, "mold.yaml")
	if err != nil {
		return nil, false, fmt.Errorf("reading mold.yaml: %w", err)
	}
	files = append(files, archiveFile{path: "mold.yaml", data: moldYAML})

	// Include flux.yaml if present
	hasFluxYAML := false
	if fluxData, err := fs.ReadFile(moldFS, "flux.yaml"); err == nil {
		files = append(files, archiveFile{path: "flux.yaml", data: fluxData})
		hasFluxYAML = true
	}

	// Include flux.schema.yaml if present
	if schemaData, err := fs.ReadFile(moldFS, "flux.schema.yaml"); err == nil {
		files = append(files, archiveFile{path: "flux.schema.yaml", data: schemaData})
	}

	// Collect command templates
	for _, cmd := range m.Commands {
		relPath := filepath.Join(".claude", "commands", cmd)
		data, err := fs.ReadFile(moldFS, relPath)
		if err != nil {
			return nil, false, fmt.Errorf("reading command %s: %w", cmd, err)
		}
		files = append(files, archiveFile{path: relPath, data: data})
	}

	// Collect skill templates
	for _, skill := range m.Skills {
		relPath := filepath.Join(".claude", "skills", skill)
		data, err := fs.ReadFile(moldFS, relPath)
		if err != nil {
			return nil, false, fmt.Errorf("reading skill %s: %w", skill, err)
		}
		files = append(files, archiveFile{path: relPath, data: data})
	}

	// Collect workflow templates
	for _, wf := range m.Workflows {
		relPath := filepath.Join(".github", "workflows", wf)
		data, err := fs.ReadFile(moldFS, relPath)
		if err != nil {
			return nil, false, fmt.Errorf("reading workflow %s: %w", wf, err)
		}
		files = append(files, archiveFile{path: relPath, data: data})
	}

	// Collect ingots directory if present
	ingotFiles, err := collectIngots(moldFS, moldDir)
	if err != nil {
		return nil, false, err
	}
	files = append(files, ingotFiles...)

	return files, hasFluxYAML, nil
}

// collectIngots walks the ingots/ directory (if it exists) and collects all files.
func collectIngots(moldFS fs.FS, _ string) ([]archiveFile, error) {
	var files []archiveFile

	// Check if ingots directory exists
	if _, err := fs.Stat(moldFS, "ingots"); err != nil {
		return nil, nil // No ingots directory, that's fine
	}

	err := fs.WalkDir(moldFS, "ingots", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := fs.ReadFile(moldFS, path)
		if err != nil {
			return fmt.Errorf("reading ingot file %s: %w", path, err)
		}
		files = append(files, archiveFile{path: path, data: data})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking ingots directory: %w", err)
	}

	return files, nil
}

// generateFluxDefaults creates a flux.yaml containing default values from the
// mold's flux variable declarations.
func generateFluxDefaults(fluxVars []mold.FluxVar) ([]byte, error) {
	if len(fluxVars) == 0 {
		return nil, nil
	}

	defaults := make(map[string]string)
	for _, fv := range fluxVars {
		if fv.Default != "" {
			defaults[fv.Name] = fv.Default
		}
	}

	if len(defaults) == 0 {
		return nil, nil
	}

	data, err := yaml.Marshal(defaults)
	if err != nil {
		return nil, fmt.Errorf("marshaling flux defaults: %w", err)
	}
	return data, nil
}

// writeTarGz creates a .tar.gz archive at outputPath with all files under the
// given prefix directory. If fluxData is non-nil, it's included as flux.yaml.
func writeTarGz(outputPath, prefix string, files []archiveFile, fluxData []byte) (int64, error) {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0750); err != nil { // #nosec G301
		return 0, fmt.Errorf("creating output directory: %w", err)
	}

	f, err := os.Create(outputPath) // #nosec G304 -- output path controlled by caller
	if err != nil {
		return 0, fmt.Errorf("creating archive file: %w", err)
	}

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	// Write each file
	for _, af := range files {
		header := &tar.Header{
			Name: filepath.Join(prefix, af.path),
			Mode: 0644,
			Size: int64(len(af.data)),
		}
		if err := tw.WriteHeader(header); err != nil {
			_ = f.Close()
			return 0, fmt.Errorf("writing tar header for %s: %w", af.path, err)
		}
		if _, err := tw.Write(af.data); err != nil {
			_ = f.Close()
			return 0, fmt.Errorf("writing tar data for %s: %w", af.path, err)
		}
	}

	// Write flux.yaml defaults if present
	if fluxData != nil {
		header := &tar.Header{
			Name: filepath.Join(prefix, "flux.yaml"),
			Mode: 0644,
			Size: int64(len(fluxData)),
		}
		if err := tw.WriteHeader(header); err != nil {
			_ = f.Close()
			return 0, fmt.Errorf("writing tar header for flux.yaml: %w", err)
		}
		if _, err := tw.Write(fluxData); err != nil {
			_ = f.Close()
			return 0, fmt.Errorf("writing tar data for flux.yaml: %w", err)
		}
	}

	// Flush writers to get accurate size
	if err := tw.Close(); err != nil {
		_ = f.Close()
		return 0, fmt.Errorf("closing tar writer: %w", err)
	}
	if err := gw.Close(); err != nil {
		_ = f.Close()
		return 0, fmt.Errorf("closing gzip writer: %w", err)
	}

	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return 0, fmt.Errorf("stating output file: %w", err)
	}

	size := info.Size()
	if err := f.Close(); err != nil {
		return 0, fmt.Errorf("closing archive file: %w", err)
	}
	return size, nil
}
