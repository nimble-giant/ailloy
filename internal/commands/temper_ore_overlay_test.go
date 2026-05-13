package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

func TestTemperOre_AcceptsOutputAndBlanks(t *testing.T) {
	dir := writeOverlayOreFixture(t, map[string]string{
		"ore.yaml":         "apiVersion: v1\nkind: ore\nname: at\nversion: 0.1.0\n",
		"flux.schema.yaml": "- name: enabled\n  type: bool\n  default: \"false\"\n",
		"flux.yaml":        "enabled: false\noutput:\n  blanks/AGENTS.md: AGENTS.md\n",
		"blanks/AGENTS.md": "hi\n",
	})
	res := mold.Temper(os.DirFS(dir))
	if res.HasErrors() {
		t.Fatalf("unexpected errors: %+v", res.Errors())
	}
}

func TestTemperOre_RejectsFromInOreOutput(t *testing.T) {
	dir := writeOverlayOreFixture(t, map[string]string{
		"ore.yaml":         "apiVersion: v1\nkind: ore\nname: at\nversion: 0.1.0\n",
		"flux.schema.yaml": "- name: enabled\n  type: bool\n  default: \"false\"\n",
		"flux.yaml":        "enabled: false\noutput:\n  AGENTS.md:\n    from: ore/other/blanks/x\n    dest: AGENTS.md\n",
		"AGENTS.md":        "hi\n",
	})
	res := mold.Temper(os.DirFS(dir))
	if !res.HasErrors() {
		t.Fatal("expected error for from: in ore output")
	}
	if !overlayContainsError(res.Errors(), "from:") {
		t.Errorf("expected error mentioning from:, got %+v", res.Errors())
	}
}

func TestTemperOre_RejectsOutputReferencingMissingBlank(t *testing.T) {
	dir := writeOverlayOreFixture(t, map[string]string{
		"ore.yaml":         "apiVersion: v1\nkind: ore\nname: at\nversion: 0.1.0\n",
		"flux.schema.yaml": "- name: enabled\n  type: bool\n  default: \"false\"\n",
		"flux.yaml":        "enabled: false\noutput:\n  blanks/missing.md: AGENTS.md\n",
	})
	res := mold.Temper(os.DirFS(dir))
	if !res.HasErrors() {
		t.Fatal("expected error for missing blank reference")
	}
	if !overlayContainsError(res.Errors(), "does not exist") {
		t.Errorf("expected missing-path error, got %+v", res.Errors())
	}
}

func TestTemperOre_RejectsNonMapOutput(t *testing.T) {
	dir := writeOverlayOreFixture(t, map[string]string{
		"ore.yaml":         "apiVersion: v1\nkind: ore\nname: at\nversion: 0.1.0\n",
		"flux.schema.yaml": "- name: enabled\n  type: bool\n  default: \"false\"\n",
		"flux.yaml":        "enabled: false\noutput: notamap\n",
	})
	res := mold.Temper(os.DirFS(dir))
	if !res.HasErrors() {
		t.Fatal("expected error for scalar output")
	}
	if !overlayContainsError(res.Errors(), "must be a map") {
		t.Errorf("expected map-required error, got %+v", res.Errors())
	}
}

func TestTemperOre_BackwardCompat_NoOutputNoBlanks(t *testing.T) {
	dir := writeOverlayOreFixture(t, map[string]string{
		"ore.yaml":         "apiVersion: v1\nkind: ore\nname: at\nversion: 0.1.0\n",
		"flux.schema.yaml": "- name: enabled\n  type: bool\n  default: \"false\"\n",
		"flux.yaml":        "enabled: false\n",
	})
	res := mold.Temper(os.DirFS(dir))
	if res.HasErrors() {
		t.Fatalf("unexpected errors on backward-compat ore: %+v", res.Errors())
	}
}

// --- helpers ---

func writeOverlayOreFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, body := range files {
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0750); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(body), 0644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	return dir
}

func overlayContainsError(errs []mold.Diagnostic, substr string) bool {
	for _, e := range errs {
		if strings.Contains(e.Message, substr) {
			return true
		}
	}
	return false
}
