package index

import "strings"

// canonicalizeSource normalizes a foundry source URL so equivalent inputs map
// to the same key. Used as the visited-set key during transitive resolution.
func canonicalizeSource(source string) string {
	s := strings.TrimSpace(source)
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimSuffix(s, "/")
	s = strings.TrimSuffix(s, ".git")
	s = strings.ToLower(s)
	return s
}
