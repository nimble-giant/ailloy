package foundries

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/nimble-giant/ailloy/pkg/styles"
)

// Tab strip + body styles use the Ailloy brand palette from pkg/styles:
//   - active tab: orange/fox accent on transparent bg, bold + underline
//   - inactive: subtle gray
//   - body: soft purple-bordered box
var (
	tabActive = lipgloss.NewStyle().
			Foreground(styles.Accent1).
			Bold(true).
			Underline(true).
			Padding(0, 1)

	tabInactive = lipgloss.NewStyle().
			Foreground(styles.Gray).
			Padding(0, 1)

	statusBar = lipgloss.NewStyle().
			Foreground(styles.LightGray).
			MarginTop(1)

	bodyBox = lipgloss.NewStyle().
		Padding(1, 2)
)

// StyleSet bundles the brand-themed lipgloss styles each tab sub-model uses
// so we don't reach into pkg/styles from every package.
type StyleSet struct {
	Heading     lipgloss.Style
	Cursor      lipgloss.Style
	Verified    lipgloss.Style
	MoldName    lipgloss.Style
	FoundryName lipgloss.Style
	Meta        lipgloss.Style
	FlashOK     lipgloss.Style
	FlashErr    lipgloss.Style
	FlashInfo   lipgloss.Style
	SevError    lipgloss.Style
	SevWarn     lipgloss.Style
	SevInfo     lipgloss.Style
}

// Theme returns the shared style set for tab sub-models.
func Theme() StyleSet {
	return StyleSet{
		Heading: lipgloss.NewStyle().
			Foreground(styles.Primary1).
			Bold(true),
		Cursor: lipgloss.NewStyle().
			Foreground(styles.Accent1).
			Bold(true),
		Verified: lipgloss.NewStyle().
			Foreground(styles.Success).
			Bold(true),
		MoldName: lipgloss.NewStyle().
			Foreground(styles.Accent1).
			Bold(true),
		FoundryName: lipgloss.NewStyle().
			Foreground(styles.Primary1).
			Bold(true),
		Meta: lipgloss.NewStyle().
			Foreground(styles.Gray),
		FlashOK: lipgloss.NewStyle().
			Foreground(styles.Success),
		FlashErr: lipgloss.NewStyle().
			Foreground(styles.Error).
			Bold(true),
		FlashInfo: lipgloss.NewStyle().
			Foreground(styles.Info),
		SevError: lipgloss.NewStyle().
			Foreground(styles.Error).
			Bold(true),
		SevWarn: lipgloss.NewStyle().
			Foreground(styles.Warning).
			Bold(true),
		SevInfo: lipgloss.NewStyle().
			Foreground(styles.Info),
	}
}
