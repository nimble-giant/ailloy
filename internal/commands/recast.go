package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

var recastCmd = &cobra.Command{
	Use:     "recast [name|source[//subpath]]",
	Aliases: []string{"upgrade"},
	Short:   "Re-resolve installed molds to newer versions",
	Long: `Re-resolve installed molds to newer versions (alias: upgrade).

Reads .ailloy/installed.yaml and refreshes each mold to its latest matching
version. If ailloy.lock exists, also updates lock entries in lockstep.

A foundry repo may host multiple molds at different subpaths that declare the
same name. Pass the full ref (e.g. github.com/owner/repo//molds/shortcut) to
disambiguate; bare names are accepted when only one matching entry exists.

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

	// Optional filter: bare name, or full source[//subpath] ref.
	entries := manifest.Molds
	if len(args) == 1 {
		match, err := resolveRecastTarget(manifest, args[0])
		if err != nil {
			return err
		}
		entries = []foundry.InstalledEntry{*match}
	}

	if recastDryRun {
		fmt.Println(styles.WorkingBanner("Previewing dependency updates (dry run)..."))
	} else {
		fmt.Println(styles.WorkingBanner("Recasting dependencies..."))
	}
	fmt.Println()

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
			if _, _, fetchErr := fetcher.Fetch(ref, resolved); fetchErr != nil {
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
	}
	return nil
}

// resolveRecastTarget interprets the user's positional argument as either a
// bare mold name or a full source[//subpath] reference, and returns the
// single matching installed entry. A bare name with multiple matches (same
// name declared in two molds at different subpaths in the same foundry) is
// rejected with a disambiguation list so the user can re-run with the full
// ref.
func resolveRecastTarget(manifest *foundry.InstalledManifest, raw string) (*foundry.InstalledEntry, error) {
	// If the arg parses as a remote-style reference, look up by (source, subpath).
	if foundry.IsRemoteReference(raw) {
		ref, err := foundry.ParseReference(raw)
		if err != nil {
			return nil, fmt.Errorf("parsing %q: %w", raw, err)
		}
		match := manifest.FindBySource(ref.CacheKey(), ref.Subpath)
		if match == nil {
			return nil, fmt.Errorf("mold %q not found in installed manifest", raw)
		}
		return match, nil
	}

	matches := manifest.FindAllByName(raw)
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("mold %q not found in installed manifest", raw)
	case 1:
		return matches[0], nil
	default:
		var lines []string
		for _, m := range matches {
			ref := m.Source
			if m.Subpath != "" {
				ref += "//" + m.Subpath
			}
			lines = append(lines, "  - "+ref)
		}
		return nil, fmt.Errorf("multiple installed molds named %q; specify the full ref:\n%s", raw, strings.Join(lines, "\n"))
	}
}
