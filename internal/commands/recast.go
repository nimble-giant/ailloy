package commands

import (
	"fmt"
	"log"
	"time"

	"github.com/nimble-giant/ailloy/internal/tui/ceremony"
	"github.com/nimble-giant/ailloy/pkg/blanks"
	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

var recastCmd = &cobra.Command{
	Use:     "recast [name]",
	Aliases: []string{"upgrade"},
	Short:   "Re-resolve installed molds to newer versions",
	Long: `Re-resolve installed molds to newer versions (alias: upgrade).

Reads .ailloy/installed.yaml and refreshes each mold to its latest matching
version. If ailloy.lock exists, also updates lock entries in lockstep.

Use --dry-run to preview changes without applying them.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runRecast,
}

var (
	recastDryRun bool
	recastGlobal bool
)

func init() {
	rootCmd.AddCommand(recastCmd)
	recastCmd.Flags().BoolVar(&recastDryRun, "dry-run", false, "preview changes without applying")
	recastCmd.Flags().BoolVarP(&recastGlobal, "global", "g", false, "operate on the global manifest/lock under ~/")
}

type recastChange struct {
	Name       string
	Source     string
	OldVersion string
	OldCommit  string
	NewVersion string
	NewCommit  string
}

func runRecast(_ *cobra.Command, args []string) error {
	manifestPath := manifestPathFor(recastGlobal)
	lockPath := lockPathFor(recastGlobal)

	manifest, err := foundry.ReadInstalledManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("reading installed manifest: %w", err)
	}
	if manifest == nil || len(manifest.Molds) == 0 {
		return fmt.Errorf("no installed manifest at %s — run %s first",
			styles.CodeStyle.Render(manifestPath),
			styles.CodeStyle.Render("ailloy cast"))
	}

	// Optional name filter.
	entries := manifest.Molds
	if len(args) == 1 {
		match := manifest.FindByName(args[0])
		if match == nil {
			return fmt.Errorf("mold %q not found in installed manifest", args[0])
		}
		entries = []foundry.InstalledEntry{*match}
	}

	if recastDryRun {
		// Dry-run keeps the lighter informational banner — no full ceremony
		// because nothing actually changes on disk.
		fmt.Println(styles.WorkingBanner("Previewing dependency updates (dry run)..."))
		fmt.Println()
	} else {
		ceremony.Open(ceremony.Recast)
	}

	git := foundry.DefaultGitRunner()
	var changes []recastChange

	// Read existing lock (if any) so we can update it in lockstep.
	existingLock, _ := foundry.ReadLockFile(lockPath)

	for _, entry := range entries {
		ref, err := referenceFromInstalledEntry(&entry)
		if err != nil {
			fmt.Printf("%s skipping %s: %v\n", styles.WarningStyle.Render("!"), entry.Name, err)
			continue
		}
		resolved, err := foundry.ResolveVersion(ref, git)
		if err != nil {
			fmt.Printf("%s skipping %s: %v\n", styles.WarningStyle.Render("!"), entry.Name, err)
			continue
		}
		if resolved.Tag == entry.Version && resolved.Commit == entry.Commit {
			fmt.Println(styles.InfoStyle.Render("  ") + entry.Name + " is already up to date (" + styles.CodeStyle.Render(entry.Version) + ")")
			continue
		}
		changes = append(changes, recastChange{
			Name:       entry.Name,
			Source:     entry.Source,
			OldVersion: entry.Version,
			OldCommit:  entry.Commit,
			NewVersion: resolved.Tag,
			NewCommit:  resolved.Commit,
		})

		if !recastDryRun {
			fetcher, ferr := foundry.NewFetcher(git)
			if ferr != nil {
				fmt.Printf("%s skipping %s: fetcher: %v\n", styles.WarningStyle.Render("!"), entry.Name, ferr)
				changes = changes[:len(changes)-1]
				continue
			}
			fetchedFS, _, fetchErr := fetcher.Fetch(ref, resolved)
			if fetchErr != nil {
				fmt.Printf("%s skipping %s: fetch: %v\n", styles.WarningStyle.Render("!"), entry.Name, fetchErr)
				changes = changes[:len(changes)-1]
				continue
			}

			manifest.UpsertEntry(foundry.InstalledEntry{
				Name:    entry.Name,
				Source:  entry.Source,
				Subpath: entry.Subpath,
				Version: resolved.Tag,
				Commit:  resolved.Commit,
				CastAt:  time.Now().UTC(),
			})

			// Reconcile the freshly resolved mold's dependency graph: install
			// any newly declared deps and prune any that the mold no longer
			// declares. moldKey mirrors the cast-time key (source@subpath when
			// subpath is set) so dependent strings stay consistent.
			moldKey := entry.Source
			if entry.Subpath != "" {
				moldKey += "@" + entry.Subpath
			}
			reader := blanks.NewMoldReader(fetchedFS)
			freshMold, mErr := reader.LoadManifest()
			if mErr != nil {
				log.Printf("warning: loading fresh mold manifest for %s: %v", entry.Name, mErr)
			} else if freshMold != nil {
				// Auto-install newly declared deps. Recast operates on the
				// project's installed.yaml (or global per --global). Local-path
				// deps are refused because recast walks remote references.
				if err := installDeclaredDeps(freshMold, moldKey, recastGlobal, false); err != nil {
					log.Printf("warning: installing deps for %s: %v", entry.Name, err)
				}
				// Cascade-prune deps the mold no longer declares.
				if err := pruneRemovedDeps(manifestPathFor(recastGlobal), moldKey, freshMold.Dependencies, recastGlobal); err != nil {
					log.Printf("warning: pruning removed deps for %s: %v", entry.Name, err)
				}
				// Re-read manifest so subsequent mold iterations see the
				// changes from installDeclaredDeps + pruneRemovedDeps.
				if reread, _ := foundry.ReadInstalledManifest(manifestPathFor(recastGlobal)); reread != nil {
					manifest = reread
				}
			}

			if existingLock != nil {
				existingLock.UpsertEntry(foundry.LockEntry{
					Name:      entry.Name,
					Source:    entry.Source,
					Version:   resolved.Tag,
					Commit:    resolved.Commit,
					Subpath:   entry.Subpath,
					Timestamp: time.Now().UTC(),
				})
			}
		}
	}

	if len(changes) == 0 {
		fmt.Println()
		fmt.Println(styles.SuccessStyle.Render("All dependencies are up to date."))
		return nil
	}

	if !recastDryRun {
		if err := foundry.WriteInstalledManifest(manifestPath, manifest); err != nil {
			return fmt.Errorf("writing installed manifest: %w", err)
		}
		if existingLock != nil {
			if err := foundry.WriteLockFile(lockPath, existingLock); err != nil {
				return fmt.Errorf("writing lock file: %w", err)
			}
		}
	}

	fmt.Println()
	if recastDryRun {
		fmt.Println(styles.InfoStyle.Render("Changes that would be applied:"))
	} else {
		fmt.Println(styles.SuccessStyle.Render("Updated dependencies:"))
	}
	fmt.Println()
	for _, c := range changes {
		fmt.Printf("  %s  %s %s %s\n",
			styles.FoxBullet(c.Name),
			styles.CodeStyle.Render(c.OldVersion),
			styles.InfoStyle.Render("->"),
			styles.CodeStyle.Render(c.NewVersion),
		)
	}

	fmt.Println()
	if recastDryRun {
		fmt.Println(styles.InfoStyle.Render("Run without --dry-run to apply these changes."))
	} else {
		fmt.Println(styles.SuccessBanner("Recast complete!"))
		ceremony.Stamp(ceremony.Recast, fmt.Sprintf("%d mold(s) updated", len(changes)))
	}
	return nil
}
