package commands

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/nimble-giant/ailloy/pkg/extensions"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

// extManager is lazily constructed once per CLI invocation. Every
// extension-related cobra runner calls extensionsManager() rather than
// importing pkg/extensions directly so the lifetime is uniform.
var extManager *extensions.Manager

func extensionsManager() (*extensions.Manager, error) {
	if extManager != nil {
		return extManager, nil
	}
	version := rootCmd.Version
	if version == "" {
		version = "dev"
	}
	m, err := extensions.NewManager(version)
	if err != nil {
		return nil, err
	}
	extManager = m
	return extManager, nil
}

var extensionsCmd = &cobra.Command{
	Use:     "extensions",
	Aliases: []string{"ext"},
	Short:   "Manage CLI extensions",
	Long: `Manage ailloy CLI extensions: list available, install, remove,
update, and reset consent.

Extensions are separate binaries that ailloy downloads on demand and
execs to provide additional commands. The first official extension is
'docs', which ships the rich in-CLI documentation TUI.

Bidirectional verbs are also supported, e.g. ailloy list extensions
or ailloy install extension docs.`,
}

var extensionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available and installed extensions",
	RunE:  runExtensionsList,
}

var extensionsInstallCmd = &cobra.Command{
	Use:   "install <name|github.com/owner/repo>",
	Short: "Install an extension",
	Args:  cobra.ExactArgs(1),
	RunE:  runExtensionsInstall,
}

var extensionsRemoveCmd = &cobra.Command{
	Use:     "remove <name>",
	Aliases: []string{"uninstall"},
	Short:   "Remove an installed extension",
	Args:    cobra.ExactArgs(1),
	RunE:    runExtensionsRemove,
}

var extensionsUpdateCmd = &cobra.Command{
	Use:   "update [name]",
	Short: "Check for and install a newer release of an extension",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runExtensionsUpdate,
}

var extensionsShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show full manifest detail for one extension",
	Args:  cobra.ExactArgs(1),
	RunE:  runExtensionsShow,
}

var (
	extensionsResetConsentOnly bool
)

var extensionsResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset extensions state (full wipe by default)",
	Long: `Reset removes installed extensions and clears the manifest.
With --consent-only, binaries are kept but the consent / declined
flags are cleared so first-run prompts re-fire.`,
	RunE: runExtensionsReset,
}

func init() {
	rootCmd.AddCommand(extensionsCmd)
	extensionsCmd.AddCommand(extensionsListCmd)
	extensionsCmd.AddCommand(extensionsInstallCmd)
	extensionsCmd.AddCommand(extensionsRemoveCmd)
	extensionsCmd.AddCommand(extensionsUpdateCmd)
	extensionsCmd.AddCommand(extensionsShowCmd)
	extensionsCmd.AddCommand(extensionsResetCmd)

	extensionsResetCmd.Flags().BoolVar(&extensionsResetConsentOnly, "consent-only",
		false, "keep installed binaries; only clear consent records")
}

func runExtensionsList(cmd *cobra.Command, _ []string) error {
	m, err := extensionsManager()
	if err != nil {
		return err
	}
	entries, err := m.List()
	if err != nil {
		return err
	}

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
		Headers("Name", "Status", "Version", "Source")

	for _, e := range entries {
		status := "available"
		if e.Installed {
			status = "installed"
		} else if e.Declined {
			status = "declined"
		}
		t.Row(e.Name, status, e.InstalledVersion, e.Source)
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), styles.AccentStyle.Render("Ailloy extensions"))
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), t.Render())
	return nil
}

func runExtensionsInstall(_ *cobra.Command, args []string) error {
	m, err := extensionsManager()
	if err != nil {
		return err
	}
	return m.Install(args[0])
}

func runExtensionsRemove(_ *cobra.Command, args []string) error {
	m, err := extensionsManager()
	if err != nil {
		return err
	}
	return m.Remove(args[0])
}

func runExtensionsUpdate(_ *cobra.Command, args []string) error {
	m, err := extensionsManager()
	if err != nil {
		return err
	}
	if len(args) == 1 {
		return m.Update(args[0])
	}
	// No name → update everything.
	entries, err := m.List()
	if err != nil {
		return err
	}
	var lastErr error
	for _, e := range entries {
		if !e.Installed {
			continue
		}
		if err := m.Update(e.Name); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func runExtensionsShow(cmd *cobra.Command, args []string) error {
	m, err := extensionsManager()
	if err != nil {
		return err
	}
	e, ok, err := m.Show(args[0])
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("extension %q is not installed", args[0])
	}
	out := cmd.OutOrStdout()
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Primary1)
	_, _ = fmt.Fprintf(out, "%s %s\n", keyStyle.Render("Name:"), e.Name)
	_, _ = fmt.Fprintf(out, "%s %s\n", keyStyle.Render("Source:"), e.Source)
	_, _ = fmt.Fprintf(out, "%s %s\n", keyStyle.Render("Version:"), e.InstalledVersion)
	if e.AilloyDocsVersion != "" {
		_, _ = fmt.Fprintf(out, "%s %s\n", keyStyle.Render("Built from ailloy docs:"), e.AilloyDocsVersion)
	}
	_, _ = fmt.Fprintf(out, "%s %s\n", keyStyle.Render("Binary:"), e.BinaryPath)
	_, _ = fmt.Fprintf(out, "%s %s\n", keyStyle.Render("SHA-256:"), e.SHA256)
	if !e.InstalledAt.IsZero() {
		_, _ = fmt.Fprintf(out, "%s %s\n", keyStyle.Render("Installed:"), e.InstalledAt.Format("2006-01-02 15:04:05 UTC"))
	}
	_, _ = fmt.Fprintf(out, "%s %v\n", keyStyle.Render("Auto-update:"), e.AutoUpdate)
	return nil
}

func runExtensionsReset(_ *cobra.Command, _ []string) error {
	m, err := extensionsManager()
	if err != nil {
		return err
	}
	return m.Reset(extensionsResetConsentOnly)
}

// ----- bidirectional verbs (used by verbs.go) -----

var listExtensionsSubCmd = &cobra.Command{
	Use:   "extensions",
	Short: "List available and installed extensions",
	RunE:  runExtensionsList,
}

var installExtensionSubCmd = &cobra.Command{
	Use:   "extension <name|github.com/owner/repo>",
	Short: "Install an ailloy extension",
	Args:  cobra.ExactArgs(1),
	RunE:  runExtensionsInstall,
}

var removeExtensionSubCmd = &cobra.Command{
	Use:   "extension <name>",
	Short: "Remove an ailloy extension",
	Args:  cobra.ExactArgs(1),
	RunE:  runExtensionsRemove,
}

var updateExtensionSubCmd = &cobra.Command{
	Use:   "extension [name]",
	Short: "Update an ailloy extension",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runExtensionsUpdate,
}

var showExtensionSubCmd = &cobra.Command{
	Use:   "extension <name>",
	Short: "Show full manifest detail for one extension",
	Args:  cobra.ExactArgs(1),
	RunE:  runExtensionsShow,
}

// helper: pretty-print a single line for non-interactive users.
func summarizeExtension(e extensions.ListEntry) string {
	status := "available"
	if e.Installed {
		status = "installed"
	} else if e.Declined {
		status = "declined"
	}
	parts := []string{e.Name, status}
	if e.InstalledVersion != "" {
		parts = append(parts, e.InstalledVersion)
	}
	return strings.Join(parts, "  ")
}

var _ = summarizeExtension // keep helper available for tests / future use
