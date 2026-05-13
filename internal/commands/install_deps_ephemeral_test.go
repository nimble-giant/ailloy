package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

func TestEphemeralOreResolver_ExposesOreOutputAndFS(t *testing.T) {
	tmp := t.TempDir()
	oreDir := filepath.Join(tmp, "ore")
	if err := os.MkdirAll(filepath.Join(oreDir, "blanks"), 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	files := map[string]string{
		filepath.Join(oreDir, "ore.yaml"):         "apiVersion: v1\nkind: ore\nname: at\nversion: 0.1.0\n",
		filepath.Join(oreDir, "flux.schema.yaml"): "- name: enabled\n  type: bool\n  default: \"false\"\n",
		filepath.Join(oreDir, "flux.yaml"):        "enabled: false\noutput:\n  blanks/AGENTS.md: AGENTS.md\n",
		filepath.Join(oreDir, "blanks/AGENTS.md"): "# at agents\n",
	}
	for p, body := range files {
		if err := os.WriteFile(p, []byte(body), 0644); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
	}

	moldDir := filepath.Join(tmp, "consumer")
	if err := os.MkdirAll(moldDir, 0750); err != nil {
		t.Fatalf("mkdir mold: %v", err)
	}
	moldYAML := "apiVersion: v1\nkind: Mold\nname: c\nversion: 0.1.0\ndependencies:\n  - ore: " + oreDir + "\n    version: 0.1.0\n"
	if err := os.WriteFile(filepath.Join(moldDir, "mold.yaml"), []byte(moldYAML), 0644); err != nil {
		t.Fatalf("write mold.yaml: %v", err)
	}

	manifest, err := mold.LoadMold(filepath.Join(moldDir, "mold.yaml"))
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	r, err := ResolveDepsEphemeral(manifest, true)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	sources := r.OreSources()
	if len(sources) != 1 {
		t.Fatalf("expected 1 ore source, got %d", len(sources))
	}
	s := sources[0]
	if s.Namespace != "at" {
		t.Errorf("namespace = %q, want %q", s.Namespace, "at")
	}
	if s.FS == nil {
		t.Fatal("FS is nil")
	}
	if _, ok := s.Output["blanks/AGENTS.md"]; !ok {
		t.Errorf("Output missing blanks/AGENTS.md: %+v", s.Output)
	}
}

func TestEphemeralOreResolver_BackwardCompat_NoOutputBlock(t *testing.T) {
	tmp := t.TempDir()
	oreDir := filepath.Join(tmp, "ore")
	if err := os.MkdirAll(oreDir, 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	files := map[string]string{
		filepath.Join(oreDir, "ore.yaml"):         "apiVersion: v1\nkind: ore\nname: legacy\nversion: 0.1.0\n",
		filepath.Join(oreDir, "flux.schema.yaml"): "- name: enabled\n  type: bool\n  default: \"false\"\n",
		filepath.Join(oreDir, "flux.yaml"):        "enabled: false\n",
	}
	for p, body := range files {
		if err := os.WriteFile(p, []byte(body), 0644); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
	}

	moldDir := filepath.Join(tmp, "consumer")
	if err := os.MkdirAll(moldDir, 0750); err != nil {
		t.Fatalf("mkdir mold: %v", err)
	}
	moldYAML := "apiVersion: v1\nkind: Mold\nname: c\nversion: 0.1.0\ndependencies:\n  - ore: " + oreDir + "\n    version: 0.1.0\n"
	if err := os.WriteFile(filepath.Join(moldDir, "mold.yaml"), []byte(moldYAML), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	manifest, err := mold.LoadMold(filepath.Join(moldDir, "mold.yaml"))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	r, err := ResolveDepsEphemeral(manifest, true)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	sources := r.OreSources()
	if len(sources) != 1 {
		t.Fatalf("expected 1 ore source, got %d", len(sources))
	}
	if sources[0].Output != nil {
		t.Errorf("legacy ore should have nil Output, got %+v", sources[0].Output)
	}
	// Schema overlay should still work — Defaults should not contain
	// 'output' under ore.legacy.
	defaults := r.Defaults()
	if ore, ok := defaults["ore"].(map[string]any); ok {
		if leg, ok := ore["legacy"].(map[string]any); ok {
			if _, hasOutput := leg["output"]; hasOutput {
				t.Errorf("output key leaked into namespaced defaults: %+v", leg)
			}
		}
	}
}
