package evolution

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"golang.org/x/term"
)

// Layout thresholds. Below `compactHeight`, the cinematic still plays but
// drops the giant fox and centers a smaller composition (title + version
// chip + flash + digit wheel + sparkles). Below `minHeight` it skips
// entirely.
const (
	minWidth      = 60
	minHeight     = 16
	compactHeight = 36
)

// Run plays the evolution cinematic in the alt-screen and returns when it
// completes or is skipped. Returns true if it played, false if it was
// suppressed. Always safe to call after a successful binary swap; the
// cinematic itself never writes anything permanent — the caller should
// print a receipt line to stdout regardless of the return value.
//
// When `AILLOY_DEBUG=1`, emits a one-line reason to stderr if the cinematic
// is suppressed, so silent skips don't go unnoticed in future regressions.
func Run(currentVersion, targetVersion string) bool {
	if !styles.ShouldAnimate() {
		debug("ShouldAnimate()=false")
		return false
	}
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		debug("term.GetSize: %v", err)
		return false
	}
	if w < minWidth || h < minHeight {
		debug("terminal %dx%d below minimum %dx%d", w, h, minWidth, minHeight)
		return false
	}
	model := New(w, h, currentVersion, targetVersion)
	prog := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := prog.Run(); err != nil {
		debug("tea.Run: %v", err)
		return false
	}
	return true
}

func debug(format string, args ...any) {
	if os.Getenv("AILLOY_DEBUG") == "" {
		return
	}
	fmt.Fprintf(os.Stderr, "[evolution] "+format+"\n", args...)
}
