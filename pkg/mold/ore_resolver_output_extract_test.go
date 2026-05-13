package mold

import (
	"testing"
	"testing/fstest"
)

func TestLoadOreOverlaysFromFS_DoesNotLeakOutputIntoNamespacedDefaults(t *testing.T) {
	root := fstest.MapFS{
		"ores/agent_targets/ore.yaml":         &fstest.MapFile{Data: []byte("apiVersion: v1\nkind: ore\nname: agent_targets\nversion: 0.1.0\n")},
		"ores/agent_targets/flux.schema.yaml": &fstest.MapFile{Data: []byte("- name: enabled\n  type: bool\n  default: \"false\"\n")},
		"ores/agent_targets/flux.yaml":        &fstest.MapFile{Data: []byte("enabled: false\noutput:\n  blanks/AGENTS.md: AGENTS.md\n")},
		"ores/agent_targets/blanks/AGENTS.md": &fstest.MapFile{Data: []byte("hi")},
	}
	_, defaults, err := LoadOreOverlaysFromFS(root, "ores", nil)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	ore, _ := defaults["ore"].(map[string]any)
	if ore == nil {
		t.Fatalf("expected ore key in defaults: %+v", defaults)
	}
	at, _ := ore["agent_targets"].(map[string]any)
	if at == nil {
		t.Fatalf("expected agent_targets namespace: %+v", ore)
	}
	if _, hasOutput := at["output"]; hasOutput {
		t.Errorf("output key leaked into namespaced defaults: %+v", at)
	}
	if at["enabled"] != false {
		t.Errorf("non-output keys should remain: %+v", at)
	}
}
