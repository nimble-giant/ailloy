package commands

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/spf13/cobra"
)

var (
	cacheClearMolds   bool
	cacheClearIndexes bool
	cacheClearDryRun  bool
	cacheClearYes     bool
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage ailloy's on-disk cache",
	Long: `Manage ailloy's on-disk cache.

Available subcommands:
  clear      Clear cached mold artifacts and foundry indexes`,
}

var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear ailloy's on-disk cache",
	Long: `Clear ailloy's on-disk cache.

By default, clears both mold artifacts and foundry indexes under ~/.ailloy/cache.
Use --molds or --indexes to narrow the scope. Use --dry-run to preview without
deleting. Use --yes to skip the confirmation prompt.`,
	Args: cobra.NoArgs,
	RunE: runCacheClear,
}

func init() {
	rootCmd.AddCommand(cacheCmd)
	cacheCmd.AddCommand(cacheClearCmd)

	registerCacheClearFlags(cacheClearCmd)
}

func registerCacheClearFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&cacheClearMolds, "molds", false, "clear only the mold artifact cache")
	cmd.Flags().BoolVar(&cacheClearIndexes, "indexes", false, "clear only the foundry index cache")
	cmd.Flags().BoolVar(&cacheClearDryRun, "dry-run", false, "preview what would be cleared without deleting")
	cmd.Flags().BoolVarP(&cacheClearYes, "yes", "y", false, "skip the confirmation prompt")
}

func runCacheClear(cmd *cobra.Command, _ []string) error {
	// Implemented in Task 9.
	return nil
}

type moldStats struct {
	Refs     int
	Versions int
	Bytes    int64
}

func gatherMoldStats(moldRoot, indexRoot string) (moldStats, error) {
	var stats moldStats

	entries, err := foundry.ListCachedMolds(moldRoot)
	if err != nil {
		return stats, err
	}
	stats.Refs = len(entries)
	for _, e := range entries {
		stats.Versions += len(e.Versions)
	}

	indexAbs, err := filepath.Abs(indexRoot)
	if err != nil {
		return stats, fmt.Errorf("resolving index root: %w", err)
	}

	walkErr := filepath.WalkDir(moldRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			abs, absErr := filepath.Abs(path)
			if absErr != nil {
				return absErr
			}
			if abs == indexAbs {
				return fs.SkipDir
			}
			return nil
		}
		info, ierr := d.Info()
		if ierr != nil {
			return ierr
		}
		stats.Bytes += info.Size()
		return nil
	})
	if walkErr != nil && !errors.Is(walkErr, fs.ErrNotExist) {
		return stats, walkErr
	}
	return stats, nil
}

type indexStats struct {
	Indexes int
	Bytes   int64
}

func gatherIndexStats(indexRoot string) (indexStats, error) {
	var stats indexStats

	if _, err := os.Stat(indexRoot); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return stats, nil
		}
		return stats, err
	}

	walkErr := filepath.WalkDir(indexRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if _, statErr := os.Stat(filepath.Join(path, "foundry.yaml")); statErr == nil {
				stats.Indexes++
			}
			return nil
		}
		info, ierr := d.Info()
		if ierr != nil {
			return ierr
		}
		stats.Bytes += info.Size()
		return nil
	})
	if walkErr != nil && !errors.Is(walkErr, fs.ErrNotExist) {
		return stats, walkErr
	}
	return stats, nil
}

func removeMolds(moldRoot string) (int, []error) {
	entries, err := os.ReadDir(moldRoot)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return 0, nil
		}
		return 0, []error{err}
	}
	var (
		removed int
		errs    []error
	)
	for _, e := range entries {
		if e.Name() == "indexes" {
			continue
		}
		p := filepath.Join(moldRoot, e.Name())
		if rmErr := os.RemoveAll(p); rmErr != nil {
			errs = append(errs, fmt.Errorf("remove %s: %w", p, rmErr))
			continue
		}
		removed++
	}
	return removed, errs
}

func humanizeBytes(n int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)
	switch {
	case n >= gb:
		return fmt.Sprintf("%.1f GB", float64(n)/float64(gb))
	case n >= mb:
		return fmt.Sprintf("%.1f MB", float64(n)/float64(mb))
	case n >= kb:
		return fmt.Sprintf("%.1f KB", float64(n)/float64(kb))
	default:
		return fmt.Sprintf("%d B", n)
	}
}
