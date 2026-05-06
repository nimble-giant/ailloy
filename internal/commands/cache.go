package commands

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/foundry/index"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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
	Args:          cobra.NoArgs,
	RunE:          runCacheClear,
	SilenceErrors: true,
	SilenceUsage:  true,
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
	moldRoot, err := foundry.CacheDir()
	if err != nil {
		return err
	}
	indexRoot, err := index.IndexCacheDir()
	if err != nil {
		return err
	}
	exit, runErr := executeCacheClear(cacheClearOptions{
		MoldRoot:  moldRoot,
		IndexRoot: indexRoot,
		Molds:     cacheClearMolds,
		Indexes:   cacheClearIndexes,
		DryRun:    cacheClearDryRun,
		Yes:       cacheClearYes,
		Stdout:    cmd.OutOrStdout(),
		Stdin:     cmd.InOrStdin(),
		IsTTY:     stdinIsTTY,
	})
	if runErr != nil {
		return runErr
	}
	if exit != 0 {
		return fmt.Errorf("cache clear completed with errors")
	}
	return nil
}

type cacheClearOptions struct {
	MoldRoot  string
	IndexRoot string

	Molds   bool
	Indexes bool
	DryRun  bool
	Yes     bool

	Stdout io.Writer
	Stdin  io.Reader
	IsTTY  func() bool
}

func executeCacheClear(o cacheClearOptions) (int, error) {
	wantMolds, wantIndexes := o.Molds, o.Indexes
	if !wantMolds && !wantIndexes {
		wantMolds, wantIndexes = true, true
	}

	var (
		molds *moldStats
		idx   *indexStats
	)
	if wantMolds {
		s, err := gatherMoldStats(o.MoldRoot, o.IndexRoot)
		if err != nil {
			return 1, fmt.Errorf("gather mold stats: %w", err)
		}
		molds = &s
	}
	if wantIndexes {
		s, err := gatherIndexStats(o.IndexRoot)
		if err != nil {
			return 1, fmt.Errorf("gather index stats: %w", err)
		}
		idx = &s
	}

	if isEmptySelection(molds, idx) {
		fmt.Fprintln(o.Stdout, "Cache is already empty.")
		return 0, nil
	}

	fmt.Fprint(o.Stdout, renderCachePreview(displayPath(o.MoldRoot), molds, idx))

	if o.DryRun {
		return 0, nil
	}

	if !o.Yes {
		if !o.IsTTY() {
			return 1, fmt.Errorf("refusing to clear cache without --yes in non-interactive shell")
		}
		ok, err := confirmInteractive(o.Stdin, o.Stdout, "\nProceed? [y/N] ")
		if err != nil {
			return 1, err
		}
		if !ok {
			fmt.Fprintln(o.Stdout, "Cancelled.")
			return 0, nil
		}
	}

	var (
		removedIndexes int
		errs           []error
	)
	if wantMolds {
		_, e := removeMolds(o.MoldRoot)
		errs = append(errs, e...)
	}
	if wantIndexes && idx != nil {
		removedIndexes = idx.Indexes
		if err := removeIndexes(o.IndexRoot); err != nil {
			errs = append(errs, err)
		}
	}

	for _, e := range errs {
		fmt.Fprintf(o.Stdout, "warning: %s\n", e.Error())
	}

	freed := int64(0)
	if molds != nil {
		freed += molds.Bytes
	}
	if idx != nil {
		freed += idx.Bytes
	}

	switch {
	case wantMolds && wantIndexes:
		refs := 0
		versions := 0
		if molds != nil {
			refs = molds.Refs
			versions = molds.Versions
		}
		fmt.Fprintf(o.Stdout, "Cleared %d molds (%d versions), %d indexes — freed %s.\n",
			refs, versions, removedIndexes, humanizeBytes(freed))
	case wantMolds:
		refs := 0
		versions := 0
		if molds != nil {
			refs = molds.Refs
			versions = molds.Versions
		}
		fmt.Fprintf(o.Stdout, "Cleared %d molds (%d versions) — freed %s.\n",
			refs, versions, humanizeBytes(freed))
	case wantIndexes:
		fmt.Fprintf(o.Stdout, "Cleared %d indexes — freed %s.\n",
			removedIndexes, humanizeBytes(freed))
	}

	if len(errs) > 0 {
		fmt.Fprintf(o.Stdout, "Cleared with %d errors.\n", len(errs))
		return 1, nil
	}
	return 0, nil
}

func isEmptySelection(molds *moldStats, idx *indexStats) bool {
	moldsEmpty := molds == nil || (molds.Refs == 0 && molds.Bytes == 0)
	idxEmpty := idx == nil || (idx.Indexes == 0 && idx.Bytes == 0)
	return moldsEmpty && idxEmpty
}

func displayPath(p string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return p
	}
	if p == home {
		return "~"
	}
	sep := string(os.PathSeparator)
	if strings.HasPrefix(p, home+sep) {
		return "~" + strings.TrimPrefix(p, home)
	}
	return p
}

func stdinIsTTY() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
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
	for _, e := range entries {
		if e.Host == "indexes" {
			continue
		}
		stats.Refs++
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

func removeIndexes(indexRoot string) error {
	if err := os.RemoveAll(indexRoot); err != nil {
		return fmt.Errorf("remove %s: %w", indexRoot, err)
	}
	return nil
}

func renderCachePreview(cacheRootDisplay string, molds *moldStats, idx *indexStats) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Ailloy cache:  %s\n\n", cacheRootDisplay)

	var totalBytes int64
	if molds != nil {
		fmt.Fprintf(&b, "  Molds      %d refs, %d versions  (%s)\n",
			molds.Refs, molds.Versions, humanizeBytes(molds.Bytes))
		totalBytes += molds.Bytes
	}
	if idx != nil {
		fmt.Fprintf(&b, "  Indexes    %d indexes              (%s)\n",
			idx.Indexes, humanizeBytes(idx.Bytes))
		totalBytes += idx.Bytes
	}

	if molds != nil && idx != nil {
		fmt.Fprintf(&b, "\n  Total:                            (%s)\n", humanizeBytes(totalBytes))
	}
	return b.String()
}

func confirmInteractive(in io.Reader, out io.Writer, prompt string) (bool, error) {
	if _, err := fmt.Fprint(out, prompt); err != nil {
		return false, err
	}
	r := bufio.NewReader(in)
	line, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}
	resp := strings.ToLower(strings.TrimSpace(line))
	return resp == "y" || resp == "yes", nil
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
