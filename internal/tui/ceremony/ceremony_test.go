package ceremony

import (
	"strings"
	"testing"
)

// removeEscapes strips ANSI SGR sequences so we can assert visible content.
func removeEscapes(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func TestComposeStamp_PlainContent(t *testing.T) {
	got := removeEscapes(composeStamp("🔥", "TEMPERED", "0 errors, 2 warnings", Temper.Primary, Temper.Accent, false))
	// Must contain glyph, word, separator, and the summary verbatim.
	for _, want := range []string{"🔥", "TEMPERED", "—", "0 errors, 2 warnings"} {
		if !strings.Contains(got, want) {
			t.Errorf("stamp missing %q in %q", want, got)
		}
	}
}

func TestComposeStamp_NoSummary(t *testing.T) {
	got := removeEscapes(composeStamp("⚗️", "ASSAY COMPLETE", "", Assay.Primary, Assay.Accent, false))
	if strings.Contains(got, "—") {
		t.Errorf("empty summary should drop the separator: %q", got)
	}
	if !strings.Contains(got, "ASSAY COMPLETE") {
		t.Errorf("stamp missing word: %q", got)
	}
}

func TestComposeFailStamp_HasFailedSuffix(t *testing.T) {
	got := removeEscapes(composeFailStamp("🔥", "TEMPER", "3 errors, 0 warnings"))
	if !strings.Contains(got, "TEMPER FAILED") {
		t.Errorf("fail stamp missing 'FAILED': %q", got)
	}
	if !strings.Contains(got, "3 errors, 0 warnings") {
		t.Errorf("fail stamp dropped summary: %q", got)
	}
}

func TestThemes_Populated(t *testing.T) {
	// Sanity: every theme has the required fields. Cheap regression for
	// future copy-pasted entries.
	cases := []struct {
		name string
		th   Theme
	}{
		{"assay", Assay},
		{"temper", Temper},
		{"cast", Cast},
		{"recast", Recast},
		{"quench", Quench},
		{"smelt", Smelt},
		{"forge", Forge},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.th.Name == "" {
				t.Error("Name is empty")
			}
			if c.th.Verb == "" {
				t.Error("Verb is empty")
			}
			if c.th.StampWord == "" {
				t.Error("StampWord is empty")
			}
			if c.th.Glyph == "" {
				t.Error("Glyph is empty")
			}
			if c.th.StrikeChar == "" {
				t.Error("StrikeChar is empty")
			}
			if c.th.Strikes < 1 {
				t.Errorf("Strikes must be >=1, got %d", c.th.Strikes)
			}
		})
	}
}
