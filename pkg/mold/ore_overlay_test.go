package mold

import "testing"

func TestOreReservedKeys(t *testing.T) {
	if OreOutputKey != "output" {
		t.Errorf("OreOutputKey = %q, want %q", OreOutputKey, "output")
	}
	if OreBlanksDir != "blanks" {
		t.Errorf("OreBlanksDir = %q, want %q", OreBlanksDir, "blanks")
	}
}
