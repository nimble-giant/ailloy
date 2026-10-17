package commands

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/nimble-giant/ailloy/pkg/projectstate"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

var stateCmd = &cobra.Command{
	Use:   "state",
	Short: "Locate ailloy's per-project bookkeeping",
	Long: `Locate ailloy's per-project bookkeeping.

Ailloy keeps what it needs to track a project (installed manifest, ailloy.lock,
install state, persisted project flux, installed ores and ingots) outside the
project tree, under ~/.ailloy/projects/<project>-<hash>/. Only a mold's
rendered output lands in your project.

Set AILLOY_PROJECT_STATE_DIR to override the location; a relative value
resolves against the project root (e.g. ".ailloy" to keep state, lock
included, in the project so it can be committed).

Available subcommands:
  path       Print this project's state directory
  list       List every project with ailloy state
  migrate    Move legacy in-project bookkeeping into the state directory`,
}

var statePathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print this project's state directory",
	Args:  cobra.NoArgs,
	RunE:  func(cmd *cobra.Command, _ []string) error { return runStatePath(cmd.OutOrStdout()) },
}

var stateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List every project with ailloy state",
	Args:  cobra.NoArgs,
	RunE:  func(cmd *cobra.Command, _ []string) error { return runStateList(cmd.OutOrStdout()) },
}

var stateMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Move legacy in-project bookkeeping into the state directory",
	Long: `Move legacy in-project bookkeeping into the state directory.

Moves .ailloy/{installed.yaml,state.yaml,flux,ores,ingots} and ailloy.lock out
of the project. Every ailloy command already does this on startup; this
command just makes it explicit. Files already present in the state directory
are never overwritten.`,
	Args: cobra.NoArgs,
	Run:  func(cmd *cobra.Command, _ []string) { migrateProjectState(cmd.OutOrStdout()) },
}

func init() {
	rootCmd.AddCommand(stateCmd)
	stateCmd.AddCommand(statePathCmd, stateListCmd, stateMigrateCmd)
}

func runStatePath(w io.Writer) error {
	dir, err := filepath.Abs(projectStateDir())
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, dir)
	return err
}

func runStateList(w io.Writer) error {
	projects, err := projectstate.List()
	if err != nil {
		return err
	}
	if len(projects) == 0 {
		_, err := fmt.Fprintln(w, "No project state recorded yet.")
		return err
	}
	for _, p := range projects {
		root := p.Root
		if root == "" {
			root = "(unknown project)"
		}
		if _, err := fmt.Fprintf(w, "%s\n  %s\n", root, p.Dir); err != nil {
			return err
		}
	}
	return nil
}

// skipsStateHooks reports whether a command should run without touching
// project state (shell completion and help).
func skipsStateHooks(name string) bool {
	switch name {
	case "help", "completion", "__complete", "__completeNoDesc":
		return true
	}
	return false
}

// migrateProjectState moves legacy in-project bookkeeping into the state dir
// and prints a short notice to w. Failures are reported, never fatal: the
// legacy files stay where they were.
func migrateProjectState(w io.Writer) {
	res, err := projectstate.MigrateLegacy(".")
	if len(res.Moved) > 0 {
		_, _ = fmt.Fprintln(w, styles.InfoStyle.Render("Moved ailloy bookkeeping out of this project: ")+
			strings.Join(res.Moved, ", ")+" → "+styles.CodeStyle.Render(res.Dir))
	}
	if len(res.Conflicts) > 0 {
		_, _ = fmt.Fprintln(w, styles.WarningStyle.Render("warning: ")+
			"left legacy "+strings.Join(res.Conflicts, ", ")+" in place; "+res.Dir+" already has them")
	}
	if err != nil {
		_, _ = fmt.Fprintln(w, styles.WarningStyle.Render("warning: ")+"migrating ailloy state: "+err.Error())
	}
}
