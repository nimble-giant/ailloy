package commands

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"dario.cat/mergo"
	"github.com/nimble-giant/ailloy/internal/tui/ceremony"
	"github.com/nimble-giant/ailloy/pkg/blanks"
	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/merge"
	"github.com/nimble-giant/ailloy/pkg/mold"
	"github.com/nimble-giant/ailloy/pkg/smelt"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

var forgeCmd = &cobra.Command{
	Use:     "forge [mold-dir]",
	Aliases: []string{"blank", "template"},
	Short:   "Dry-run render of mold blanks",
	Long: `Render all blanks in the given mold with flux values and print the result (alias: blank, template).

This is the "what would cast produce?" preview, analogous to helm template.
If run from a stuffed binary (created by smelt -o binary), the embedded mold
is used automatically when no mold-dir is provided.
By default, rendered output is printed to stdout. Use --output to write files to a directory.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runForge,
}

var (
	forgeOutputDir                string
	forgeSetValues                []string
	forgeValFiles                 []string
	forgeForceReplaceOnParseError bool
)

func init() {
	rootCmd.AddCommand(forgeCmd)

	forgeCmd.Flags().StringVarP(&forgeOutputDir, "output", "o", "", "write rendered files to this directory instead of stdout")
	forgeCmd.Flags().StringArrayVar(&forgeSetValues, "set", nil, "set flux values (key=value)")
	forgeCmd.Flags().StringArrayVarP(&forgeValFiles, "values", "f", nil, "flux value files (can be repeated, later files override earlier)")
	forgeCmd.Flags().BoolVar(&forgeForceReplaceOnParseError,
		"force-replace-on-parse-error",
		false,
		"if a destination uses strategy: merge but is unparseable, replace it instead of erroring (only used with --output)")
}

// loadForgeFlux loads layered flux values using Helm-style precedence:
// ore defaults < mold flux.yaml < mold.yaml schema defaults < -f files
// (left to right) < --set flags. The resolver may be nil — callers that
// don't resolve ore deps will get pre-Phase-9 behavior.
func loadForgeFlux(reader *blanks.MoldReader, resolver *EphemeralOreResolver) (map[string]any, error) {
	// Layer 0: Ore-namespace defaults (resolved ephemerally). Lowest priority;
	// the mold's own flux.yaml deep-merges on top via mergo.WithOverride.
	flux := make(map[string]any)
	if resolver != nil {
		if err := mergo.Merge(&flux, resolver.Defaults()); err != nil {
			return nil, fmt.Errorf("layering ore defaults: %w", err)
		}
	}

	// Layer 1: Load mold flux.yaml on top of ore defaults
	fluxDefaults, err := reader.LoadFluxDefaults()
	if err != nil {
		fluxDefaults = make(map[string]any)
	}

	// Layer 2: Apply mold.yaml schema defaults
	manifest, _ := reader.LoadManifest()
	if manifest != nil && len(manifest.Flux) > 0 {
		fluxDefaults = mold.ApplyFluxDefaults(manifest.Flux, fluxDefaults)
	}

	if err := mergo.Merge(&flux, fluxDefaults, mergo.WithOverride); err != nil {
		return nil, fmt.Errorf("merging mold defaults over ore defaults: %w", err)
	}

	// Layer 3: Layer -f files left-to-right (each overrides previous)
	if len(forgeValFiles) > 0 {
		overlay, err := mold.LayerFluxFiles(forgeValFiles)
		if err != nil {
			return nil, err
		}
		for k, v := range overlay {
			flux[k] = v
		}
	}

	// Layer 4: Apply --set overrides (highest precedence)
	if err := mold.ApplySetOverrides(flux, forgeSetValues); err != nil {
		return nil, err
	}

	return flux, nil
}

// renderFile processes a single blank and returns the rendered content.
func renderFile(name string, content []byte, flux map[string]any, opts ...mold.TemplateOption) (string, error) {
	rendered, err := mold.ProcessTemplate(string(content), flux, opts...)
	if err != nil {
		return "", fmt.Errorf("template %s: %w", name, err)
	}
	return rendered, nil
}

type renderedFile struct {
	destPath string // relative output path (e.g. ".claude/commands/brainstorm.md")
	content  string
	strategy string
}

func runForge(_ *cobra.Command, args []string) error {
	reader, remote, err := resolveForgeReader(args)
	if err != nil {
		return err
	}

	manifest, err := reader.LoadManifest()
	if err != nil {
		return fmt.Errorf("failed to load mold manifest: %w", err)
	}

	// Resolve ore deps ephemerally for the preview render — never touches
	// .ailloy/ores/. Local-path deps are refused when the parent mold itself
	// was loaded from a remote source (mirrors installDeclaredDeps' rule).
	oreResolver, err := ResolveDepsEphemeral(manifest, !remote)
	if err != nil {
		return fmt.Errorf("resolving ore deps for forge: %w", err)
	}

	flux, err := loadForgeFlux(reader, oreResolver)
	if err != nil {
		return err
	}

	// Validate: prefer flux.schema.yaml, fall back to mold.yaml flux: section.
	schema, _ := reader.LoadFluxSchema()
	if schema == nil && len(manifest.Flux) > 0 {
		schema = manifest.Flux
	}
	mergedSchema, _, _, mergeErr := oreResolver.MergeInto(schema, nil)
	if mergeErr != nil {
		return fmt.Errorf("merging ore schema overlays: %w", mergeErr)
	}
	if err := mold.ValidateFlux(mergedSchema, flux); err != nil {
		log.Printf("warning: %v", err)
	}

	// Build ingot resolver
	ingotResolver := buildIngotResolver(flux, reader.Root())
	opts := []mold.TemplateOption{mold.WithIngotResolver(ingotResolver)}

	// Load ignore patterns from .ailloyignore and mold.yaml.
	ignorePatterns := mold.LoadIgnorePatterns(reader.FS(), manifest)

	var resolveOpts []mold.ResolveOption
	if len(ignorePatterns) > 0 {
		resolveOpts = append(resolveOpts, mold.WithIgnorePatterns(ignorePatterns))
	}

	// Resolve all output files from the flux.
	resolved, err := mold.ResolveFiles(flux["output"], reader.FS(), resolveOpts...)
	if err != nil {
		return fmt.Errorf("resolving output files: %w", err)
	}

	var files []renderedFile
	for _, rf := range resolved {
		content, err := fs.ReadFile(reader.FS(), rf.SrcPath)
		if err != nil {
			return fmt.Errorf("reading %s: %w", rf.SrcPath, err)
		}

		var rendered string
		if rf.Process {
			fluxForFile := flux
			if len(rf.Set) > 0 {
				fluxForFile = mold.MergeSet(flux, rf.Set)
			}
			rendered, err = renderFile(rf.SrcPath, content, fluxForFile, opts...)
			if err != nil {
				return err
			}
		} else {
			rendered = string(content)
		}

		// Skip files that render to empty or whitespace-only content (#130)
		if rf.Process && strings.TrimSpace(rendered) == "" {
			log.Printf("skipping %s: rendered to empty content", rf.SrcPath)
			continue
		}

		files = append(files, renderedFile{
			destPath: rf.DestPath,
			content:  rendered,
			strategy: rf.Strategy,
		})
	}

	ceremony.Open(ceremony.Forge)

	if forgeOutputDir != "" {
		if err := writeForgeFiles(files, forgeOutputDir, forgeForceReplaceOnParseError, manifest.Name); err != nil {
			return err
		}
		ceremony.Stamp(ceremony.Forge, fmt.Sprintf("%d file(s) → %s", len(files), forgeOutputDir))
		return nil
	}
	// Stdout-rendered preview: skip the trailing stamp so pipe consumers
	// (e.g. `ailloy forge | code -`) don't get an extra trailing line.
	return printForgeFiles(files)
}

// buildIngotResolver creates an IngotResolver with the standard search path order:
// mold source root (so bundled ingots are found when casting from a remote or
// path mold), current directory (mold-local), project .ailloy/, global ~/.ailloy/.
// moldRoot may be empty when the mold has no on-disk root (e.g., embedded molds).
func buildIngotResolver(flux map[string]any, moldRoot string) *mold.IngotResolver {
	var searchPaths []string

	if moldRoot != "" {
		searchPaths = append(searchPaths, moldRoot)
	}
	searchPaths = append(searchPaths, ".")

	if _, err := os.Stat(".ailloy"); err == nil {
		searchPaths = append(searchPaths, ".ailloy")
	}

	if homeDir, err := os.UserHomeDir(); err == nil {
		globalDir := filepath.Join(homeDir, ".ailloy")
		if _, err := os.Stat(globalDir); err == nil {
			searchPaths = append(searchPaths, globalDir)
		}
	}

	return mold.NewIngotResolver(searchPaths, flux)
}

func printForgeFiles(files []renderedFile) error {
	for i, f := range files {
		fmt.Println(styles.AccentStyle.Render("--- " + f.destPath + " ---"))
		fmt.Print(f.content)
		if !strings.HasSuffix(f.content, "\n") {
			fmt.Println()
		}
		if i < len(files)-1 {
			fmt.Println()
		}
	}
	return nil
}

func writeForgeFiles(files []renderedFile, outputDir string, forceReplaceOnParseError bool, moldName string) error {
	for _, f := range files {
		dest := filepath.Join(outputDir, f.destPath)
		switch f.strategy {
		case "merge":
			err := merge.MergeFile(dest, []byte(f.content), merge.Options{
				ForceReplaceOnParseError: forceReplaceOnParseError,
			})
			if err != nil {
				var pe *merge.ParseError
				if errors.As(err, &pe) {
					return fmt.Errorf(
						"failed to merge into %s: existing %s file could not be parsed: %w. "+
							"Re-run with --force-replace-on-parse-error to overwrite",
						pe.Path, pe.Format, pe.Err)
				}
				return fmt.Errorf("failed to merge %s: %w", dest, err)
			}
			fmt.Println(styles.SuccessStyle.Render("Merged ") + styles.CodeStyle.Render(dest))
		case "append":
			if moldName == "" {
				return fmt.Errorf("append strategy requires a mold name (dest %s)", dest)
			}
			err := merge.AppendFile(dest, []byte(f.content), merge.AppendOptions{
				MoldName: moldName,
			})
			if err != nil {
				return fmt.Errorf("failed to append into %s: %w", dest, err)
			}
			fmt.Println(styles.SuccessStyle.Render("Appended ") + styles.CodeStyle.Render(dest))
		case "", "replace":
			if err := os.MkdirAll(filepath.Dir(dest), 0750); err != nil { // #nosec G301 -- Output directories need group read access
				return fmt.Errorf("creating directory for %s: %w", f.destPath, err)
			}
			//#nosec G306 -- Rendered blanks need to be readable
			if err := os.WriteFile(dest, []byte(f.content), 0644); err != nil {
				return fmt.Errorf("writing %s: %w", f.destPath, err)
			}
			fmt.Println(styles.SuccessStyle.Render("Wrote ") + styles.CodeStyle.Render(dest))
		default:
			return fmt.Errorf("unknown strategy %q on output for %s", f.strategy, dest)
		}
	}
	return nil
}

// resolveForgeReader creates a MoldReader from args or the embedded mold.
// The remote return flag indicates whether the mold itself was loaded from a
// remote source — used by the caller to refuse local-path ore deps from a
// remotely-resolved mold.
func resolveForgeReader(args []string) (*blanks.MoldReader, bool, error) {
	if len(args) >= 1 {
		if foundry.IsRemoteReference(args[0]) {
			fsys, root, err := foundry.ResolveWithRoot(args[0])
			if err != nil {
				return nil, true, fmt.Errorf("resolving remote mold: %w", err)
			}
			return blanks.NewMoldReaderFromFS(fsys, root), true, nil
		}
		reader, err := blanks.NewMoldReaderFromPath(args[0])
		return reader, false, err
	}
	if smelt.HasEmbeddedMold() {
		fsys, err := smelt.OpenEmbeddedMold()
		if err != nil {
			return nil, false, fmt.Errorf("opening embedded mold: %w", err)
		}
		return blanks.NewMoldReader(fsys), false, nil
	}
	return nil, false, fmt.Errorf("mold directory is required: ailloy forge <mold-dir>")
}
