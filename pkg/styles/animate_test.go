package styles

import (
	"os"
	"testing"

	"golang.org/x/term"
)

// stdoutIsTTY tells the test whether the process's stdout would normally be
// considered a TTY. When running under `go test` or in CI it almost always
// isn't, so the env-var test cases that *should* return true are conditionally
// skipped.
func stdoutIsTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func TestShouldAnimate_KillSwitches(t *testing.T) {
	t.Cleanup(func() { SetNoAnimate(false) })

	cases := []struct {
		name string
		set  func(t *testing.T)
	}{
		{"no-animate flag", func(t *testing.T) { SetNoAnimate(true) }},
		{"AILLOY_NO_ANIMATE", func(t *testing.T) { t.Setenv("AILLOY_NO_ANIMATE", "1") }},
		{"NO_COLOR", func(t *testing.T) { t.Setenv("NO_COLOR", "1") }},
		{"CI", func(t *testing.T) { t.Setenv("CI", "true") }},
		{"TERM=dumb", func(t *testing.T) { t.Setenv("TERM", "dumb") }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			SetNoAnimate(false)
			// Clear other kill switches so each subtest only exercises one.
			t.Setenv("AILLOY_NO_ANIMATE", "")
			t.Setenv("NO_COLOR", "")
			t.Setenv("CI", "")
			t.Setenv("TERM", "xterm-256color")

			tc.set(t)

			if ShouldAnimate() {
				t.Fatalf("expected ShouldAnimate=false with %s set", tc.name)
			}
		})
	}
}

func TestShouldAnimate_RequiresTTY(t *testing.T) {
	t.Cleanup(func() { SetNoAnimate(false) })

	SetNoAnimate(false)
	t.Setenv("AILLOY_NO_ANIMATE", "")
	t.Setenv("NO_COLOR", "")
	t.Setenv("CI", "")
	t.Setenv("TERM", "xterm-256color")

	got := ShouldAnimate()
	if stdoutIsTTY() {
		if !got {
			t.Fatal("expected ShouldAnimate=true when stdout is a TTY and no kill switch is set")
		}
	} else if got {
		t.Fatal("expected ShouldAnimate=false when stdout is not a TTY")
	}
}

func TestPause_NoOpWhenSuppressed(t *testing.T) {
	t.Cleanup(func() { SetNoAnimate(false) })

	SetNoAnimate(true)
	// 5 seconds would be obvious if Pause actually slept; the test runner's
	// default timeout is much shorter than that.
	Pause(5_000_000_000)
}
