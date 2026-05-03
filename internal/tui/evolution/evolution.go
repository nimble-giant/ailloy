package evolution

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"golang.org/x/term"
)

const (
	minWidth  = 80
	minHeight = 36
)

// Run plays the evolution cinematic in the alt-screen and returns when it
// completes or is skipped. Returns true if it played, false if it was
// suppressed (non-TTY, kill switch set, terminal too small, or program
// errored). Always safe to call after a successful binary swap; the
// cinematic itself never writes anything permanent — the caller should
// print a receipt line to stdout regardless of the return value.
func Run(currentVersion, targetVersion string) bool {
	if !styles.ShouldAnimate() {
		return false
	}
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w < minWidth || h < minHeight {
		return false
	}
	prog := tea.NewProgram(New(w, h, currentVersion, targetVersion), tea.WithAltScreen())
	if _, err := prog.Run(); err != nil {
		return false
	}
	return true
}
