package commands

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nimble-giant/ailloy/pkg/mold"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/nimble-giant/ailloy/pkg/templates"
	"github.com/spf13/cobra"
)

var castCmd = &cobra.Command{
	Use:     "cast [mold-dir]",
	Aliases: []string{"install"},
	Short:   "Cast Ailloy configuration into a project",
	Long: `Cast Ailloy configuration into a project (alias: install).

Installs rendered templates from the given mold into the current repository.
Use -f to layer additional flux value files (Helm-style).`,
	RunE: runCast,
}

var (
	withWorkflows bool
	castSetFlags  []string
	castValFiles  []string
)

func init() {
	rootCmd.AddCommand(castCmd)

	castCmd.Flags().BoolVar(&withWorkflows, "with-workflows", false, "include GitHub Actions workflow templates (e.g. Claude Code agent)")
	castCmd.Flags().StringArrayVar(&castSetFlags, "set", nil, "override flux variable (format: key=value, can be repeated)")
	castCmd.Flags().StringArrayVarP(&castValFiles, "values", "f", nil, "flux value files (can be repeated, later files override earlier)")
}

func runCast(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("mold directory is required: ailloy cast <mold-dir>")
	}

	moldDir := args[0]
	reader, err := templates.NewMoldReaderFromPath(moldDir)
	if err != nil {
		return err
	}

	return castProject(reader)
}

// loadCastFlux loads layered flux values using Helm-style precedence:
// mold flux.yaml < mold.yaml schema defaults < -f files (left to right) < --set flags
func loadCastFlux(reader *templates.MoldReader) (map[string]any, error) {
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
	if len(castValFiles) > 0 {
		overlay, err := mold.LayerFluxFiles(castValFiles)
		if err != nil {
			return nil, err
		}
		for k, v := range overlay {
			flux[k] = v
		}
	}

	// Layer 4: Apply --set overrides (highest precedence)
	if err := mold.ApplySetOverrides(flux, castSetFlags); err != nil {
		return nil, err
	}

	return flux, nil
}

func castProject(reader *templates.MoldReader) error {
	// Welcome message
	fmt.Println(styles.WorkingBanner("Casting Ailloy project structure..."))
	fmt.Println()

	// Check runtime dependencies
	checkDependencies()

	// Check if we're in a git repository
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		warning := styles.WarningStyle.Render("⚠️  Warning: ") +
			"Not in a Git repository. Consider running " +
			styles.CodeStyle.Render("git init") + " first."
		fmt.Println(warning)
		fmt.Println()
	}

	// Create Claude Code directory structure
	dirs := []string{
		".claude",
		".claude/commands",
		".claude/skills",
	}
	if withWorkflows {
		dirs = append(dirs, ".github/workflows")
	}

	fmt.Println(styles.InfoStyle.Render("📁 Creating directory structure..."))
	for i, dir := range dirs {
		fmt.Print(styles.ProgressStep(i+1, len(dirs), "Creating "+dir))
		time.Sleep(100 * time.Millisecond) // Small delay for visual effect

		if err := os.MkdirAll(dir, 0750); err != nil { // #nosec G301 -- Project directories need group read access
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		fmt.Print("\r" + styles.SuccessStyle.Render("✅ Created directory: ") + styles.CodeStyle.Render(dir) + "\n")
	}
	fmt.Println()

	// Copy template files from mold
	if err := copyTemplateFiles(reader); err != nil {
		return fmt.Errorf("failed to copy template files: %w", err)
	}

	// Copy workflow templates (opt-in)
	if withWorkflows {
		if err := copyWorkflowTemplates(reader); err != nil {
			return fmt.Errorf("failed to copy workflow templates: %w", err)
		}
	}
	// Copy skill files from mold
	if err := copySkillFiles(reader); err != nil {
		return fmt.Errorf("failed to copy skill files: %w", err)
	}
	// Success celebration
	fmt.Println()
	successMessage := "Project casting complete!"
	fmt.Println(styles.SuccessBanner(successMessage))
	fmt.Println()

	// Summary box
	summaryContent := styles.SuccessStyle.Render("🎉 Setup Complete!\n\n") +
		styles.FoxBullet("Command templates: ") + styles.CodeStyle.Render(".claude/commands/") + "\n"
	summaryContent += styles.FoxBullet("Skill templates:   ") + styles.CodeStyle.Render(".claude/skills/") + "\n"
	if withWorkflows {
		summaryContent += styles.FoxBullet("Workflow templates: ") + styles.CodeStyle.Render(".github/workflows/") + "\n"
	}
	summaryContent += styles.FoxBullet("Ready for AI-powered development! 🚀")

	// Check if CLAUDE.md exists and suggest creating one if not
	if _, err := os.Stat("CLAUDE.md"); os.IsNotExist(err) {
		summaryContent += "\n\n" +
			styles.InfoStyle.Render("💡 Tip: ") +
			"No " + styles.CodeStyle.Render("CLAUDE.md") + " detected. " +
			"Run " + styles.CodeStyle.Render("/init") + " in Claude Code to create one."
	}

	summary := styles.SuccessBoxStyle.Render(summaryContent)

	fmt.Println(summary)

	return nil
}

// copyTemplateFiles copies markdown template files from the mold to the project directory
func copyTemplateFiles(reader *templates.MoldReader) error {
	templateDir := ".claude/commands"

	flux, err := loadCastFlux(reader)
	if err != nil {
		flux = make(map[string]any)
	}

	// Validate: prefer flux.schema.yaml, fall back to mold.yaml flux: section
	manifest, _ := reader.LoadManifest()
	var schema []mold.FluxVar
	if s, err := reader.LoadFluxSchema(); err == nil && s != nil {
		schema = s
	} else if manifest != nil && len(manifest.Flux) > 0 {
		schema = manifest.Flux
	}
	if err := mold.ValidateFlux(schema, flux); err != nil {
		log.Printf("warning: %v", err)
	}

	// Build ingot resolver
	resolver := buildIngotResolver(flux)
	opts := []mold.TemplateOption{mold.WithIngotResolver(resolver)}

	// Use manifest-driven command list
	var cmdList []string
	if manifest != nil {
		cmdList = manifest.Commands
	}

	for _, templateName := range cmdList {
		// Read from mold filesystem
		content, err := reader.GetTemplate(templateName)
		if err != nil {
			// Create a placeholder if template doesn't exist
			content = []byte(fmt.Sprintf(`# %s

This is a placeholder for the %s Claude Code command template.

## Usage

Add your Claude Code command documentation here.

## Notes

- This template was auto-generated during ailloy cast
- Replace this content with your actual Claude Code command
`, strings.TrimSuffix(templateName, ".md"), strings.TrimSuffix(templateName, ".md")))
		}

		// Process template variables
		processedContent, err := mold.ProcessTemplate(string(content), flux, opts...)
		if err != nil {
			return fmt.Errorf("failed to process template %s: %w", templateName, err)
		}

		// Write to project directory
		destPath := filepath.Join(templateDir, templateName)
		//#nosec G306 -- Templates need to be readable
		if err := os.WriteFile(destPath, []byte(processedContent), 0644); err != nil {
			return fmt.Errorf("failed to write template %s: %w", templateName, err)
		}

		fmt.Println(styles.SuccessStyle.Render("✅ Created template: ") + styles.CodeStyle.Render(destPath))
	}

	return nil
}

// copyWorkflowTemplates copies GitHub Actions workflow templates from the mold to the project
func copyWorkflowTemplates(reader *templates.MoldReader) error {
	workflowDir := ".github/workflows"

	manifest, _ := reader.LoadManifest()
	var wfList []string
	if manifest != nil {
		wfList = manifest.Workflows
	}

	for _, workflowName := range wfList {
		content, err := reader.GetWorkflowTemplate(workflowName)
		if err != nil {
			return fmt.Errorf("failed to read workflow template %s: %w", workflowName, err)
		}

		destPath := filepath.Join(workflowDir, workflowName)
		//#nosec G306 -- Workflow files need to be readable
		if err := os.WriteFile(destPath, content, 0644); err != nil {
			return fmt.Errorf("failed to write workflow %s: %w", workflowName, err)
		}

		fmt.Println(styles.SuccessStyle.Render("✅ Created workflow: ") + styles.CodeStyle.Render(destPath))
	}

	return nil
}

// copySkillFiles copies skill files from the mold to the project directory
func copySkillFiles(reader *templates.MoldReader) error {
	skillDir := ".claude/skills"

	flux, err := loadCastFlux(reader)
	if err != nil {
		flux = make(map[string]any)
	}

	// Build ingot resolver
	resolver := buildIngotResolver(flux)
	opts := []mold.TemplateOption{mold.WithIngotResolver(resolver)}

	// Use manifest-driven skill list
	manifest, _ := reader.LoadManifest()
	var skillList []string
	if manifest != nil {
		skillList = manifest.Skills
	}

	for _, skillName := range skillList {
		// Read from mold filesystem
		content, err := reader.GetSkill(skillName)
		if err != nil {
			// Create a placeholder if skill doesn't exist
			content = []byte(fmt.Sprintf(`# Skill: %s

This is a placeholder for the %s Claude Code skill.

## Usage

Add your Claude Code skill documentation here.

## Notes

- This skill was auto-generated during ailloy cast
- Replace this content with your actual Claude Code skill
`, strings.TrimSuffix(skillName, ".md"), strings.TrimSuffix(skillName, ".md")))
		}

		// Process template variables
		processedContent, err := mold.ProcessTemplate(string(content), flux, opts...)
		if err != nil {
			return fmt.Errorf("failed to process skill %s: %w", skillName, err)
		}

		// Write to project directory
		destPath := filepath.Join(skillDir, skillName)
		//#nosec G306 -- Skills need to be readable
		if err := os.WriteFile(destPath, []byte(processedContent), 0644); err != nil {
			return fmt.Errorf("failed to write skill %s: %w", skillName, err)
		}

		fmt.Println(styles.SuccessStyle.Render("✅ Created skill: ") + styles.CodeStyle.Render(destPath))
	}

	return nil
}
