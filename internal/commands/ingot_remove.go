package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

var (
	ingotRemoveForce  bool
	ingotRemoveGlobal bool
)

var ingotRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove an installed ingot",
	Long: `Remove an installed ingot.

Refuses to remove if other molds still depend on this ingot, unless --force
is passed. To uninstall a mold and its unshared dependents, use 'ailloy uninstall <mold>'.`,
	Args: cobra.ExactArgs(1),
	RunE: runIngotRemove,
}

func init() {
	ingotCmd.AddCommand(ingotRemoveCmd)
	ingotRemoveCmd.Flags().BoolVar(&ingotRemoveForce, "force", false, "remove even if other molds depend on this ingot")
	ingotRemoveCmd.Flags().BoolVar(&ingotRemoveGlobal, "global", false, "remove from ~/.ailloy/ingots/ instead of ./.ailloy/ingots/")
}

func runIngotRemove(_ *cobra.Command, args []string) error {
	name := args[0]
	manifestPath := manifestPathFor(ingotRemoveGlobal)
	if manifestPath == "" {
		return fmt.Errorf("cannot determine manifest path for global scope")
	}
	manifest, err := foundry.ReadInstalledManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("reading installed manifest: %w", err)
	}
	if manifest == nil {
		return fmt.Errorf("no ingots installed")
	}
	entry := manifest.FindArtifact("ingot", name)
	if entry == nil {
		return fmt.Errorf("ingot %q not installed", name)
	}
	hasMoldDeps := false
	for _, dep := range entry.Dependents {
		if dep != "user" {
			hasMoldDeps = true
			break
		}
	}
	if hasMoldDeps && !ingotRemoveForce {
		var molds []string
		for _, dep := range entry.Dependents {
			if dep != "user" {
				molds = append(molds, dep)
			}
		}
		return fmt.Errorf("cannot remove ingot %q: still required by mold(s) %v; use --force to override or 'ailloy uninstall' on the mold", name, molds)
	}

	baseDir := filepath.Join(".ailloy", "ingots", name)
	if ingotRemoveGlobal {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("determining home directory: %w", err)
		}
		baseDir = filepath.Join(home, ".ailloy", "ingots", name)
	}
	if err := os.RemoveAll(baseDir); err != nil {
		return fmt.Errorf("removing %s: %w", baseDir, err)
	}
	// Drop entry from manifest. Ingots have no Alias, so match strictly by Name.
	kept := manifest.Ingots[:0]
	for _, e := range manifest.Ingots {
		if e.Name != name {
			kept = append(kept, e)
		}
	}
	manifest.Ingots = kept
	if err := foundry.WriteInstalledManifest(manifestPath, manifest); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}
	fmt.Println(styles.SuccessStyle.Render("Ingot removed: ") + styles.AccentStyle.Render(name))

	// Drop from lock if present (project scope only).
	if !ingotRemoveGlobal {
		if lock, _ := foundry.ReadLockFile(foundry.LockFileName); lock != nil {
			dropIngotLockEntry(lock, name)
			_ = foundry.WriteLockFile(foundry.LockFileName, lock)
		}
	}
	return nil
}

func dropIngotLockEntry(lock *foundry.LockFile, name string) {
	kept := lock.Ingots[:0]
	for _, e := range lock.Ingots {
		if e.Name != name {
			kept = append(kept, e)
		}
	}
	lock.Ingots = kept
}
