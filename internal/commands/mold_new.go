package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

var newMoldCmd = &cobra.Command{
	Use:     "new <name>",
	Aliases: []string{"create"},
	Short:   "Scaffold a new mold directory",
	Long: `Scaffold a new mold directory with boilerplate files (alias: create).

Creates a ready-to-use mold with mold.yaml, flux.yaml, sample blanks,
and an optional AGENTS.md for tool-agnostic agent instructions.

Example:
  ailloy mold new my-mold
  ailloy mold new my-mold -o /tmp
  ailloy mold new my-mold --no-agents`,
	Args: cobra.ExactArgs(1),
	RunE: runNewMold,
}

var (
	newMoldOutput   string
	newMoldNoAgents bool
)

func init() {
	newMoldCmd.Flags().StringVarP(&newMoldOutput, "output", "o", ".", "parent directory to create the mold in")
	newMoldCmd.Flags().BoolVar(&newMoldNoAgents, "no-agents", false, "skip generating AGENTS.md")
}

func runNewMold(_ *cobra.Command, args []string) error {
	name := args[0]

	// Validate name
	if strings.ContainsAny(name, "/\\:*?\"<>|") {
		return fmt.Errorf("invalid mold name %q: contains special characters", name)
	}

	moldDir := filepath.Join(newMoldOutput, name)

	// Check target doesn't already exist
	if _, err := os.Stat(moldDir); err == nil {
		return fmt.Errorf("directory %s already exists", moldDir)
	}

	fmt.Println(styles.WorkingBanner("Scaffolding new mold..."))
	fmt.Println()

	// Create directory structure
	dirs := []string{
		moldDir,
		filepath.Join(moldDir, "commands"),
		filepath.Join(moldDir, "skills"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0750); err != nil { // #nosec G301 -- Mold directories need group read access
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Write scaffold files
	files := map[string]string{
		"mold.yaml":         scaffoldMoldYaml(name),
		"flux.yaml":         scaffoldFluxYaml,
		"commands/hello.md": scaffoldCommandBlank,
		"skills/helper.md":  scaffoldSkillBlank,
	}

	if !newMoldNoAgents {
		files["AGENTS.md"] = scaffoldAgentsMd
	}

	for relPath, content := range files {
		dest := filepath.Join(moldDir, relPath)
		//#nosec G306 -- Mold files need to be readable
		if err := os.WriteFile(dest, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", dest, err)
		}
		fmt.Println(styles.SuccessStyle.Render("  created ") + styles.CodeStyle.Render(relPath))
	}

	fmt.Println()
	fmt.Println(styles.SuccessBanner("Mold scaffolded at " + moldDir))
	fmt.Println()

	nextSteps := styles.InfoStyle.Render("Next steps:\n\n") +
		"  1. Edit " + styles.CodeStyle.Render("mold.yaml") + " to set description and author\n" +
		"  2. Add your blanks to " + styles.CodeStyle.Render("commands/") + " and " + styles.CodeStyle.Render("skills/") + "\n" +
		"  3. Validate with " + styles.CodeStyle.Render("ailloy temper "+moldDir) + "\n" +
		"  4. Preview with " + styles.CodeStyle.Render("ailloy forge "+moldDir) + "\n" +
		"  5. Install with " + styles.CodeStyle.Render("ailloy cast "+moldDir)
	fmt.Println(styles.InfoBoxStyle.Render(nextSteps))

	return nil
}

func scaffoldMoldYaml(name string) string {
	return fmt.Sprintf(`apiVersion: v1
kind: mold
name: %s
version: 0.1.0
description: ""
author:
  name: ""
`, name)
}

const scaffoldFluxYaml = `output:
  commands: .claude/commands
  skills: .claude/skills

project_name: my-project
`

const scaffoldAgentsMd = `# {{project_name}} Agent Instructions

## Build & Test

- Run tests: ` + "`make test`" + `
- Lint: ` + "`make lint`" + `

## Code Style

- Follow project conventions
`

const scaffoldCommandBlank = `# Hello

A sample command blank for {{project_name}}.

## Usage

Use this blank as a starting point for your own commands.
`

const scaffoldSkillBlank = `# Helper

A sample skill blank for {{project_name}}.
`
