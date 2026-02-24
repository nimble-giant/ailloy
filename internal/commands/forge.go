package commands

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/nimble-giant/ailloy/pkg/blanks"
	"github.com/nimble-giant/ailloy/pkg/foundry"
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
	forgeOutputDir string
	forgeSetValues []string
	forgeValFiles  []string
)

func init() {
	rootCmd.AddCommand(forgeCmd)

	forgeCmd.Flags().StringVarP(&forgeOutputDir, "output", "o", "", "write rendered files to this directory instead of stdout")
	forgeCmd.Flags().StringArrayVar(&forgeSetValues, "set", nil, "set flux values (key=value)")
	forgeCmd.Flags().StringArrayVarP(&forgeValFiles, "values", "f", nil, "flux value files (can be repeated, later files override earlier)")
}

// loadForgeFlux loads layered flux values using Helm-style precedence:
// mold flux.yaml < mold.yaml schema defaults < -f files (left to right) < --set flags
func loadForgeFlux(reader *blanks.MoldReader) (map[string]any, error) {
	// Layer 1: Load mold flux.yaml as base
	fluxDefaults, err := reader.LoadFluxDefaults()
	if err != nil {
		fluxDefaults = make(map[string]any)
	}

	// Layer 2: Apply mold.yaml schema defaults
	manifest, _ := reader.LoadManifest()
	if manifest != nil && len(manifest.Flux) > 0 {
		fluxDefaults = mold.ApplyFluxDefaults(manifest.Flux, fluxDefaults)
	}

	flux := make(map[string]any)
	for k, v := range fluxDefaults {
		flux[k] = v
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
}

func runForge(_ *cobra.Command, args []string) error {
	reader, err := resolveForgeReader(args)
	if err != nil {
		return err
	}

	flux, err := loadForgeFlux(reader)
	if err != nil {
		return err
	}

	manifest, err := reader.LoadManifest()
	if err != nil {
		return fmt.Errorf("failed to load mold manifest: %w", err)
	}

	// Validate: prefer flux.schema.yaml, fall back to mold.yaml flux: section
	schema, _ := reader.LoadFluxSchema()
	if schema == nil && len(manifest.Flux) > 0 {
		schema = manifest.Flux
	}
	if err := mold.ValidateFlux(schema, flux); err != nil {
		log.Printf("warning: %v", err)
	}

	// Build ingot resolver
	resolver := buildIngotResolver(flux)
	opts := []mold.TemplateOption{mold.WithIngotResolver(resolver)}

	// Resolve all output files from the flux.
	resolved, err := mold.ResolveFiles(flux["output"], reader.FS())
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
			rendered, err = renderFile(rf.SrcPath, content, flux, opts...)
			if err != nil {
				return err
			}
		} else {
			rendered = string(content)
		}

		files = append(files, renderedFile{
			destPath: rf.DestPath,
			content:  rendered,
		})
	}

	fmt.Println(styles.WorkingBanner("Forging blanks..."))
	fmt.Println()

	if forgeOutputDir != "" {
		return writeForgeFiles(files, forgeOutputDir)
	}
	return printForgeFiles(files)
}

// buildIngotResolver creates an IngotResolver with the standard search path order:
// current directory (mold-local), project .ailloy/, global ~/.ailloy/.
func buildIngotResolver(flux map[string]any) *mold.IngotResolver {
	searchPaths := []string{"."}

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

func writeForgeFiles(files []renderedFile, outputDir string) error {
	for _, f := range files {
		dest := filepath.Join(outputDir, f.destPath)
		if err := os.MkdirAll(filepath.Dir(dest), 0750); err != nil { // #nosec G301 -- Output directories need group read access
			return fmt.Errorf("creating directory for %s: %w", f.destPath, err)
		}
		//#nosec G306 -- Rendered blanks need to be readable
		if err := os.WriteFile(dest, []byte(f.content), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", f.destPath, err)
		}
		fmt.Println(styles.SuccessStyle.Render("wrote ") + styles.CodeStyle.Render(dest))
	}
	return nil
}

// resolveForgeReader creates a MoldReader from args or the embedded mold.
func resolveForgeReader(args []string) (*blanks.MoldReader, error) {
	if len(args) >= 1 {
		if foundry.IsRemoteReference(args[0]) {
			fsys, err := foundry.Resolve(args[0])
			if err != nil {
				return nil, fmt.Errorf("resolving remote mold: %w", err)
			}
			return blanks.NewMoldReader(fsys), nil
		}
		return blanks.NewMoldReaderFromPath(args[0])
	}
	if smelt.HasEmbeddedMold() {
		fsys, err := smelt.OpenEmbeddedMold()
		if err != nil {
			return nil, fmt.Errorf("opening embedded mold: %w", err)
		}
		return blanks.NewMoldReader(fsys), nil
	}
	return nil, fmt.Errorf("mold directory is required: ailloy forge <mold-dir>")
}
