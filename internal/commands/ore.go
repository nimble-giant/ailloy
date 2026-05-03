package commands

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/mold"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

var snakeCaseRE = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

func isSnakeCase(s string) bool { return snakeCaseRE.MatchString(s) }

var oreCmd = &cobra.Command{
	Use:   "ore",
	Short: "Work with Ailloy ores (reusable flux schemas)",
	Long: `Commands for managing Ailloy ores.

Ores are reusable flux-schema fragments. An ore packages a named group of
flux variables with their schema (flux.schema.yaml) and defaults (flux.yaml)
under the namespace ore.<name>.*. Consumers install ores from a git repo
with 'ailloy ore add'; molds declare them in mold.yaml dependencies:.`,
}

var oreGetCmd = &cobra.Command{
	Use:   "get <reference>",
	Short: "Download an ore without installing",
	Args:  cobra.ExactArgs(1),
	RunE:  runOreGet,
}

var oreAddCmd = &cobra.Command{
	Use:   "add <reference>",
	Short: "Download and register an ore",
	Args:  cobra.ExactArgs(1),
	RunE:  runOreAdd,
}

var oreNewCmd = &cobra.Command{
	Use:   "new <name>",
	Short: "Scaffold a new ore directory",
	Args:  cobra.ExactArgs(1),
	RunE:  runOreNew,
}

var oreRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove an installed ore",
	Args:  cobra.ExactArgs(1),
	RunE:  runOreRemove,
}

var (
	oreAddAlias    string
	oreAddGlobal   bool
	oreRemoveForce bool
)

func init() {
	rootCmd.AddCommand(oreCmd)
	oreCmd.AddCommand(oreGetCmd, oreAddCmd, oreNewCmd, oreRemoveCmd)

	oreAddCmd.Flags().StringVar(&oreAddAlias, "as", "", "namespace alias (install at ore.<alias>.* instead of ore.<name>.*)")
	oreAddCmd.Flags().BoolVar(&oreAddGlobal, "global", false, "install under ~/.ailloy/ores/ instead of ./.ailloy/ores/")
	oreRemoveCmd.Flags().BoolVar(&oreRemoveForce, "force", false, "remove even if other molds depend on this ore")
}

func runOreGet(_ *cobra.Command, args []string) error {
	ref := args[0]
	if !foundry.IsRemoteReference(ref) {
		return fmt.Errorf("expected a remote reference (e.g. github.com/owner/repo), got %q", ref)
	}
	fmt.Println(styles.WorkingBanner("Downloading ore..."))
	fmt.Println()

	parsed, err := foundry.ParseReference(ref)
	if err != nil {
		return fmt.Errorf("parsing reference: %w", err)
	}
	fsys, err := foundry.Resolve(ref)
	if err != nil {
		return fmt.Errorf("resolving ore: %w", err)
	}
	ore, err := mold.LoadOreFromFS(fsys, "ore.yaml")
	if err != nil {
		return fmt.Errorf("invalid ore manifest: %w", err)
	}
	if ore.Kind != "ore" {
		return fmt.Errorf("manifest kind=%q, expected 'ore'", ore.Kind)
	}

	cacheDir, err := foundry.CacheDir()
	if err != nil {
		return fmt.Errorf("determining cache directory: %w", err)
	}
	lock, _ := foundry.ReadLockFile(foundry.LockFileName)
	version := "latest"
	if entry := lock.FindEntry(parsed.CacheKey(), parsed.Subpath); entry != nil {
		version = entry.Version
	}
	cachePath := foundry.VersionDir(cacheDir, parsed, version)
	if parsed.Subpath != "" {
		cachePath = filepath.Join(cachePath, parsed.Subpath)
	}

	fmt.Println(styles.SuccessStyle.Render("Downloaded: ") + styles.AccentStyle.Render(ore.Name+" "+ore.Version))
	if ore.Description != "" {
		fmt.Println(styles.SubtleStyle.Render("  " + ore.Description))
	}
	fmt.Println(styles.InfoStyle.Render("Cache path: ") + styles.CodeStyle.Render(cachePath))
	return nil
}

func runOreAdd(_ *cobra.Command, args []string) error {
	ref := args[0]

	if !foundry.IsRemoteReference(ref) {
		return fmt.Errorf("expected a remote reference (e.g. github.com/owner/repo), got %q", ref)
	}

	fmt.Println(styles.WorkingBanner("Adding ore..."))
	fmt.Println()

	fsys, result, err := foundry.ResolveWithMetadata(ref)
	if err != nil {
		return fmt.Errorf("resolving ore: %w", err)
	}

	// Validate ore.yaml exists and has kind=ore.
	ore, err := mold.LoadOreFromFS(fsys, "ore.yaml")
	if err != nil {
		return fmt.Errorf("invalid ore manifest: %w", err)
	}
	if ore.Kind != "ore" {
		return fmt.Errorf("manifest kind=%q, expected 'ore'", ore.Kind)
	}

	// Determine install-dir name (alias if provided, else ore.Name).
	installName := ore.Name
	if oreAddAlias != "" {
		installName = oreAddAlias
	}
	if !isSnakeCase(installName) {
		return fmt.Errorf("ore install name must be snake_case (lowercase + underscore), got %q", installName)
	}

	// Determine destination root: project or global.
	var destRoot string
	if oreAddGlobal {
		home, herr := os.UserHomeDir()
		if herr != nil {
			return fmt.Errorf("determining home directory: %w", herr)
		}
		destRoot = filepath.Join(home, ".ailloy", "ores")
	} else {
		destRoot = filepath.Join(".ailloy", "ores")
	}
	destDir := filepath.Join(destRoot, installName)
	if err := os.MkdirAll(destDir, 0750); err != nil {
		return fmt.Errorf("creating ore directory: %w", err)
	}

	err = fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		destPath := filepath.Join(destDir, path)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0750)
		}

		content, err := fs.ReadFile(fsys, path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		if err := os.WriteFile(destPath, content, 0644); err != nil { // #nosec G306 -- ore files need to be readable
			return fmt.Errorf("writing %s: %w", destPath, err)
		}

		fmt.Println(styles.SuccessStyle.Render("  + ") + styles.CodeStyle.Render(destPath))
		return nil
	})
	if err != nil {
		return fmt.Errorf("copying ore files: %w", err)
	}

	fmt.Println()
	fmt.Println(styles.SuccessStyle.Render("Ore added: ") + styles.AccentStyle.Render(ore.Name+" "+ore.Version))
	fmt.Println(styles.InfoStyle.Render("Installed to: ") + styles.CodeStyle.Render(destDir))

	if err := recordInstalledArtifact("ore", result, oreAddAlias, oreAddGlobal); err != nil {
		log.Printf("warning: failed to update installed manifest: %v", err)
	}

	// Update ailloy.lock if present (project scope only).
	if !oreAddGlobal {
		if lock, _ := foundry.ReadLockFile(foundry.LockFileName); lock != nil {
			lock.UpsertArtifactLock("ore", foundry.LockEntry{
				Name:      result.Ref.Repo,
				Source:    result.Ref.CacheKey(),
				Subpath:   result.Ref.Subpath,
				Version:   result.Resolved.Tag,
				Commit:    result.Resolved.Commit,
				Alias:     oreAddAlias,
				Timestamp: time.Now().UTC(),
			})
			if err := foundry.WriteLockFile(foundry.LockFileName, lock); err != nil {
				log.Printf("warning: failed to update lock file: %v", err)
			}
		}
	}

	return nil
}

func runOreNew(_ *cobra.Command, args []string) error {
	name := args[0]
	if !isSnakeCase(name) {
		return fmt.Errorf("ore name must be snake_case (lowercase + underscore), got %q", name)
	}
	if _, err := os.Stat(name); err == nil {
		return fmt.Errorf("directory %q already exists", name)
	}
	if err := os.MkdirAll(name, 0750); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	manifest := fmt.Sprintf(`apiVersion: v1
kind: ore
name: %s
version: 0.1.0
description: ""
author:
  name: ""
  url: ""
requires:
  ailloy: ">=0.7.0"
`, name)
	schema := `# Ore schema entries are unprefixed; the ailloy loader prepends ore.<name>.
# at install time. See docs/ore.md for authoring conventions.
- name: enabled
  type: bool
  description: "Enable this ore"
  default: "false"
`
	defaults := `enabled: false
`
	for path, body := range map[string]string{
		filepath.Join(name, "ore.yaml"):         manifest,
		filepath.Join(name, "flux.schema.yaml"): schema,
		filepath.Join(name, "flux.yaml"):        defaults,
	} {
		if err := os.WriteFile(path, []byte(body), 0644); err != nil { // #nosec G306
			return fmt.Errorf("writing %s: %w", path, err)
		}
		fmt.Println(styles.SuccessStyle.Render("  + ") + styles.CodeStyle.Render(path))
	}
	fmt.Println()
	fmt.Println(styles.SuccessStyle.Render("Ore scaffolded: ") + styles.AccentStyle.Render(name))
	return nil
}

// Stub implemented in Task 6.4.
func runOreRemove(_ *cobra.Command, _ []string) error {
	return fmt.Errorf("ore remove: not yet implemented")
}
