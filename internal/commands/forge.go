package commands

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/nimble-giant/ailloy/pkg/config"
	"github.com/nimble-giant/ailloy/pkg/mold"
	"github.com/nimble-giant/ailloy/pkg/styles"
	embeddedtemplates "github.com/nimble-giant/ailloy/pkg/templates"
	"github.com/spf13/cobra"
)

var forgeCmd = &cobra.Command{
	Use:     "forge",
	Aliases: []string{"template"},
	Short:   "Dry-run render of mold templates",
	Long: `Render all templates in the current mold with flux values and print the result (alias: template).

This is the "what would cast produce?" preview, analogous to helm template.
By default, rendered output is printed to stdout. Use --output to write files to a directory.`,
	RunE: runForge,
}

var (
	forgeOutputDir string
	forgeSetValues []string
)

func init() {
	rootCmd.AddCommand(forgeCmd)

	forgeCmd.Flags().StringVarP(&forgeOutputDir, "output", "o", "", "write rendered files to this directory instead of stdout")
	forgeCmd.Flags().StringArrayVar(&forgeSetValues, "set", nil, "set flux values (key=value)")
}

// loadForgeConfig loads the layered config identical to cast, then applies --set overrides.
func loadForgeConfig(setValues []string) (*config.Config, error) {
	cfg, err := config.LoadConfig(false)
	if err != nil {
		cfg = &config.Config{
			Templates: config.TemplateConfig{
				Flux: make(map[string]string),
			},
		}
	}

	globalCfg, err := config.LoadConfig(true)
	if err == nil && globalCfg.Templates.Flux != nil {
		for key, value := range globalCfg.Templates.Flux {
			if _, exists := cfg.Templates.Flux[key]; !exists {
				cfg.Templates.Flux[key] = value
			}
		}
	}

	config.MergeOreFlux(cfg)

	// Apply --set overrides (highest precedence)
	for _, kv := range setValues {
		key, value, ok := strings.Cut(kv, "=")
		if !ok {
			return nil, fmt.Errorf("invalid --set value %q: expected key=value", kv)
		}
		cfg.Templates.Flux[key] = value
	}

	return cfg, nil
}

// renderFile processes a single template and returns the rendered content.
func renderFile(name string, content []byte, cfg *config.Config, opts ...config.TemplateOption) (string, error) {
	rendered, err := config.ProcessTemplate(string(content), cfg.Templates.Flux, &cfg.Ore, opts...)
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
	cfg, err := loadForgeConfig(forgeSetValues)
	if err != nil {
		return err
	}

	manifest, err := embeddedtemplates.LoadManifest()
	if err != nil {
		return fmt.Errorf("failed to load mold manifest: %w", err)
	}

	// Apply flux defaults from manifest schema
	cfg.Templates.Flux = mold.ApplyFluxDefaults(manifest.Flux, cfg.Templates.Flux)

	// Validate flux against schema (log warnings, don't fail)
	if err := mold.ValidateFlux(manifest.Flux, cfg.Templates.Flux); err != nil {
		log.Printf("warning: %v", err)
	}

	// Build ingot resolver with search paths:
	// 1. Current directory (mold-local)
	// 2. Project .ailloy/
	// 3. Global ~/.ailloy/
	resolver := buildIngotResolver(cfg)
	opts := []config.TemplateOption{config.WithIngotResolver(resolver)}

	var files []renderedFile

	// Render command templates
	for _, name := range manifest.Commands {
		content, err := embeddedtemplates.GetTemplate(name)
		if err != nil {
			return fmt.Errorf("reading command template %s: %w", name, err)
		}
		rendered, err := renderFile(name, content, cfg, opts...)
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
		content, err := embeddedtemplates.GetSkill(name)
		if err != nil {
			return fmt.Errorf("reading skill template %s: %w", name, err)
		}
		rendered, err := renderFile(name, content, cfg, opts...)
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
		content, err := embeddedtemplates.GetWorkflowTemplate(name)
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
func buildIngotResolver(cfg *config.Config) *config.IngotResolver {
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

	return config.NewIngotResolver(searchPaths, cfg.Templates.Flux, &cfg.Ore)
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
