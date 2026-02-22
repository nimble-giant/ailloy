package commands

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/mold"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

var ingotCmd = &cobra.Command{
	Use:   "ingot",
	Short: "Work with Ailloy ingots (reusable components)",
	Long: `Commands for managing Ailloy ingots.

Ingots are reusable template components that can be included in molds
via the {{ingot "name"}} template function.`,
}

var ingotGetCmd = &cobra.Command{
	Use:   "get <reference>",
	Short: "Download an ingot without installing",
	Long: `Download an ingot to the local cache without installing it.

The reference follows the standard format: <host>/<owner>/<repo>[@<version>][//<subpath>]
After download, the cache path is printed.`,
	Args: cobra.ExactArgs(1),
	RunE: runIngotGet,
}

var ingotAddCmd = &cobra.Command{
	Use:   "add <reference>",
	Short: "Download and register an ingot",
	Long: `Download an ingot and register it in the project's .ailloy/ingots/ directory.

The ingot files are copied into .ailloy/ingots/<name>/ for use by the template engine.`,
	Args: cobra.ExactArgs(1),
	RunE: runIngotAdd,
}

func init() {
	rootCmd.AddCommand(ingotCmd)
	ingotCmd.AddCommand(ingotGetCmd)
	ingotCmd.AddCommand(ingotAddCmd)
}

func runIngotGet(_ *cobra.Command, args []string) error {
	ref := args[0]

	if !foundry.IsRemoteReference(ref) {
		return fmt.Errorf("expected a remote reference (e.g. github.com/owner/repo), got %q", ref)
	}

	fmt.Println(styles.WorkingBanner("Downloading ingot..."))
	fmt.Println()

	parsed, err := foundry.ParseReference(ref)
	if err != nil {
		return fmt.Errorf("parsing reference: %w", err)
	}

	fsys, err := foundry.Resolve(ref)
	if err != nil {
		return fmt.Errorf("resolving ingot: %w", err)
	}

	// Validate ingot.yaml exists.
	ingot, err := mold.LoadIngotFromFS(fsys, "ingot.yaml")
	if err != nil {
		return fmt.Errorf("invalid ingot manifest: %w", err)
	}

	// Print cache path.
	cacheDir, err := foundry.CacheDir()
	if err != nil {
		return fmt.Errorf("determining cache directory: %w", err)
	}

	lock, _ := foundry.ReadLockFile(foundry.LockFileName)
	version := "latest"
	if entry := lock.FindEntry(parsed.CacheKey()); entry != nil {
		version = entry.Version
	}

	cachePath := foundry.VersionDir(cacheDir, parsed, version)
	if parsed.Subpath != "" {
		cachePath = filepath.Join(cachePath, parsed.Subpath)
	}

	fmt.Println(styles.SuccessStyle.Render("Downloaded: ") + styles.AccentStyle.Render(ingot.Name+" "+ingot.Version))
	if ingot.Description != "" {
		fmt.Println(styles.SubtleStyle.Render("  " + ingot.Description))
	}
	fmt.Println(styles.InfoStyle.Render("Cache path: ") + styles.CodeStyle.Render(cachePath))

	return nil
}

func runIngotAdd(_ *cobra.Command, args []string) error {
	ref := args[0]

	if !foundry.IsRemoteReference(ref) {
		return fmt.Errorf("expected a remote reference (e.g. github.com/owner/repo), got %q", ref)
	}

	fmt.Println(styles.WorkingBanner("Adding ingot..."))
	fmt.Println()

	fsys, err := foundry.Resolve(ref)
	if err != nil {
		return fmt.Errorf("resolving ingot: %w", err)
	}

	// Validate ingot.yaml exists.
	ingot, err := mold.LoadIngotFromFS(fsys, "ingot.yaml")
	if err != nil {
		return fmt.Errorf("invalid ingot manifest: %w", err)
	}

	// Copy ingot into project .ailloy/ingots/<name>/
	destDir := filepath.Join(".ailloy", "ingots", ingot.Name)
	if err := os.MkdirAll(destDir, 0750); err != nil {
		return fmt.Errorf("creating ingot directory: %w", err)
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

		if err := os.WriteFile(destPath, content, 0644); err != nil { // #nosec G306 -- ingot files need to be readable
			return fmt.Errorf("writing %s: %w", destPath, err)
		}

		fmt.Println(styles.SuccessStyle.Render("  + ") + styles.CodeStyle.Render(destPath))
		return nil
	})
	if err != nil {
		return fmt.Errorf("copying ingot files: %w", err)
	}

	fmt.Println()
	fmt.Println(styles.SuccessStyle.Render("Ingot added: ") + styles.AccentStyle.Render(ingot.Name+" "+ingot.Version))
	fmt.Println(styles.InfoStyle.Render("Installed to: ") + styles.CodeStyle.Render(destDir))

	return nil
}
