package plugin

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

// RenderedFile is a single flux-rendered blank produced by cast, identified by
// its intended cast destination (e.g. ".claude/commands/foo.md").
type RenderedFile struct {
	CastDest string
	Content  []byte
}

// ManifestInput supplies the fields synthesized into .claude-plugin/plugin.json.
type ManifestInput struct {
	Name        string
	Version     string
	Description string
	Author      mold.Author
}

// Packager writes a Claude Code plugin to OutputDir from already-rendered blanks.
type Packager struct {
	OutputDir string
}

// Package wipes OutputDir's contents, routes each RenderedFile into its plugin
// internal path, writes the manifest, and writes README.md if provided. AGENTS.md
// is routed via the normal RenderedFile path (CastDest == "AGENTS.md").
func (p *Packager) Package(files []RenderedFile, manifest ManifestInput, readme []byte) error {
	if p.OutputDir == "" {
		return fmt.Errorf("packager: OutputDir is empty")
	}

	if info, err := os.Stat(p.OutputDir); err == nil && !info.IsDir() {
		return fmt.Errorf("plugin target %s exists and is not a directory", p.OutputDir)
	}

	if err := wipeContents(p.OutputDir); err != nil {
		return fmt.Errorf("wiping plugin output dir: %w", err)
	}
	if err := os.MkdirAll(p.OutputDir, 0o750); err != nil { // #nosec G301 -- plugin dir needs group read access
		return fmt.Errorf("creating plugin output dir: %w", err)
	}

	written := make(map[string]string) // plugin internal path -> source CastDest
	for _, rf := range files {
		internal, ok := translatePath(rf.CastDest)
		if !ok {
			log.Printf("plugin packager: dropping unrecognized path %q", rf.CastDest)
			continue
		}
		if prev, dup := written[internal]; dup {
			return fmt.Errorf("plugin path collision at %s: %q and %q both map there", internal, prev, rf.CastDest)
		}
		written[internal] = rf.CastDest

		dest := filepath.Join(p.OutputDir, internal)
		if err := os.MkdirAll(filepath.Dir(dest), 0o750); err != nil { // #nosec G301
			return fmt.Errorf("creating dir for %s: %w", dest, err)
		}
		if err := os.WriteFile(dest, rf.Content, 0o644); err != nil { // #nosec G306 -- plugin contents need to be readable
			return fmt.Errorf("writing %s: %w", dest, err)
		}
	}

	if err := writeManifest(p.OutputDir, manifest); err != nil {
		return err
	}

	if len(readme) > 0 {
		readmePath := filepath.Join(p.OutputDir, "README.md")
		if err := os.WriteFile(readmePath, readme, 0o644); err != nil { // #nosec G306
			return fmt.Errorf("writing README.md: %w", err)
		}
	}

	return nil
}

// translatePath maps a cast destination to its plugin-internal path. The bool
// reports whether the path is recognized; unrecognized paths are dropped.
func translatePath(castDest string) (string, bool) {
	clean := filepath.ToSlash(castDest)
	switch {
	case strings.HasPrefix(clean, ".claude/commands/"):
		return filepath.FromSlash(strings.TrimPrefix(clean, ".claude/")), true
	case strings.HasPrefix(clean, ".claude/skills/"):
		return filepath.FromSlash(strings.TrimPrefix(clean, ".claude/")), true
	case strings.HasPrefix(clean, ".claude/agents/"):
		return filepath.FromSlash(strings.TrimPrefix(clean, ".claude/")), true
	case strings.HasPrefix(clean, ".claude/hooks/"):
		return filepath.FromSlash(strings.TrimPrefix(clean, ".claude/")), true
	case clean == "AGENTS.md":
		return "AGENTS.md", true
	}
	return "", false
}

// writeManifest synthesizes and writes .claude-plugin/plugin.json. Empty fields
// (description, author) are omitted entirely; missing version defaults to 0.1.0.
func writeManifest(outputDir string, m ManifestInput) error {
	manifest := map[string]any{
		"name": m.Name,
	}
	version := m.Version
	if strings.TrimSpace(version) == "" {
		version = "0.1.0"
	}
	manifest["version"] = version
	if strings.TrimSpace(m.Description) != "" {
		manifest["description"] = m.Description
	}
	if strings.TrimSpace(m.Author.Name) != "" {
		author := map[string]string{"name": m.Author.Name}
		manifest["author"] = author
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling plugin.json: %w", err)
	}

	manifestDir := filepath.Join(outputDir, ".claude-plugin")
	if err := os.MkdirAll(manifestDir, 0o750); err != nil { // #nosec G301
		return fmt.Errorf("creating .claude-plugin dir: %w", err)
	}
	manifestPath := filepath.Join(manifestDir, "plugin.json")
	if err := os.WriteFile(manifestPath, data, 0o644); err != nil { // #nosec G306
		return fmt.Errorf("writing plugin.json: %w", err)
	}
	return nil
}

// wipeContents removes every entry inside dir without touching dir itself or
// its siblings. If dir does not exist, this is a no-op.
func wipeContents(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(dir, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}
