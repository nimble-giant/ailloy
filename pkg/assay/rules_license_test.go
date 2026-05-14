package assay

import (
	"os"
	"path/filepath"
	"testing"
)

func writeManifest(t *testing.T, dir, file, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, file), []byte(body), 0644); err != nil { //#nosec G306
		t.Fatal(err)
	}
}

func TestLicenseMissingRule_FiresOnMoldWithoutLicense(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, "mold.yaml", "apiVersion: v1\nkind: mold\nname: m\nversion: 1.0.0\n")

	diags := (&licenseMissingRule{}).Check(&RuleContext{RootDir: root, Config: DefaultConfig()})
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d: %+v", len(diags), diags)
	}
	if diags[0].File != "mold.yaml" {
		t.Errorf("expected File=mold.yaml, got %q", diags[0].File)
	}
	if diags[0].Rule != "license-missing" {
		t.Errorf("expected Rule=license-missing, got %q", diags[0].Rule)
	}
}

func TestLicenseMissingRule_SilentWhenDeclared(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, "mold.yaml", "apiVersion: v1\nkind: mold\nname: m\nversion: 1.0.0\nlicense: Apache-2.0\n")

	diags := (&licenseMissingRule{}).Check(&RuleContext{RootDir: root, Config: DefaultConfig()})
	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics when license is set, got: %+v", diags)
	}
}

func TestLicenseMissingRule_FiresPerIngotInMultiIngotLayout(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, filepath.Join(root, "ingots", "alpha"), "ingot.yaml",
		"apiVersion: v1\nkind: ingot\nname: alpha\nversion: 1.0.0\n")
	writeManifest(t, filepath.Join(root, "ingots", "beta"), "ingot.yaml",
		"apiVersion: v1\nkind: ingot\nname: beta\nversion: 1.0.0\nlicense: MIT\n")

	diags := (&licenseMissingRule{}).Check(&RuleContext{RootDir: root, Config: DefaultConfig()})
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic (only alpha), got %d: %+v", len(diags), diags)
	}
	if filepath.ToSlash(diags[0].File) != "ingots/alpha/ingot.yaml" {
		t.Errorf("expected File=ingots/alpha/ingot.yaml, got %q", diags[0].File)
	}
}

func TestLicenseMissingRule_NoManifestNoOp(t *testing.T) {
	root := t.TempDir()
	diags := (&licenseMissingRule{}).Check(&RuleContext{RootDir: root, Config: DefaultConfig()})
	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics for empty root, got: %+v", diags)
	}
}
