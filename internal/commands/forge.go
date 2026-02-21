package commands

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/nimble-giant/ailloy/pkg/mold"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/nimble-giant/ailloy/pkg/templates"
	"github.com/spf13/cobra"
)

var forgeCmd = &cobra.Command{
	Use:     "forge [mold-dir]",
	Aliases: []string{"template"},
	Short:   "Dry-run render of mold templates",
	Long: `Render all templates in the given mold with flux values and print the result (alias: template).

This is the "what would cast produce?" preview, analogous to helm template.
By default, rendered output is printed to stdout. Use --output to write files to a directory.`,
	Args: cobra.ExactArgs(1),
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
func loadForgeFlux(reader *templates.MoldReader) (map[string]any, error) {
	// Layer 1: Load mold flux.yaml as base
	fluxDefaults, err := reader.LoadFluxDefaults()
	if err != nil {
		fluxDefaults = make(map[string]any)
	}

	// Layer 2: Apply mold.yaml schema defaults (backwards compat)
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

// renderFile processes a single template and returns the rendered content.
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

func runForge(cmd *cobra.Command, args []string) error {
	moldDir := args[0]

	reader, err := templates.NewMoldReaderFromPath(moldDir)
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

	var files []renderedFile

	// Render command templates
	for _, name := range manifest.Commands {
		content, err := reader.GetTemplate(name)
		if err != nil {
			return fmt.Errorf("reading command template %s: %w", name, err)
		}
		rendered, err := renderFile(name, content, flux, opts...)
		if err != nil {
			return err
		}
		files = append(files, renderedFile{
			destPath: filepath.Join(".claude", "commands", name),
			content:  rendered,
		})
	}

	// Render skill templates
	for _, name := range manifest.Skills {
		content, err := reader.GetSkill(name)
		if err != nil {
			return fmt.Errorf("reading skill template %s: %w", name, err)
		}
		rendered, err := renderFile(name, content, flux, opts...)
		if err != nil {
			return err
		}
		files = append(files, renderedFile{
			destPath: filepath.Join(".claude", "skills", name),
			content:  rendered,
		})
	}

	// Workflow templates are not Go-templated, include as-is
	for _, name := range manifest.Workflows {
		content, err := reader.GetWorkflowTemplate(name)
		if err != nil {
			return fmt.Errorf("reading workflow template %s: %w", name, err)
		}
		files = append(files, renderedFile{
			destPath: filepath.Join(".github", "workflows", name),
			content:  string(content),
		})
	}

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
		//#nosec G306 -- Rendered templates need to be readable
		if err := os.WriteFile(dest, []byte(f.content), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", f.destPath, err)
		}
		fmt.Println(styles.SuccessStyle.Render("wrote ") + styles.CodeStyle.Render(dest))
	}
	return nil
}
