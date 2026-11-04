package mold

import (
	"os"
	"testing"
)

// TestMain points HOME at a throwaway dir so tests never read or write the
// developer's real ~/.ailloy (per-project state now lives there).
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "ailloy-test-home-*")
	if err != nil {
		panic(err)
	}
	_ = os.Setenv("HOME", home)
	_ = os.Unsetenv("AILLOY_PROJECT_STATE_DIR")
	code := m.Run()
	_ = os.RemoveAll(home)
	os.Exit(code)
}
