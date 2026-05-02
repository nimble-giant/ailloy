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

// IndexLookup fetches a parsed Index for a given source URL. Implementations
// may be network-backed (Fetcher) or cache-backed (LoadCachedIndex).
type IndexLookup func(source string) (*Index, error)

// ResolvedFoundry is one node in the resolved foundry tree.
type ResolvedFoundry struct {
	Index    *Index
	Source   string   // canonical source URL
	Parents  []string // chain of foundry Names from root to this node's parent (empty for root)
	Children []*ResolvedFoundry
}

// ResolvedMold is a mold paired with the foundry that owns it.
type ResolvedMold struct {
	Entry   MoldEntry
	Foundry *ResolvedFoundry
}

// ResolutionWarning captures a non-fatal problem encountered during Resolve
// (e.g., a child foundry that failed to fetch).
type ResolutionWarning struct {
	Source string
	Err    error
}

// Resolver walks a foundry tree depth-first via an IndexLookup, deduplicating
// shared subtrees and silently breaking cycles via a visited-set keyed by
// canonical source URL.
type Resolver struct {
	lookup   IndexLookup
	visited  map[string]*ResolvedFoundry
	warnings []ResolutionWarning
}

// NewResolver constructs a Resolver around the given lookup.
func NewResolver(lookup IndexLookup) *Resolver {
	return &Resolver{
		lookup:  lookup,
		visited: make(map[string]*ResolvedFoundry),
	}
}

// Warnings returns any non-fatal issues collected during the most recent Resolve.
func (r *Resolver) Warnings() []ResolutionWarning {
	return r.warnings
}

// Resolve fetches the root foundry at rootSource and recursively resolves all
// child foundries. Returns the root node and a flat depth-first list of every
// reachable mold (each with a back-pointer to its owning foundry).
func (r *Resolver) Resolve(rootSource string) (*ResolvedFoundry, []ResolvedMold, error) {
	root, err := r.resolveNode(rootSource, nil)
	if err != nil {
		return nil, nil, err
	}
	molds := flattenMolds(root, make(map[string]bool))
	return root, molds, nil
}

func (r *Resolver) resolveNode(source string, parents []string) (*ResolvedFoundry, error) {
	key := canonicalizeSource(source)
	if existing, ok := r.visited[key]; ok {
		return existing, nil
	}
	idx, err := r.lookup(source)
	if err != nil {
		return nil, err
	}
	node := &ResolvedFoundry{
		Index:   idx,
		Source:  key,
		Parents: append([]string(nil), parents...),
	}
	r.visited[key] = node
	return node, nil
}

func flattenMolds(node *ResolvedFoundry, seen map[string]bool) []ResolvedMold {
	if seen[node.Source] {
		return nil
	}
	seen[node.Source] = true
	out := make([]ResolvedMold, 0, len(node.Index.Molds))
	for _, m := range node.Index.Molds {
		out = append(out, ResolvedMold{Entry: m, Foundry: node})
	}
	for _, child := range node.Children {
		out = append(out, flattenMolds(child, seen)...)
	}
	return out
}
