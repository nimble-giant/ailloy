package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kriscoleman/ailloy/pkg/templates"
)

// Generator handles the generation of Claude Code plugins from Ailloy templates
type Generator struct {
	OutputDir string
	Config    *Config
	templates []TemplateInfo
}

// Config represents the plugin configuration
type Config struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Author      Author `json:"author"`
}

// Author represents plugin author information
type Author struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	URL   string `json:"url"`
}

// TemplateInfo holds information about a template
type TemplateInfo struct {
	Name        string
	Description string
	Content     []byte
}

// NewGenerator creates a new plugin generator
func NewGenerator(outputDir string) *Generator {
	return &Generator{
		OutputDir: outputDir,
	}
}

// Generate creates the complete plugin structure
func (g *Generator) Generate() error {
	// Load all templates
	if err := g.loadTemplates(); err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}

	// Create directory structure
	if err := g.createStructure(); err != nil {
		return fmt.Errorf("failed to create structure: %w", err)
	}

	// Generate manifest
	if err := g.generateManifest(); err != nil {
		return fmt.Errorf("failed to generate manifest: %w", err)
	}

	// Transform and write commands
	if err := g.generateCommands(); err != nil {
		return fmt.Errorf("failed to generate commands: %w", err)
	}

	// Generate README
	if err := g.generateREADME(); err != nil {
		return fmt.Errorf("failed to generate README: %w", err)
	}

	// Generate hooks configuration
	if err := g.generateHooks(); err != nil {
		return fmt.Errorf("failed to generate hooks: %w", err)
	}

	// Generate installation script
	if err := g.generateInstallScript(); err != nil {
		return fmt.Errorf("failed to generate install script: %w", err)
	}

	return nil
}

// loadTemplates loads all templates from the embedded filesystem
func (g *Generator) loadTemplates() error {
	templateList, err := templates.ListTemplates()
	if err != nil {
		return err
	}

	for _, tmplName := range templateList {
		content, err := templates.GetTemplate(tmplName)
		if err != nil {
			return fmt.Errorf("failed to load template %s: %w", tmplName, err)
		}

		// Extract description from content
		desc := extractDescription(content)

		g.templates = append(g.templates, TemplateInfo{
			Name:        strings.TrimSuffix(tmplName, ".md"),
			Description: desc,
			Content:     content,
		})
	}

	return nil
}

// createStructure creates the plugin directory structure
func (g *Generator) createStructure() error {
	dirs := []string{
		filepath.Join(g.OutputDir, ".claude-plugin"),
		filepath.Join(g.OutputDir, "commands"),
		filepath.Join(g.OutputDir, "agents"),
		filepath.Join(g.OutputDir, "hooks"),
		filepath.Join(g.OutputDir, "scripts"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0750); err != nil { // #nosec G301 -- Plugin directories need group read access
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// generateManifest creates the plugin.json file
func (g *Generator) generateManifest() error {
	// Use only the fields documented in the official Claude Code plugin schema
	// See: https://code.claude.com/docs/en/plugins
	manifest := map[string]interface{}{
		"name":        g.Config.Name,
		"version":     g.Config.Version,
		"description": g.Config.Description,
		"author": map[string]string{
			"name": g.Config.Author.Name,
		},
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}

	manifestPath := filepath.Join(g.OutputDir, ".claude-plugin", "plugin.json")
	return os.WriteFile(manifestPath, data, 0644) // #nosec G306 -- Plugin manifest needs to be readable
}

// generateCommands transforms templates into Claude Code commands
func (g *Generator) generateCommands() error {
	transformer := NewTransformer()

	for _, tmpl := range g.templates {
		// Transform template to command format
		command, err := transformer.Transform(tmpl)
		if err != nil {
			return fmt.Errorf("failed to transform template %s: %w", tmpl.Name, err)
		}

		// Write command file
		cmdPath := filepath.Join(g.OutputDir, "commands", tmpl.Name+".md")
		//#nosec G306 -- Command files need to be readable
		if err := os.WriteFile(cmdPath, command, 0644); err != nil {
			return fmt.Errorf("failed to write command %s: %w", tmpl.Name, err)
		}
	}

	return nil
}

// generateREADME creates the plugin README
func (g *Generator) generateREADME() error {
	readme := g.buildREADME()
	readmePath := filepath.Join(g.OutputDir, "README.md")
	return os.WriteFile(readmePath, []byte(readme), 0644) // #nosec G306 -- README needs to be readable
}

// generateHooks creates the hooks configuration
func (g *Generator) generateHooks() error {
	hooks := map[string]interface{}{
		"hooks": []map[string]string{
			{
				"name":        "pre-issue-create",
				"event":       "command:before",
				"pattern":     "/create-issue",
				"action":      "validate-github-auth",
				"description": "Ensure GitHub CLI is authenticated",
			},
			{
				"name":        "variable-substitution",
				"event":       "template:load",
				"pattern":     "*",
				"action":      "substitute-variables",
				"description": "Replace template variables",
			},
		},
	}

	data, err := json.MarshalIndent(hooks, "", "  ")
	if err != nil {
		return err
	}

	hooksPath := filepath.Join(g.OutputDir, "hooks", "hooks.json")
	return os.WriteFile(hooksPath, data, 0644) // #nosec G306 -- Hooks config needs to be readable
}

// generateInstallScript creates the installation script
func (g *Generator) generateInstallScript() error {
	script := `#!/bin/bash
# Auto-generated Ailloy Claude Code Plugin Installation Script
set -e

PLUGIN_NAME="` + g.Config.Name + `"
PLUGIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "🦊 Installing Ailloy Plugin for Claude Code..."
echo "==========================================="

# Check requirements
if ! command -v gh &> /dev/null; then
    echo "⚠️  GitHub CLI (gh) not found. Please install it."
fi

if ! command -v git &> /dev/null; then
    echo "❌ Git is required but not installed."
    exit 1
fi

echo "✅ Requirements checked"
echo ""
echo "📦 Plugin Structure:"
echo "  Plugin Name: $PLUGIN_NAME"
echo "  Plugin Path: $PLUGIN_DIR"
echo "  Commands:    $(ls -1 "$PLUGIN_DIR/commands" | wc -l) available"
echo ""
echo "Available Commands:"
for cmd in "$PLUGIN_DIR/commands"/*.md; do
    basename "$cmd" .md | sed 's/^/  🔹 \/'$PLUGIN_NAME':/'
done
echo ""
echo "🎉 Plugin Ready!"
echo ""
echo "To use the plugin, run Claude Code with the --plugin-dir flag:"
echo "  claude --plugin-dir $PLUGIN_DIR"
echo ""
echo "Commands are namespaced as /$PLUGIN_NAME:<command-name>"
echo "Example: /$PLUGIN_NAME:create-issue"
`

	scriptPath := filepath.Join(g.OutputDir, "scripts", "install.sh")
	//#nosec G306 -- Install script needs execute permission
	if err := os.WriteFile(scriptPath, []byte(script), 0750); err != nil {
		return err
	}

	return nil
}

// Helper functions

func extractDescription(content []byte) string {
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "## Purpose") {
			// Look for the next non-empty line
			for i := 1; i < len(lines); i++ {
				if nextLine := strings.TrimSpace(lines[i]); nextLine != "" && !strings.HasPrefix(nextLine, "#") {
					// Truncate if too long
					if len(nextLine) > 100 {
						return nextLine[:97] + "..."
					}
					return nextLine
				}
			}
		}
	}
	return "AI-assisted workflow command"
}

func (g *Generator) buildREADME() string {
	var cmdList strings.Builder
	for _, tmpl := range g.templates {
		cmdList.WriteString(fmt.Sprintf("| `/%s:%s` | %s |\n", g.Config.Name, tmpl.Name, tmpl.Description))
	}

	readme := `# 🧠 Ailloy Plugin for Claude Code

*Auto-generated from Ailloy CLI templates*

## 🚀 Quick Start

### Usage

Run Claude Code with the plugin loaded:

` + "```bash\n" + fmt.Sprintf("claude --plugin-dir /path/to/%s", g.OutputDir) + "\n```" + `

Or run the install script to see available commands:

` + "```bash\n" + fmt.Sprintf("cd %s\n./scripts/install.sh", g.OutputDir) + "\n```" + `

## 📋 Available Commands

Commands are namespaced under ` + "`/" + g.Config.Name + ":`" + `. Use them in Claude Code like:

` + "```\n" + fmt.Sprintf("/%s:create-issue", g.Config.Name) + "\n```" + `

| Command | Description |
|---------|-------------|
` + cmdList.String() + `
## 📚 Learn More

- [Ailloy Documentation](https://github.com/kriscoleman/ailloy)
- [Claude Code Plugins Guide](https://docs.anthropic.com/en/docs/claude-code/plugins)

---

*Generated by Ailloy CLI v` + g.Config.Version + `*
`
	return readme
}