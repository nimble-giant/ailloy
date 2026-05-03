package data

import "strings"

// MoldIdentity returns the canonical join key used to match a Discover catalog
// entry against an installed-manifest entry.
//
// Catalog rows store the mold's full source (which may embed a //subpath, e.g.
// "github.com/owner/repo//molds/launch"), while installed rows store the bare
// repo source plus a separate Subpath field. Without normalisation the two
// strings never match for any subpath-bearing mold and the "installed" badge
// silently drops off Discover after a TUI restart.
//
// Pass either form: an embedded "//subpath" in source is preferred, otherwise
// the explicit subpath argument is appended.
func MoldIdentity(source, subpath string) string {
	base := source
	if i := strings.Index(source, "//"); i != -1 {
		if subpath == "" {
			subpath = source[i+2:]
		}
		base = source[:i]
	}
	if subpath == "" {
		return base
	}
	return base + "//" + subpath
}
