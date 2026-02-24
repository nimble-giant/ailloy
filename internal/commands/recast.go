package commands

import (
	"fmt"
	"time"

	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

var recastCmd = &cobra.Command{
	Use:     "recast [name]",
	Aliases: []string{"upgrade"},
	Short:   "Re-resolve dependencies to newer versions",
	Long: `Re-resolve dependencies to newer versions (alias: upgrade).

Fetches the latest available versions for all locked dependencies (or a single
named dependency) from their SCM sources and updates ailloy.lock.

Use --dry-run to preview changes without applying them.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runRecast,
}

var recastDryRun bool

func init() {
	rootCmd.AddCommand(recastCmd)
	recastCmd.Flags().BoolVar(&recastDryRun, "dry-run", false, "preview changes without applying")
}

// recastChange records a version change for a single dependency.
type recastChange struct {
	Name       string
	Source     string
	OldVersion string
	OldCommit  string
	NewVersion string
	NewCommit  string
}

func runRecast(_ *cobra.Command, args []string) error {
	lock, err := foundry.ReadLockFile(foundry.LockFileName)
	if err != nil {
		return fmt.Errorf("reading lock file: %w", err)
	}
	if lock == nil || len(lock.Molds) == 0 {
		return fmt.Errorf("no lock file found — run %s first", styles.CodeStyle.Render("ailloy cast"))
	}

	// Filter to a single dependency if name provided.
	var entries []foundry.LockEntry
	if len(args) == 1 {
		entry := lock.FindEntryByName(args[0])
		if entry == nil {
			return fmt.Errorf("dependency %q not found in lock file", args[0])
		}
		entries = []foundry.LockEntry{*entry}
	} else {
		entries = lock.Molds
	}

	if recastDryRun {
		fmt.Println(styles.WorkingBanner("Previewing dependency updates (dry run)..."))
	} else {
		fmt.Println(styles.WorkingBanner("Recasting dependencies..."))
	}
	fmt.Println()

	git := foundry.DefaultGitRunner()
	var changes []recastChange

	for _, entry := range entries {
		ref, err := foundry.ReferenceFromEntry(&entry)
		if err != nil {
			fmt.Printf("%s skipping %s: %v\n", styles.WarningStyle.Render("⚠️"), entry.Name, err)
			continue
		}

		resolved, err := foundry.ResolveVersion(ref, git)
		if err != nil {
			fmt.Printf("%s skipping %s: %v\n", styles.WarningStyle.Render("⚠️"), entry.Name, err)
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
			// Invalidate cached version so next cast fetches fresh content.
			fetcher, fErr := foundry.NewFetcher(git)
			if fErr == nil {
				// Fetch the new version into cache.
				_, _ = fetcher.Fetch(ref, resolved)
			}

			lock.UpsertEntry(foundry.LockEntry{
				Name:      entry.Name,
				Source:    entry.Source,
				Version:   resolved.Tag,
				Commit:    resolved.Commit,
				Subpath:   entry.Subpath,
				Timestamp: time.Now().UTC(),
			})
		}
	}

	if len(changes) == 0 {
		fmt.Println()
		fmt.Println(styles.SuccessStyle.Render("All dependencies are up to date."))
		return nil
	}

	// Write updated lock file.
	if !recastDryRun {
		if err := foundry.WriteLockFile(foundry.LockFileName, lock); err != nil {
			return fmt.Errorf("writing lock file: %w", err)
		}
	}

	// Print change summary.
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
	}

	return nil
}
