package commands

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nimble-giant/ailloy/internal/tui/foundries"
	"github.com/nimble-giant/ailloy/pkg/foundry/index"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var foundriesCmd = &cobra.Command{
	Use:   "foundries",
	Short: "Open the interactive foundries TUI",
	Long: `Open an interactive interface for discovering, installing,
managing, and auditing foundries and casted molds (ailloys).

The TUI has four tabs:
  Discover   browse and multi-install molds across registered foundries
  Installed  manage casted molds (project + global), update or uninstall
  Foundries  add/remove/refresh registered foundries
  Health     drift checks and assay findings on rendered files`,
	RunE: runFoundries,
}

func init() {
	rootCmd.AddCommand(foundriesCmd)
}

func runFoundries(_ *cobra.Command, _ []string) error {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return fmt.Errorf("ailloy foundries requires a TTY; use ailloy foundry list/search/etc. for scripts")
	}
	deps := foundries.Deps{
		Cast: func(ctx context.Context, source string, opts foundries.CastOptions) (foundries.CastResult, error) {
			r, err := CastMold(ctx, source, CastOptions{
				Global:        opts.Global,
				WithWorkflows: opts.WithWorkflows,
				ValueFiles:    opts.ValueFiles,
				SetOverrides:  opts.SetOverrides,
			})
			return foundries.CastResult{
				Source:     r.Source,
				MoldName:   r.MoldName,
				FilesCast:  r.FilesCast,
				Dirs:       r.Dirs,
				GlobalRoot: r.GlobalRoot,
			}, err
		},
		AddFoundry: func(cfg *index.Config, url string) (foundries.AddFoundryResult, error) {
			r, err := AddFoundryCore(cfg, url)
			return foundries.AddFoundryResult{Entry: r.Entry, AlreadyExists: r.AlreadyExists, MoldCount: r.MoldCount}, err
		},
		RemoveFoundry: func(cfg *index.Config, nameOrURL string) (index.FoundryEntry, error) {
			return RemoveFoundryCore(cfg, nameOrURL)
		},
		UpdateFoundries: func(cfg *index.Config) ([]foundries.UpdateFoundryReport, error) {
			rs, err := UpdateFoundriesCore(cfg)
			out := make([]foundries.UpdateFoundryReport, 0, len(rs))
			for _, r := range rs {
				out = append(out, foundries.UpdateFoundryReport{
					Name: r.Name, URL: r.URL, MoldCount: r.MoldCount, Persisted: r.Persisted, Err: r.Err,
				})
			}
			return out, err
		},
		InstallFoundry: func(ctx context.Context, cfg *index.Config, nameOrURL string, opts foundries.InstallFoundryOptions) ([]foundries.InstallFoundryReport, error) {
			rs, _, err := InstallFoundryCore(ctx, cfg, nameOrURL, InstallFoundryOptions{
				Global:        opts.Global,
				WithWorkflows: opts.WithWorkflows,
				DryRun:        opts.DryRun,
				Force:         opts.Force,
			})
			out := make([]foundries.InstallFoundryReport, 0, len(rs))
			for _, r := range rs {
				out = append(out, foundries.InstallFoundryReport{
					Name: r.Name, Source: r.Source, Skipped: r.Skipped, Err: r.Err, Version: r.Version,
				})
			}
			return out, err
		},
	}
	app := foundries.New(deps)
	prog := tea.NewProgram(app, tea.WithAltScreen())
	_, err := prog.Run()
	return err
}
