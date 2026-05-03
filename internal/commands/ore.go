package commands

import (
	"fmt"
	"path/filepath"

	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/mold"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

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

// Stubs implemented in later tasks of Phase 6.
func runOreAdd(_ *cobra.Command, _ []string) error { return fmt.Errorf("ore add: not yet implemented") }
func runOreNew(_ *cobra.Command, _ []string) error { return fmt.Errorf("ore new: not yet implemented") }
func runOreRemove(_ *cobra.Command, _ []string) error {
	return fmt.Errorf("ore remove: not yet implemented")
}
