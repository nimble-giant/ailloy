package commands

import (
	"testing"
)

func TestGetBinaryVersion_Git(t *testing.T) {
	// This test calls the real binary; skip if not available
	found, _ := checkBinary("git")
	if !found {
		t.Skip("git not found on PATH")
	}

	version := getBinaryVersion("git")
	if version == "" {
		t.Error("expected non-empty version string for git")
	}
	if len(version) < 2 || version[0] != 'v' {
		t.Errorf("expected version to start with 'v', got %q", version)
	}
}

func TestGetBinaryVersion_NonExistent(t *testing.T) {
	version := getBinaryVersion("nonexistent-binary-xyz-123")
	if version != "" {
		t.Errorf("expected empty version for non-existent binary, got %q", version)
	}
}

func TestCheckBinary_Exists(t *testing.T) {
	// 'sh' should exist on all UNIX systems
	found, _ := checkBinary("sh")
	if !found {
		t.Error("expected 'sh' to be found on PATH")
	}
}

func TestCheckBinary_NotExists(t *testing.T) {
	found, _ := checkBinary("nonexistent-binary-xyz-123")
	if found {
		t.Error("expected non-existent binary to not be found")
	}
}

func TestRuntimeDeps_Defined(t *testing.T) {
	if len(runtimeDeps) == 0 {
		t.Fatal("expected at least one runtime dependency defined")
	}

	expectedBinaries := map[string]bool{
		"git":    false,
		"gh":     false,
		"claude": false,
	}

	for _, dep := range runtimeDeps {
		if dep.name == "" {
			t.Error("dependency has empty name")
		}
		if dep.binary == "" {
			t.Error("dependency has empty binary")
		}
		if dep.description == "" {
			t.Errorf("dependency %s has empty description", dep.name)
		}
		if len(dep.installHelp) == 0 {
			t.Errorf("dependency %s has no install help", dep.name)
		}

		if _, ok := expectedBinaries[dep.binary]; ok {
			expectedBinaries[dep.binary] = true
		}
	}

	for binary, found := range expectedBinaries {
		if !found {
			t.Errorf("expected runtime dependency for binary %s", binary)
		}
	}
}

func TestRuntimeDeps_PlatformCoverage(t *testing.T) {
	platforms := []string{"darwin", "linux", "windows"}

	for _, dep := range runtimeDeps {
		for _, platform := range platforms {
			if _, ok := dep.installHelp[platform]; !ok {
				t.Errorf("dependency %s missing install help for platform %s", dep.name, platform)
			}
		}
	}
}
