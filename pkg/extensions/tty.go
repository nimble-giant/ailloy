package extensions

import (
	"os"

	"golang.org/x/term"
)

// isInteractive reports whether the calling shell can show prompts.
// We intentionally check both stdout AND stdin: piping on either side
// disables the consent prompt and the user must use --yes / explicit
// `ailloy ext install` instead.
func isInteractive() bool {
	if os.Getenv("AILLOY_NONINTERACTIVE") != "" {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd())) &&
		term.IsTerminal(int(os.Stdin.Fd()))
}
