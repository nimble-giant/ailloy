package commands

import (
	"strings"
	"testing"
)

func TestUpdateCmd_Registered(t *testing.T) {
	// Verify the update command is registered on the root command.
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "update" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("update command not registered on rootCmd")
	}
}

func TestUpdateCmd_Aliases(t *testing.T) {
	for _, c := range rootCmd.Commands() {
		if c.Name() != "update" {
			continue
		}
		want := map[string]bool{"self-update": true, "upgrade": true}
		for _, a := range c.Aliases {
			delete(want, a)
		}
		if len(want) > 0 {
			missing := make([]string, 0, len(want))
			for k := range want {
				missing = append(missing, k)
			}
			t.Errorf("missing aliases: %v", missing)
		}
		return
	}
	t.Fatal("update command not found")
}

func TestUpdateCmd_CheckFlag(t *testing.T) {
	for _, c := range rootCmd.Commands() {
		if c.Name() != "update" {
			continue
		}
		f := c.Flags().Lookup("check")
		if f == nil {
			t.Fatal("--check flag not registered")
		}
		if f.DefValue != "false" {
			t.Errorf("--check default = %q, want %q", f.DefValue, "false")
		}
		return
	}
	t.Fatal("update command not found")
}

func TestUpdateCmd_HelpContainsUsage(t *testing.T) {
	for _, c := range rootCmd.Commands() {
		if c.Name() != "update" {
			continue
		}
		if !strings.Contains(c.Long, "--check") {
			t.Error("Long description should mention --check flag")
		}
		if c.Short == "" {
			t.Error("Short description should not be empty")
		}
		return
	}
	t.Fatal("update command not found")
}

func TestRawVersion_Default(t *testing.T) {
	// Before SetVersionInfo is called the raw version may be empty or
	// whatever was set previously. Just verify RawVersion does not panic.
	_ = RawVersion()
}

func TestSetVersionInfo_SetsRawVersion(t *testing.T) {
	orig := rawVersion
	defer func() { rawVersion = orig }()

	SetVersionInfo("v9.8.7", "abc1234", "2026-01-01")
	if got := RawVersion(); got != "v9.8.7" {
		t.Errorf("RawVersion() = %q, want %q", got, "v9.8.7")
	}
}

func TestSetVersionInfo_UnknownBuildInfo(t *testing.T) {
	orig := rawVersion
	origVer := rootCmd.Version
	defer func() {
		rawVersion = orig
		rootCmd.Version = origVer
	}()

	SetVersionInfo("v1.0.0", "unknown", "unknown")
	if rootCmd.Version != "v1.0.0" {
		t.Errorf("rootCmd.Version = %q, want %q", rootCmd.Version, "v1.0.0")
	}
}
