package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nimble-giant/ailloy/pkg/blanks"
	"github.com/nimble-giant/ailloy/pkg/plugin"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

var (
	pluginOutputDir string
	pluginMoldDir   string
	pluginWatch     bool
	pluginForce     bool
)

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Generate and manage Claude Code plugins",
	Long: `Generate Claude Code plugins from Ailloy blanks.

This command dogfoods the Ailloy CLI by automatically generating
Claude Code plugins from the embedded blanks, ensuring consistency
between the CLI and plugin experiences.`,
}

var generatePluginCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate Claude Code plugin from blanks",
	Long: `Generate a complete Claude Code plugin from a mold directory.

The generated plugin will include:
- All commands from the mold
- Plugin manifest (plugin.json)
- README documentation
- Installation scripts
- Hooks and agents configurations`,
	RunE: runGeneratePlugin,
}

var updatePluginCmd = &cobra.Command{
	Use:   "update [path]",
	Short: "Update existing Claude Code plugin",
	Long: `Update an existing Claude Code plugin with the latest blanks.

This preserves any custom additions while updating the core commands
from the latest Ailloy blanks.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runUpdatePlugin,
}

var validatePluginCmd = &cobra.Command{
	Use:   "validate [path]",
	Short: "Validate Claude Code plugin structure",
	Long:  `Validate that a Claude Code plugin has the correct structure and all required files.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runValidatePlugin,
}

func init() {
	rootCmd.AddCommand(pluginCmd)
	pluginCmd.AddCommand(generatePluginCmd)
	pluginCmd.AddCommand(updatePluginCmd)
	pluginCmd.AddCommand(validatePluginCmd)

	// Generate command flags
	generatePluginCmd.Flags().StringVarP(&pluginOutputDir, "output", "o", "ailloy", "Output directory for generated plugin")
	generatePluginCmd.Flags().BoolVarP(&pluginWatch, "watch", "w", false, "Watch blanks and regenerate on changes")
	generatePluginCmd.Flags().BoolVarP(&pluginForce, "force", "f", false, "Overwrite existing plugin without prompting")
	generatePluginCmd.Flags().StringVar(&pluginMoldDir, "mold", "", "mold directory to generate plugin from (required)")

	// Update command flags
	updatePluginCmd.Flags().BoolVarP(&pluginForce, "force", "f", false, "Force update without backup")
	updatePluginCmd.Flags().StringVar(&pluginMoldDir, "mold", "", "mold directory to update plugin from (required)")
}

func runGeneratePlugin(cmd *cobra.Command, args []string) error {
	// Display generation header
	fmt.Println(styles.WorkingBanner("Generating Claude Code Plugin from Ailloy Blanks..."))
	fmt.Println()

	// Check if output directory exists
	if _, err := os.Stat(pluginOutputDir); err == nil && !pluginForce {
		// Directory exists, ask for confirmation
		warning := styles.WarningStyle.Render("⚠️  Warning: ") +
			fmt.Sprintf("Plugin directory '%s' already exists.", pluginOutputDir)
		fmt.Println(warning)
		fmt.Print("Overwrite? (y/N): ")

		var response string
		_, _ = fmt.Scanln(&response) // #nosec G104 -- User input prompt, error is acceptable
		if response != "y" && response != "Y" {
			fmt.Println(styles.InfoStyle.Render("Generation cancelled."))
			return nil
		}
	}

	if pluginMoldDir == "" {
		return fmt.Errorf("--mold flag is required: ailloy plugin generate --mold <mold-dir>")
	}

	reader, err := blanks.NewMoldReaderFromPath(pluginMoldDir)
	if err != nil {
		return err
	}

	// Create generator
	generator := plugin.NewGenerator(pluginOutputDir, reader)

	// Configure generator
	generator.Config = &plugin.Config{
		Name:        "ailloy",
		Version:     "1.0.0",
		Description: "AI-assisted development workflows and structured blanks for Claude Code",
		Author: plugin.Author{
			Name:  "Ailloy Team",
			Email: "support@ailloy.dev",
			URL:   "https://github.com/nimble-giant/ailloy",
		},
	}

	// Progress display
	fmt.Println(styles.InfoStyle.Render("📦 Generating plugin structure..."))

	// Generate plugin
	if err := generator.Generate(); err != nil {
		return fmt.Errorf("failed to generate plugin: %w", err)
	}

	// Success message
	fmt.Println()
	fmt.Println(styles.SuccessBanner("Plugin generated successfully!"))
	fmt.Println()

	// Display next steps - use relative path for plugin directory
	relPluginDir := "./" + pluginOutputDir
	nextSteps := styles.InfoBoxStyle.Render(
		styles.AccentStyle.Render("Next Steps:\n\n") +
			"1. Test the plugin locally:\n" +
			styles.CodeStyle.Render(fmt.Sprintf("   claude --plugin-dir %s", relPluginDir)) + "\n\n" +
			"2. Or run the install script:\n" +
			styles.CodeStyle.Render(fmt.Sprintf("   cd %s && ./scripts/install.sh", pluginOutputDir)),
	)
	fmt.Println(nextSteps)

	// Watch mode
	if pluginWatch {
		fmt.Println()
		fmt.Println(styles.InfoStyle.Render("👁  Watch mode enabled. Monitoring blanks for changes..."))
		// TODO: Implement file watching
	}

	return nil
}

func runUpdatePlugin(cmd *cobra.Command, args []string) error {
	pluginPath := "ailloy"
	if len(args) > 0 {
		pluginPath = args[0]
	}

	// Check if plugin exists
	if _, err := os.Stat(filepath.Join(pluginPath, ".ailloy", "plugin.json")); err != nil {
		return fmt.Errorf("no valid plugin found at %s", pluginPath)
	}

	if pluginMoldDir == "" {
		return fmt.Errorf("--mold flag is required: ailloy plugin update --mold <mold-dir>")
	}

	reader, err := blanks.NewMoldReaderFromPath(pluginMoldDir)
	if err != nil {
		return err
	}

	fmt.Println(styles.WorkingBanner("Updating Claude Code Plugin..."))
	fmt.Println()

	// Create updater
	updater := plugin.NewUpdater(pluginPath, reader)

	// Backup existing plugin
	if !pluginForce {
		fmt.Println(styles.InfoStyle.Render("📦 Creating backup..."))
		if err := updater.Backup(); err != nil {
			return fmt.Errorf("failed to backup plugin: %w", err)
		}
	}

	// Update plugin
	fmt.Println(styles.InfoStyle.Render("🔄 Updating commands from blanks..."))
	if err := updater.Update(); err != nil {
		return fmt.Errorf("failed to update plugin: %w", err)
	}

	// Success
	fmt.Println()
	fmt.Println(styles.SuccessBanner("Plugin updated successfully!"))

	// Show what was updated
	if updater.UpdatedFiles > 0 {
		summary := styles.InfoBoxStyle.Render(
			fmt.Sprintf("Updated %d files\n", updater.UpdatedFiles) +
				fmt.Sprintf("Added %d new commands\n", updater.NewCommands) +
				fmt.Sprintf("Preserved %d custom files", updater.PreservedFiles),
		)
		fmt.Println(summary)
	}

	return nil
}

func runValidatePlugin(cmd *cobra.Command, args []string) error {
	pluginPath := "ailloy"
	if len(args) > 0 {
		pluginPath = args[0]
	}

	fmt.Println(styles.InfoStyle.Render("🔍 Validating Claude Code Plugin..."))
	fmt.Println()

	// Create validator
	validator := plugin.NewValidator(pluginPath)

	// Run validation
	results, err := validator.Validate()
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Display results
	if results.IsValid {
		fmt.Println(styles.SuccessStyle.Render("✅ Plugin structure is valid!"))
	} else {
		fmt.Println(styles.ErrorStyle.Render("❌ Plugin validation failed"))
	}

	fmt.Println()

	// Show details
	var details []string

	if results.HasManifest {
		details = append(details, styles.SuccessStyle.Render("✓")+" Plugin manifest found")
	} else {
		details = append(details, styles.ErrorStyle.Render("✗")+" Missing plugin manifest")
	}

	if results.HasCommands {
		details = append(details, styles.SuccessStyle.Render("✓")+
			fmt.Sprintf(" %d commands found", results.CommandCount))
	} else {
		details = append(details, styles.ErrorStyle.Render("✗")+" No commands found")
	}

	if results.HasREADME {
		details = append(details, styles.SuccessStyle.Render("✓")+" README documentation present")
	} else {
		details = append(details, styles.WarningStyle.Render("⚠")+" Missing README")
	}

	detailsBox := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(styles.Primary1).
		Padding(1, 2).
		Render(strings.Join(details, "\n"))

	fmt.Println(detailsBox)

	// Show warnings
	if len(results.Warnings) > 0 {
		fmt.Println()
		fmt.Println(styles.WarningStyle.Render("Warnings:"))
		for _, warning := range results.Warnings {
			fmt.Println("  • " + warning)
		}
	}

	// Show errors
	if len(results.Errors) > 0 {
		fmt.Println()
		fmt.Println(styles.ErrorStyle.Render("Errors:"))
		for _, err := range results.Errors {
			fmt.Println("  • " + err)
		}
	}

	if !results.IsValid {
		return fmt.Errorf("plugin validation failed with %d errors", len(results.Errors))
	}

	return nil
}
