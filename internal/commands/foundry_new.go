package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

var foundryNewCmd = &cobra.Command{
	Use:     "new <name>",
	Aliases: []string{"create"},
	Short:   "Scaffold a new foundry index",
	Long: `Scaffold a new foundry index directory with a foundry.yaml and README (alias: create).

Creates a ready-to-publish foundry index that can be hosted as a git repository
or served as a static YAML file.

Example:
  ailloy foundry new my-foundry
  ailloy foundry new my-foundry -o /tmp`,
	Args: cobra.ExactArgs(1),
	RunE: runNewFoundry,
}

var newFoundryOutput string

func init() {
	foundryNewCmd.Flags().StringVarP(&newFoundryOutput, "output", "o", ".", "parent directory to create the foundry in")
}

func runNewFoundry(_ *cobra.Command, args []string) error {
	name := args[0]

	// Validate name.
	if strings.ContainsAny(name, "/\\:*?\"<>|") {
		return fmt.Errorf("invalid foundry name %q: contains special characters", name)
	}

	foundryDir := filepath.Join(newFoundryOutput, name)

	// Check target doesn't already exist.
	if _, err := os.Stat(foundryDir); err == nil {
		return fmt.Errorf("directory %s already exists", foundryDir)
	}

	fmt.Println(styles.WorkingBanner("Scaffolding new foundry index..."))
	fmt.Println()

	if err := os.MkdirAll(foundryDir, 0750); err != nil { // #nosec G301 -- Foundry directories need group read access
		return fmt.Errorf("failed to create directory %s: %w", foundryDir, err)
	}

	// Write scaffold files.
	files := map[string]string{
		"foundry.yaml": scaffoldFoundryYaml(name),
		"README.md":    scaffoldFoundryReadme(name),
	}

	for relPath, content := range files {
		dest := filepath.Join(foundryDir, relPath)
		//#nosec G306 -- Foundry files need to be readable
		if err := os.WriteFile(dest, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", dest, err)
		}
		fmt.Println(styles.SuccessStyle.Render("  created ") + styles.CodeStyle.Render(relPath))
	}

	fmt.Println()
	fmt.Println(styles.SuccessBanner("Foundry scaffolded at " + foundryDir))
	fmt.Println()

	nextSteps := styles.InfoStyle.Render("Next steps:\n\n") +
		"  1. Edit " + styles.CodeStyle.Render("foundry.yaml") + " to set description, author, and molds\n" +
		"  2. Push to a git repository\n" +
		"  3. Register with " + styles.CodeStyle.Render("ailloy foundry add <url>")
	fmt.Println(nextSteps)

	return nil
}

func scaffoldFoundryYaml(name string) string {
	return fmt.Sprintf(`apiVersion: v1
kind: foundry-index
name: %s
description: ""
author:
  name: ""
  url: ""
molds: []
`, name)
}

func scaffoldFoundryReadme(name string) string {
	return fmt.Sprintf(`# %s

An [Ailloy](https://github.com/nimble-giant/ailloy) foundry index.

## What is a Foundry Index?

A foundry index is a YAML catalog of Ailloy molds that enables SCM-agnostic mold discovery.
Users can register this foundry with:

`+"```"+`bash
ailloy foundry add <url>
`+"```"+`

Then search for molds across all registered foundries:

`+"```"+`bash
ailloy foundry search <query>
`+"```"+`

## Adding Molds

Edit `+"`foundry.yaml`"+` to add molds to this index:

`+"```"+`yaml
molds:
  - name: my-mold
    source: github.com/owner/my-mold
    description: "What this mold does"
    tags: ["tag1", "tag2"]
`+"```"+`

See the [foundry documentation](https://github.com/nimble-giant/ailloy/blob/main/docs/foundry.md) for the full schema reference.
`, name)
}
