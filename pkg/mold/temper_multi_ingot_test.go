package mold

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestTemper_MultiIngot_ValidatesEach(t *testing.T) {
	fsys := fstest.MapFS{
		"ingots/header/ingot.yaml": &fstest.MapFile{Data: []byte("apiVersion: v1\nkind: ingot\nname: header\nversion: 1.0.0\nfiles: [content.md]\n")},
		"ingots/header/content.md": &fstest.MapFile{Data: []byte("# header\n")},
		"ingots/footer/ingot.yaml": &fstest.MapFile{Data: []byte("apiVersion: v1\nkind: ingot\nname: footer\nversion: 1.0.0\nfiles: [missing.md]\n")},
	}
	result := Temper(fsys)

	if result.ManifestKind != "ingot" {
		t.Fatalf("expected ManifestKind=ingot, got %q", result.ManifestKind)
	}
	errs := result.Errors()
	if len(errs) == 0 {
		t.Fatalf("expected at least one error from footer's missing file, got %d (diags=%+v)", len(errs), result.Diagnostics)
	}
	found := false
	for _, e := range errs {
		if e.File == "ingots/footer/ingot.yaml" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error tied to footer's missing.md, got %+v", errs)
	}
}

func TestTemper_MultiIngot_AllValid(t *testing.T) {
	fsys := fstest.MapFS{
		"ingots/a/ingot.yaml": &fstest.MapFile{Data: []byte("apiVersion: v1\nkind: ingot\nname: a\nversion: 1.0.0\n")},
		"ingots/b/ingot.yaml": &fstest.MapFile{Data: []byte("apiVersion: v1\nkind: ingot\nname: b\nversion: 1.0.0\n")},
	}
	result := Temper(fsys)
	if result.HasErrors() {
		t.Fatalf("expected no errors, got %+v", result.Errors())
	}
	if result.ManifestKind != "ingot" {
		t.Fatalf("expected ManifestKind=ingot, got %q", result.ManifestKind)
	}
}

func TestTemper_MultiIngot_ParseError(t *testing.T) {
	fsys := fstest.MapFS{
		"ingots/broken/ingot.yaml": &fstest.MapFile{Data: []byte(":- not yaml")},
	}
	result := Temper(fsys)
	if !result.HasErrors() {
		t.Fatalf("expected errors for malformed multi-ingot manifest, got %+v", result.Diagnostics)
	}
	found := false
	for _, e := range result.Errors() {
		if strings.Contains(e.Message, "scanning ingots/") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected diagnostic mentioning 'scanning ingots/', got %+v", result.Errors())
	}
}
