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

// pluginPackageOpts configures a plugin packaging run independent of cast's
// package-level flag state, so both the CLI and CastMold can invoke the same
// pipeline with their respective options.
type pluginPackageOpts struct {
	Global          bool
	WithWorkflows   bool
	NameOverride    string
	VersionOverride string
}

// pluginPackageResult summarizes a packaging run for callers that want to
// surface the output location.
type pluginPackageResult struct {
	TargetDir    string
	HadWorkflows bool
	FileCount    int
}

// castClaudePlugin is the CLI entrypoint for `ailloy cast --claude-plugin`.
// It prints the banner and success message; the underlying pipeline lives in
// packageMoldAsClaudePlugin so foundry install / CastMold can share it.
func castClaudePlugin(reader *blanks.MoldReader) error {
	if !castSilent.Load() {
		fmt.Println(styles.WorkingBanner("Casting Ailloy mold as Claude Code plugin..."))
		fmt.Println()
	}

	flux, err := loadCastFlux(reader)
	if err != nil {
		flux = make(map[string]any)
	}

	res, err := packageMoldAsClaudePlugin(reader, flux, pluginPackageOpts{
		Global:          castGlobal,
		WithWorkflows:   withWorkflows,
		NameOverride:    castPluginName,
		VersionOverride: castPluginVer,
	})
	if err != nil {
		return err
	}

	if !castSilent.Load() {
		if withWorkflows && res.HadWorkflows {
			fmt.Println(styles.WarningStyle.Render("⚠️  --with-workflows has no effect with --claude-plugin: workflow blanks are not bundled into Claude Code plugins."))
		}
		fmt.Println()
		fmt.Println(styles.SuccessStyle.Render("✅ Plugin written to ") + styles.CodeStyle.Render(res.TargetDir))
		fmt.Println(styles.InfoStyle.Render("💡 Claude Code will discover the plugin at this path on its next start."))
	}

	return nil
}

// packageMoldAsClaudePlugin is the silent pipeline shared by the CLI and
// CastMold. It loads the manifest, renders all blanks against the supplied
// flux, and hands the result to plugin.Packager.
func packageMoldAsClaudePlugin(reader *blanks.MoldReader, flux map[string]any, opts pluginPackageOpts) (pluginPackageResult, error) {
	var res pluginPackageResult

	manifest, err := reader.LoadManifest()
	if err != nil {
		return res, fmt.Errorf("loading mold manifest: %w", err)
	}

	rendered, err := renderMoldFiles(reader, manifest, flux)
	if err != nil {
		return res, err
	}

	pluginFiles, hadWorkflows := filterForPlugin(rendered)
	res.HadWorkflows = hadWorkflows

	readme, err := readMoldReadme(reader, flux)
	if err != nil {
		return res, err
	}

	manifestInput, err := buildManifestInput(manifest, opts.NameOverride, opts.VersionOverride)
	if err != nil {
		return res, err
	}

	slug, err := slugifyPluginName(manifestInput.Name)
	if err != nil {
		return res, err
	}

	targetDir, err := resolvePluginTargetDir(slug, opts.Global)
	if err != nil {
		return res, err
	}
	res.TargetDir = targetDir

	p := &plugin.Packager{OutputDir: targetDir}
	if err := p.Package(pluginFiles, manifestInput, readme); err != nil {
		return res, err
	}
	res.FileCount = len(pluginFiles)
	return res, nil
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

// buildManifestInput synthesizes the plugin manifest from the mold manifest,
// applying optional name/version overrides.
func buildManifestInput(m *mold.Mold, nameOverride, versionOverride string) (plugin.ManifestInput, error) {
	out := plugin.ManifestInput{}
	if m != nil {
		out.Name = m.Name
		out.Version = m.Version
		out.Description = m.Description
		out.Author = m.Author
	}
	if nameOverride != "" {
		out.Name = nameOverride
	}
	if versionOverride != "" {
		out.Version = versionOverride
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

// resolvePluginTargetDir returns the absolute output directory for the plugin.
// When global is true, plugins go under ~/.claude/plugins/<slug>/; otherwise
// .claude/plugins/<slug>/ in the working directory.
func resolvePluginTargetDir(slug string, global bool) (string, error) {
	if global {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		return filepath.Join(homeDir, ".claude", "plugins", slug), nil
	}
	return filepath.Join(".claude", "plugins", slug), nil
}
