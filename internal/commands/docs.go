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

// rootDocs is the global --docs flag set by users on any command. When true
// the persistent pre-run renders the command's associated topic and exits
// before the command's RunE fires.
var rootDocs bool

const (
	defaultDocsWidth = 100
	maxDocsWidth     = 120
	minDocsWidth     = 40
)

var docsCmd = &cobra.Command{
	Use:   "docs [topic]",
	Short: "Render embedded ailloy documentation in the terminal",
	Long: `Render embedded ailloy documentation in the terminal.

Run with no arguments to list available topics. Pass a topic slug to
render that document via glamour.

Examples:
  ailloy docs                  # list topics
  ailloy docs flux             # show the flux variable reference
  ailloy cast --docs           # show the doc associated with ` + "`cast`" + ``,
	Args: cobra.MaximumNArgs(1),
	RunE: runDocs,
}

func init() {
	rootCmd.AddCommand(docsCmd)
	rootCmd.PersistentFlags().BoolVar(&rootDocs, "docs", false, "render the command's associated documentation and exit")

	// Hook the persistent pre-run so any command can be invoked with --docs.
	// We chain into the existing PersistentPreRun rather than replacing it.
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
		// Avoid recursion when `ailloy docs --docs` is run.
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
		// We can't return a sentinel error without tripping cobra's error path.
		cmd.RunE = func(*cobra.Command, []string) error { return nil }
		cmd.Run = nil
		cmd.PostRun = nil
		cmd.PostRunE = nil
		return nil
	}
}

func runDocs(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return printTopicList(cmd.OutOrStdout())
	}
	return renderTopicTo(cmd.OutOrStdout(), args[0])
}

// topicForCommand returns the topic slug associated with a cobra command.
// It walks up parents so that `ailloy foundry add --docs` falls back to the
// `foundry` topic when no entry exists for `foundry add`.
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
		" or pass "+styles.CodeStyle.Render("--docs")+" to any command.")
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

// renderMarkdown renders markdown via glamour using auto-detected style and
// terminal width.
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

// docsRenderWidth picks a sensible word-wrap width for glamour based on the
// terminal size, capped to keep long lines comfortably readable.
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
