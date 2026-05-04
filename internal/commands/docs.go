package commands

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	clidocs "github.com/nimble-giant/ailloy/docs"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// rootDocs is the global --docs flag set by users on any command. When
// true the persistent pre-run renders the command's associated topic
// to stdout (always the in-binary fallback path) and short-circuits.
var rootDocs bool

// docsExtensionName is the canonical extension that owns the rich docs
// experience. ailloy execs it whenever it's installed; otherwise the
// in-binary fallback (glamour-to-stdout) handles `ailloy docs`.
const docsExtensionName = "docs"

const (
	defaultDocsWidth = 100
	maxDocsWidth     = 120
	minDocsWidth     = 40
)

var (
	docsListOnly    bool
	docsNoExtension bool
	docsAutoApprove bool
)

var docsCmd = &cobra.Command{
	Use:   "docs [topic]",
	Short: "Browse ailloy documentation",
	Long: `Browse ailloy documentation.

The first time you run ailloy docs, ailloy offers to install the docs
extension — a separate binary that ships the rich in-CLI TUI with
pre-rendered glamour output for instant page loads. Subsequent runs
exec that binary directly. Decline the install, run with --no-extension,
or use a non-interactive shell to get the always-available stdout-based
fallback.

Examples:
  ailloy docs                  # launch the rich TUI (extension) or fall back
  ailloy docs flux             # render a topic to stdout
  ailloy docs --list           # list available topics
  ailloy docs --no-extension   # force the in-binary fallback
  ailloy cast --docs           # render the doc associated with cast`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDocs,
}

func init() {
	rootCmd.AddCommand(docsCmd)
	docsCmd.Flags().BoolVar(&docsListOnly, "list", false, "list topics as a table")
	docsCmd.Flags().BoolVar(&docsNoExtension, "no-extension", false,
		"skip the docs extension; use the in-binary fallback")
	docsCmd.Flags().BoolVar(&docsAutoApprove, "yes", false,
		"auto-approve the first-run extension install prompt")
	rootCmd.PersistentFlags().BoolVar(&rootDocs, "docs", false,
		"render the command's associated documentation and exit")

	prev := rootCmd.PersistentPreRun
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if prev != nil {
			prev(cmd, args)
		}
	}
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if !rootDocs {
			return nil
		}
		if cmd == docsCmd {
			return nil
		}
		topic := topicForCommand(cmd)
		if topic == "" {
			return fmt.Errorf("no documentation topic registered for %q (run `ailloy docs` to list topics)", cmd.CommandPath())
		}
		if err := renderTopicTo(cmd.OutOrStdout(), topic); err != nil {
			return err
		}
		// Short-circuit: replace the command's body so its action is skipped.
		cmd.RunE = func(*cobra.Command, []string) error { return nil }
		cmd.Run = nil
		cmd.PostRun = nil
		cmd.PostRunE = nil
		return nil
	}
}

func runDocs(cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()

	// Explicit list flag → always print the table from the in-binary docs.
	if docsListOnly {
		return printTopicList(out)
	}

	// Topic argument → always render to stdout via the in-binary fallback.
	// Useful for piping (`ailloy docs flux | less`) and predictable in CI.
	if len(args) == 1 {
		return renderTopicTo(out, args[0])
	}

	// Non-interactive contexts (tests, pipes, CI) skip the extension — its
	// output bypasses cmd.OutOrStdout() and goes straight to the real fd,
	// which breaks captured-stdout tests and pipe consumers.
	if !isInteractive() {
		return printTopicList(out)
	}

	// Bare `ailloy docs`. Prefer the extension when allowed.
	if !docsNoExtension {
		if code, ran, err := tryExecDocsExtension(args); ran {
			if err != nil {
				return err
			}
			if code != 0 {
				os.Exit(code)
			}
			return nil
		}
	}

	return printTopicList(out)
}

// tryExecDocsExtension handles the bare `ailloy docs` happy path. Returns
// (exitCode, ran=true, err) when the extension was used (whether
// installed already or freshly installed via prompt). Returns ran=false
// when the caller should fall back to the in-binary path.
func tryExecDocsExtension(args []string) (int, bool, error) {
	mgr, err := extensionsManager()
	if err != nil {
		return 0, false, nil
	}

	if mgr.IsInstalled(docsExtensionName) {
		code, err := mgr.Run(docsExtensionName, args)
		return code, true, err
	}

	if mgr.IsDeclined(docsExtensionName) {
		return 0, false, nil
	}

	approved := docsAutoApprove
	if !approved {
		ok, err := mgr.PromptInstall(docsExtensionName)
		if err != nil {
			return 0, false, err
		}
		approved = ok
	}
	if !approved {
		fmt.Fprintln(os.Stderr,
			"Using the in-binary docs fallback. Install the extension anytime with `ailloy ext install docs`.")
		return 0, false, nil
	}
	if err := mgr.Install(docsExtensionName); err != nil {
		fmt.Fprintln(os.Stderr,
			"Couldn't install the docs extension; using the in-binary fallback.")
		fmt.Fprintln(os.Stderr, "  ", err)
		return 0, false, nil
	}
	code, err := mgr.Run(docsExtensionName, args)
	return code, true, err
}

// isInteractive reports whether stdout AND stdin are attached to a TTY.
func isInteractive() bool {
	return term.IsTerminal(int(os.Stdout.Fd())) && term.IsTerminal(int(os.Stdin.Fd()))
}

// topicForCommand walks up parents so that `ailloy foundry add --docs`
// falls back to the `foundry` topic when no entry exists for `foundry add`.
func topicForCommand(cmd *cobra.Command) string {
	for c := cmd; c != nil; c = c.Parent() {
		if slug, ok := clidocs.CommandTopic[c.Name()]; ok {
			return slug
		}
	}
	return ""
}

func printTopicList(w io.Writer) error {
	topics := clidocs.List()

	header := lipgloss.NewStyle().Bold(true).Foreground(styles.Primary1)
	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(styles.Primary1)).
		StyleFunc(func(row, _ int) lipgloss.Style {
			if row == table.HeaderRow {
				return header
			}
			return lipgloss.NewStyle()
		}).
		Headers("Topic", "Description")

	for _, topic := range topics {
		t.Row(topic.Slug, topic.Summary)
	}

	if _, err := fmt.Fprintln(w, styles.AccentStyle.Render("Available documentation topics")); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, t.Render()); err != nil {
		return err
	}
	_, err := fmt.Fprintln(w, "Render a topic with "+styles.CodeStyle.Render("ailloy docs <topic>")+
		" or install the docs extension for a richer experience: "+
		styles.CodeStyle.Render("ailloy ext install docs"))
	return err
}

func renderTopicTo(w io.Writer, slug string) error {
	body, err := clidocs.Read(slug)
	if err != nil {
		return err
	}
	rendered, err := renderMarkdown(string(body))
	if err != nil {
		return fmt.Errorf("rendering %s: %w", slug, err)
	}
	_, err = io.WriteString(w, rendered)
	return err
}

// renderMarkdown is the in-binary fallback renderer.
func renderMarkdown(md string) (string, error) {
	width := docsRenderWidth()
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return "", err
	}
	defer func() { _ = r.Close() }()
	return r.Render(md)
}

func docsRenderWidth() int {
	width := defaultDocsWidth
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		width = w
	}
	if width > maxDocsWidth {
		width = maxDocsWidth
	}
	if width < minDocsWidth {
		width = minDocsWidth
	}
	return width
}
