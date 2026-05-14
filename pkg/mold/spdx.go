package mold

import (
	"sort"
	"strings"
)

// curatedSPDXIDs is the embedded set of common SPDX license identifiers
// recognized by ailloy. Authors needing an ID outside this list can use the
// SPDX `LicenseRef-<id>` form for custom/proprietary licenses without
// triggering a warning. Expand this list as community needs grow — the SPDX
// canonical list at https://spdx.org/licenses/ is the source of truth.
var curatedSPDXIDs = []string{
	"0BSD",
	"AGPL-3.0-only",
	"AGPL-3.0-or-later",
	"Apache-2.0",
	"Artistic-2.0",
	"BSD-2-Clause",
	"BSD-3-Clause",
	"BSD-3-Clause-Clear",
	"BSL-1.0",
	"CC-BY-4.0",
	"CC-BY-SA-4.0",
	"CC0-1.0",
	"EPL-2.0",
	"GPL-2.0-only",
	"GPL-2.0-or-later",
	"GPL-3.0-only",
	"GPL-3.0-or-later",
	"ISC",
	"LGPL-2.1-only",
	"LGPL-2.1-or-later",
	"LGPL-3.0-only",
	"LGPL-3.0-or-later",
	"MIT",
	"MPL-2.0",
	"Unlicense",
	"WTFPL",
	"Zlib",
}

// spdxIDSet is a lookup of curated SPDX IDs by lowercased form for
// case-insensitive matching. SPDX IDs are case-sensitive in spec, but author
// typos are common — we lower-compare for IsValidSPDX, and the suggester
// returns the canonical casing.
var spdxIDSet = func() map[string]string {
	m := make(map[string]string, len(curatedSPDXIDs))
	for _, id := range curatedSPDXIDs {
		m[strings.ToLower(id)] = id
	}
	return m
}()

// IsValidSPDX reports whether s is a recognized SPDX identifier from the
// curated list, or an SPDX `LicenseRef-<id>` reference. Empty input returns
// false; callers handle the "license omitted" case separately.
func IsValidSPDX(s string) bool {
	if s == "" {
		return false
	}
	if strings.HasPrefix(s, "LicenseRef-") {
		// Per SPDX spec, LicenseRef- + idstring where idstring is
		// [a-zA-Z0-9.-]+. Any non-empty suffix is accepted here; deeper
		// validation isn't worth the spec-tracking burden.
		return len(s) > len("LicenseRef-")
	}
	_, ok := spdxIDSet[strings.ToLower(s)]
	return ok
}

// CanonicalSPDX returns the canonically-cased SPDX ID for s if s is in the
// curated list (case-insensitive match). For LicenseRef- inputs and unknown
// IDs, it returns s unchanged.
func CanonicalSPDX(s string) string {
	if strings.HasPrefix(s, "LicenseRef-") {
		return s
	}
	if canonical, ok := spdxIDSet[strings.ToLower(s)]; ok {
		return canonical
	}
	return s
}

// SuggestSPDX returns the closest curated SPDX ID to s by edit distance,
// for inclusion in temper warnings. Returns "" when no candidate is within
// a reasonable distance, to avoid suggesting nonsense for wholly unrelated
// input — the cap is min(4, len(s)-1), so short or exotic inputs (e.g. a
// stray emoji) get no suggestion rather than a misleading one.
func SuggestSPDX(s string) string {
	if s == "" {
		return ""
	}
	target := strings.ToLower(s)
	maxDist := min(4, len(target)-1)
	if maxDist < 1 {
		return ""
	}
	bestID := ""
	bestDist := maxDist + 1
	for _, id := range curatedSPDXIDs {
		d := levenshtein(target, strings.ToLower(id))
		if d < bestDist {
			bestDist = d
			bestID = id
		}
	}
	return bestID
}

// CuratedSPDXIDs returns a sorted copy of the curated SPDX identifier list.
// Useful for help text and `ailloy license` tooling.
func CuratedSPDXIDs() []string {
	out := make([]string, len(curatedSPDXIDs))
	copy(out, curatedSPDXIDs)
	sort.Strings(out)
	return out
}

// levenshtein computes edit distance between two strings using the classic
// row-by-row DP. Small inputs (SPDX IDs are <30 chars) — no need for a
// trickier algorithm.
func levenshtein(a, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(curr[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}
