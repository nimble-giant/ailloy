package commands

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/nimble-giant/ailloy/pkg/blanks"
	"github.com/nimble-giant/ailloy/pkg/mold"
	"github.com/nimble-giant/ailloy/pkg/plugin"
	"github.com/nimble-giant/ailloy/pkg/styles"
)

// castAsPlugin renders the mold via the existing flux pipeline and packages the
// result as a Claude Code plugin under .claude/plugins/<slug>/ (or
// ~/.claude/plugins/<slug>/ when --global is set).
func castAsPlugin(reader *blanks.MoldReader) error {
	fmt.Println(styles.WorkingBanner("Casting Ailloy mold as Claude Code plugin..."))
	fmt.Println()

	manifest, err := reader.LoadManifest()
	if err != nil {
		return fmt.Errorf("failed to load mold manifest: %w", err)
	}

	flux, err := loadCastFlux(reader)
	if err != nil {
		flux = make(map[string]any)
	}

	rendered, err := renderMoldFiles(reader, manifest, flux)
	if err != nil {
		return err
	}

	pluginFiles, hadWorkflows := filterForPlugin(rendered)
	if withWorkflows && hadWorkflows {
		fmt.Println(styles.WarningStyle.Render("⚠️  --with-workflows has no effect with --as-plugin: workflow blanks are not bundled into Claude Code plugins."))
	}

	readme, err := readMoldReadme(reader, flux)
	if err != nil {
		return err
	}

	manifestInput, err := buildManifestInput(manifest)
	if err != nil {
		return err
	}

	slug, err := slugifyPluginName(manifestInput.Name)
	if err != nil {
		return err
	}

	targetDir, err := resolvePluginTargetDir(slug)
	if err != nil {
		return err
	}

	p := &plugin.Packager{OutputDir: targetDir}
	if err := p.Package(pluginFiles, manifestInput, readme); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println(styles.SuccessStyle.Render("✅ Plugin written to ") + styles.CodeStyle.Render(targetDir))
	fmt.Println(styles.InfoStyle.Render("💡 Claude Code will discover the plugin at this path on its next start."))

	return nil
}

// renderMoldFiles runs the cast rendering pipeline (flux validation, ingot
// resolver, template processing) and returns the resulting files in memory.
// Files that render to empty content are skipped, matching cast's on-disk
// behavior (#130).
func renderMoldFiles(reader *blanks.MoldReader, manifest *mold.Mold, flux map[string]any) ([]plugin.RenderedFile, error) {
	ignorePatterns := mold.LoadIgnorePatterns(reader.FS(), manifest)
	var resolveOpts []mold.ResolveOption
	if len(ignorePatterns) > 0 {
		resolveOpts = append(resolveOpts, mold.WithIgnorePatterns(ignorePatterns))
	}

	resolved, err := mold.ResolveFiles(flux["output"], reader.FS(), resolveOpts...)
	if err != nil {
		return nil, fmt.Errorf("resolving output files: %w", err)
	}

	var schema []mold.FluxVar
	if s, lerr := reader.LoadFluxSchema(); lerr == nil && s != nil {
		schema = s
	} else if manifest != nil && len(manifest.Flux) > 0 {
		schema = manifest.Flux
	}
	if verr := mold.ValidateFlux(schema, flux); verr != nil {
		log.Printf("warning: %v", verr)
	}

	resolver := buildIngotResolver(flux, reader.Root())
	tplOpts := []mold.TemplateOption{mold.WithIngotResolver(resolver)}

	out := make([]plugin.RenderedFile, 0, len(resolved))
	for _, rf := range resolved {
		content, rerr := fs.ReadFile(reader.FS(), rf.SrcPath)
		if rerr != nil {
			return nil, fmt.Errorf("reading %s: %w", rf.SrcPath, rerr)
		}
		var output []byte
		if rf.Process {
			processed, perr := mold.ProcessTemplate(string(content), flux, tplOpts...)
			if perr != nil {
				return nil, fmt.Errorf("processing %s: %w", rf.SrcPath, perr)
			}
			if strings.TrimSpace(processed) == "" {
				continue
			}
			output = []byte(processed)
		} else {
			output = content
		}
		out = append(out, plugin.RenderedFile{CastDest: rf.DestPath, Content: output})
	}
	return out, nil
}

// filterForPlugin drops .github/workflows/ entries from the rendered set and
// reports whether any were present (so the caller can warn).
func filterForPlugin(files []plugin.RenderedFile) ([]plugin.RenderedFile, bool) {
	var hadWorkflows bool
	out := make([]plugin.RenderedFile, 0, len(files))
	for _, f := range files {
		if strings.HasPrefix(filepath.ToSlash(f.CastDest), ".github/workflows/") {
			hadWorkflows = true
			continue
		}
		out = append(out, f)
	}
	return out, hadWorkflows
}

// readMoldReadme reads README.md from the mold root and processes it through
// flux. Returns nil if the mold has no README.
func readMoldReadme(reader *blanks.MoldReader, flux map[string]any) ([]byte, error) {
	raw, err := fs.ReadFile(reader.FS(), "README.md")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading mold README.md: %w", err)
	}
	resolver := buildIngotResolver(flux, reader.Root())
	processed, perr := mold.ProcessTemplate(string(raw), flux, mold.WithIngotResolver(resolver))
	if perr != nil {
		return nil, fmt.Errorf("processing mold README.md: %w", perr)
	}
	return []byte(processed), nil
}

// buildManifestInput synthesizes the plugin manifest from the mold manifest
// with optional flag overrides.
func buildManifestInput(m *mold.Mold) (plugin.ManifestInput, error) {
	out := plugin.ManifestInput{}
	if m != nil {
		out.Name = m.Name
		out.Version = m.Version
		out.Description = m.Description
		out.Author = m.Author
	}
	if castPluginName != "" {
		out.Name = castPluginName
	}
	if castPluginVer != "" {
		out.Version = castPluginVer
	}
	if strings.TrimSpace(out.Name) == "" {
		return out, fmt.Errorf("plugin requires a name; set 'name' in mold.yaml or pass --plugin-name")
	}
	return out, nil
}

// slugifyPluginName lowercases the name, replaces runs of non-alphanumeric
// characters with a single dash, and trims leading/trailing dashes. Returns an
// error when the result is empty.
func slugifyPluginName(name string) (string, error) {
	var b strings.Builder
	prevDash := true
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "", fmt.Errorf("plugin name %q produces an empty slug; pass --plugin-name <slug>", name)
	}
	return slug, nil
}

// resolvePluginTargetDir returns the absolute output directory for the plugin,
// honoring --global.
func resolvePluginTargetDir(slug string) (string, error) {
	if castGlobal {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		return filepath.Join(homeDir, ".claude", "plugins", slug), nil
	}
	return filepath.Join(".claude", "plugins", slug), nil
}
