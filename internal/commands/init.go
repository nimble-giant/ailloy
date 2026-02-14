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

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Ailloy configuration",
	Long: `Initialize Ailloy configuration for a project or globally.

By default, initializes Ailloy structure in the current repository.
Use -g or --global to install user-level configuration instead.`,
	RunE: runInit,
}

var globalInit bool

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().BoolVarP(&globalInit, "global", "g", false, "install user-level configuration instead of project-level")
}

func runInit(cmd *cobra.Command, args []string) error {
	if globalInit {
		return initGlobal()
	}

	// Default to project initialization
	return initProject()
}

func initProject() error {
	// Welcome message
	fmt.Println(styles.WorkingBanner("Initializing Ailloy project structure..."))
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

	// Success celebration
	fmt.Println()
	successMessage := "Project initialization complete!"
	fmt.Println(styles.SuccessBanner(successMessage))
	fmt.Println()

	// Summary box
	summaryContent := styles.SuccessStyle.Render("🎉 Setup Complete!\n\n") +
		styles.FoxBullet("Command templates: ") + styles.CodeStyle.Render(".claude/commands/") + "\n" +
		styles.FoxBullet("Ready for AI-powered development! 🚀")

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

// copyTemplateFiles copies markdown template files from embedded sources to the project directory
func copyTemplateFiles() error {
	templateDir := ".claude/commands"

	// Load configuration to get template variables
	cfg, err := config.LoadConfig(false) // Load project config
	if err != nil {
		// If config loading fails, continue with empty variables
		cfg = &config.Config{
			Templates: config.TemplateConfig{
				Variables: make(map[string]string),
			},
		}
	}

	// Try to load global config and merge variables
	globalCfg, err := config.LoadConfig(true)
	if err == nil && globalCfg.Templates.Variables != nil {
		// Merge global variables (project variables take precedence)
		for key, value := range globalCfg.Templates.Variables {
			if _, exists := cfg.Templates.Variables[key]; !exists {
				cfg.Templates.Variables[key] = value
			}
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

- This template was auto-generated during ailloy init
- Replace this content with your actual Claude Code command
`, strings.TrimSuffix(templateName, ".md"), strings.TrimSuffix(templateName, ".md")))
		}

		// Process template variables
		processedContent := config.ProcessTemplate(string(content), cfg.Templates.Variables)

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

func initGlobal() error {
	fmt.Println("Initializing global Ailloy configuration...")

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

	fmt.Println("\n✅ Global initialization complete!")
	fmt.Printf("Configuration stored in: %s\n", globalDir)
	fmt.Println("Edit the configuration files to set up your AI providers and preferences.")

	return nil
}
