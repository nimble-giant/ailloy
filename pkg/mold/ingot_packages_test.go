package mold

import (
	"testing"
	"testing/fstest"
)

func TestDiscoverIngotPackages_RootLayout(t *testing.T) {
	fsys := fstest.MapFS{
		"ingot.yaml": &fstest.MapFile{Data: []byte("apiVersion: v1\nkind: ingot\nname: solo\nversion: 0.1.0\n")},
	}
	pkgs, err := DiscoverIngotPackages(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}
	if pkgs[0].Name != "solo" {
		t.Errorf("expected name %q, got %q", "solo", pkgs[0].Name)
	}
	if pkgs[0].Subpath != "" {
		t.Errorf("expected empty subpath for root layout, got %q", pkgs[0].Subpath)
	}
	if pkgs[0].Root != "." {
		t.Errorf("expected root %q, got %q", ".", pkgs[0].Root)
	}
}

func TestDiscoverIngotPackages_MultiLayout(t *testing.T) {
	fsys := fstest.MapFS{
		"ingots/header/ingot.yaml":   &fstest.MapFile{Data: []byte("apiVersion: v1\nkind: ingot\nname: header\nversion: 1.0.0\n")},
		"ingots/footer/ingot.yaml":   &fstest.MapFile{Data: []byte("apiVersion: v1\nkind: ingot\nname: footer\nversion: 1.0.0\n")},
		"ingots/empty-dir/README.md": &fstest.MapFile{Data: []byte("not an ingot")},
	}
	pkgs, err := DiscoverIngotPackages(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}
	if pkgs[0].Name != "footer" || pkgs[0].Subpath != "ingots/footer" {
		t.Errorf("pkgs[0]: got name=%q subpath=%q, want name=%q subpath=%q",
			pkgs[0].Name, pkgs[0].Subpath, "footer", "ingots/footer")
	}
	if pkgs[1].Name != "header" || pkgs[1].Subpath != "ingots/header" {
		t.Errorf("pkgs[1]: got name=%q subpath=%q, want name=%q subpath=%q",
			pkgs[1].Name, pkgs[1].Subpath, "header", "ingots/header")
	}
}

func TestDiscoverIngotPackages_NoIngot(t *testing.T) {
	fsys := fstest.MapFS{"README.md": &fstest.MapFile{Data: []byte("not an ingot repo")}}
	pkgs, err := DiscoverIngotPackages(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 0 {
		t.Fatalf("expected 0 packages, got %d", len(pkgs))
	}
}

func TestDiscoverIngotPackages_RootShadowsMulti(t *testing.T) {
	fsys := fstest.MapFS{
		"ingot.yaml":               &fstest.MapFile{Data: []byte("apiVersion: v1\nkind: ingot\nname: root\nversion: 0.1.0\n")},
		"ingots/header/ingot.yaml": &fstest.MapFile{Data: []byte("apiVersion: v1\nkind: ingot\nname: header\nversion: 1.0.0\n")},
	}
	pkgs, err := DiscoverIngotPackages(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package (root shadows multi), got %d", len(pkgs))
	}
	if pkgs[0].Name != "root" {
		t.Errorf("expected root layout to win, got %q", pkgs[0].Name)
	}
}

func TestDiscoverIngotPackages_BadManifest(t *testing.T) {
	fsys := fstest.MapFS{
		"ingots/broken/ingot.yaml": &fstest.MapFile{Data: []byte(":- not yaml")},
	}
	_, err := DiscoverIngotPackages(fsys)
	if err == nil {
		t.Fatal("expected error for malformed multi-ingot manifest, got nil")
	}
}
