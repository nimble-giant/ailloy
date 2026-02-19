package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nimble-giant/ailloy/pkg/config"
	"github.com/nimble-giant/ailloy/pkg/styles"
	embeddedtemplates "github.com/nimble-giant/ailloy/pkg/templates"
	"github.com/spf13/cobra"
)

var castCmd = &cobra.Command{
	Use:     "cast",
	Aliases: []string{"install"},
	Short:   "Cast Ailloy configuration into a project",
	Long: `Cast Ailloy configuration into a project or globally (alias: install).

By default, casts Ailloy structure into the current repository.
Use -g or --global to install user-level configuration instead.`,
	RunE: runCast,
}

var (
	globalInit    bool
	withWorkflows bool
	setFlags      []string
)

func init() {
	rootCmd.AddCommand(castCmd)

	castCmd.Flags().BoolVarP(&globalInit, "global", "g", false, "install user-level configuration instead of project-level")
	castCmd.Flags().BoolVar(&withWorkflows, "with-workflows", false, "include GitHub Actions workflow templates (e.g. Claude Code agent)")
	castCmd.Flags().StringArrayVar(&setFlags, "set", nil, "override flux variable (format: key=value, can be repeated)")
}

func runCast(cmd *cobra.Command, args []string) error {
	if globalInit {
		return castGlobal()
	}

	// Default to project casting
	return castProject()
}

func castProject() error {
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

	// Copy template files from embedded templates
	if err := copyTemplateFiles(); err != nil {
		return fmt.Errorf("failed to copy template files: %w", err)
	}

	// Copy workflow templates (opt-in)
	if withWorkflows {
		if err := copyWorkflowTemplates(); err != nil {
			return fmt.Errorf("failed to copy workflow templates: %w", err)
		}
	}
	// Copy skill files from embedded templates
	if err := copySkillFiles(); err != nil {
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

// loadLayeredFluxConfig loads flux variables and ore config using full layering,
// including --set flag overrides.
func loadLayeredFluxConfig() (*config.Config, error) {
	if len(setFlags) > 0 {
		return config.LoadLayeredConfig(setFlags)
	}

	// Standard two-layer merge: project + global
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
	return cfg, nil
}

// copyTemplateFiles copies markdown template files from embedded sources to the project directory
func copyTemplateFiles() error {
	templateDir := ".claude/commands"

	cfg, err := loadLayeredFluxConfig()
	if err != nil {
		cfg = &config.Config{
			Templates: config.TemplateConfig{
				Flux: make(map[string]string),
			},
		}
	}

	// Define template files to copy
	templates := []string{
		"brainstorm.md",
		"pr-description.md",
		"create-issue.md",
		"start-issue.md",
		"update-pr.md",
		"open-pr.md",
		"preflight.md",
		"pr-comments.md",
		"pr-review.md",
	}

	for _, templateName := range templates {
		// Read from embedded filesystem
		content, err := embeddedtemplates.GetTemplate(templateName)
		if err != nil {
			// Create a placeholder if embedded file doesn't exist
			content = []byte(fmt.Sprintf(`# %s

This is a placeholder for the %s Claude Code command template.

## Usage

Add your Claude Code command documentation here.

## Notes

- This template was auto-generated during ailloy cast
- Replace this content with your actual Claude Code command
`, strings.TrimSuffix(templateName, ".md"), strings.TrimSuffix(templateName, ".md")))
		}

		// Process template variables (with ore-aware rendering)
		processedContent, err := config.ProcessTemplate(string(content), cfg.Templates.Flux, &cfg.Ore)
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

// copyWorkflowTemplates copies GitHub Actions workflow templates from embedded sources to the project
func copyWorkflowTemplates() error {
	workflowDir := ".github/workflows"

	workflows := []string{
		"claude-code.yml",
		"claude-code-review.yml",
	}

	for _, workflowName := range workflows {
		content, err := embeddedtemplates.GetWorkflowTemplate(workflowName)
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

// copySkillFiles copies skill files from embedded sources to the project directory
func copySkillFiles() error {
	skillDir := ".claude/skills"

	cfg, err := loadLayeredFluxConfig()
	if err != nil {
		cfg = &config.Config{
			Templates: config.TemplateConfig{
				Flux: make(map[string]string),
			},
		}
	}

	// Define skill files to copy
	skills := []string{
		"brainstorm.md",
	}

	for _, skillName := range skills {
		// Read from embedded filesystem
		content, err := embeddedtemplates.GetSkill(skillName)
		if err != nil {
			// Create a placeholder if embedded file doesn't exist
			content = []byte(fmt.Sprintf(`# Skill: %s

This is a placeholder for the %s Claude Code skill.

## Usage

Add your Claude Code skill documentation here.

## Notes

- This skill was auto-generated during ailloy cast
- Replace this content with your actual Claude Code skill
`, strings.TrimSuffix(skillName, ".md"), strings.TrimSuffix(skillName, ".md")))
		}

		// Process template variables (with ore-aware rendering)
		processedContent, err := config.ProcessTemplate(string(content), cfg.Templates.Flux, &cfg.Ore)
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
func castGlobal() error {
	fmt.Println("Casting global Ailloy configuration...")

	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Create global .ailloy directory structure
	globalDir := filepath.Join(homeDir, ".ailloy")
	dirs := []string{
		globalDir,
		filepath.Join(globalDir, "templates"),
		filepath.Join(globalDir, "providers"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0750); err != nil { // #nosec G301 -- User config directories need group read access
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		fmt.Printf("Created directory: %s\n", dir)
	}

	// Create global configuration
	configPath := filepath.Join(globalDir, "ailloy.yaml")
	configContent := `# Ailloy Global Configuration
user:
  name: ""
  email: ""

providers:
  claude:
    enabled: true
    api_key_env: "ANTHROPIC_API_KEY"

  gpt:
    enabled: false
    api_key_env: "OPENAI_API_KEY"

templates:
  auto_update: true
  repositories:
    - "https://github.com/nimble-giant/ailloy-templates"

preferences:
  default_provider: "claude"
  verbose_output: false
`

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil { // User config - restricted permissions
		return fmt.Errorf("failed to create global config file: %w", err)
	}
	fmt.Printf("Created global configuration: %s\n", configPath)

	fmt.Println("\n✅ Global casting complete!")
	fmt.Printf("Configuration stored in: %s\n", globalDir)
	fmt.Println("Edit the configuration files to set up your AI providers and preferences.")

	return nil
}
