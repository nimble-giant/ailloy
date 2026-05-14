package mold

import "testing"

func TestIsValidSPDX(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"empty", "", false},
		{"canonical Apache", "Apache-2.0", true},
		{"canonical MIT", "MIT", true},
		{"canonical BSD-3-Clause", "BSD-3-Clause", true},
		{"case-insensitive match", "apache-2.0", true},
		{"LicenseRef custom", "LicenseRef-Internal-1", true},
		{"LicenseRef bare prefix", "LicenseRef-", false},
		{"unknown", "Apache 2", false},
		{"made-up", "Megacorp-1.0", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsValidSPDX(tc.input); got != tc.want {
				t.Errorf("IsValidSPDX(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestSuggestSPDX(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"typo Apache 2", "Apache2.0", "Apache-2.0"},
		{"lowercase mit", "mit", "MIT"},
		{"BSD-3 typo", "BSD-3Clause", "BSD-3-Clause"},
		{"unrelated nonsense", "🦊", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := SuggestSPDX(tc.input); got != tc.want {
				t.Errorf("SuggestSPDX(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestCanonicalSPDX(t *testing.T) {
	if got := CanonicalSPDX("apache-2.0"); got != "Apache-2.0" {
		t.Errorf("CanonicalSPDX(apache-2.0) = %q, want Apache-2.0", got)
	}
	if got := CanonicalSPDX("LicenseRef-X"); got != "LicenseRef-X" {
		t.Errorf("CanonicalSPDX(LicenseRef-X) = %q, want LicenseRef-X", got)
	}
	if got := CanonicalSPDX("MadeUp"); got != "MadeUp" {
		t.Errorf("CanonicalSPDX(MadeUp) = %q, want MadeUp (unchanged)", got)
	}
}
