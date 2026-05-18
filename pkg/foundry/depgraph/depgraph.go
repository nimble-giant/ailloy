// Package depgraph builds and resolves a transitive mold dependency graph.
//
// A mold's manifest may declare other molds it depends on (via
// `dependencies: [{ mold: ..., version: ... }]`). When `ailloy cast` runs on
// a root mold, the full graph of mold-on-mold dependencies must be resolved
// and installed alongside the root. This package owns that resolution.
//
// Resolution policy: highest-compatible semver (npm/cargo style). For each
// unique mold reached via the graph, all version constraints are intersected;
// the highest version satisfying every constraint wins. Conflicts (no version
// satisfies all constraints) and cycles are surfaced as actionable errors.
//
// Non-semver pins (branch names, exact SHAs) are matched by equality across
// all encountered references — any disagreement is an error.
package depgraph

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/mold"
)

// NodeKey is the canonical identity of a mold in the graph: its source URL
// plus optional subpath. Aliases (`as:` on the dep declaration) do NOT change
// the key — the same underlying mold reached via two aliases is one node.
type NodeKey struct {
	Source  string // host/owner/repo
	Subpath string
}

// String returns "source//subpath" or just "source" when no subpath.
func (k NodeKey) String() string {
	if k.Subpath == "" {
		return k.Source
	}
	return k.Source + "//" + k.Subpath
}

// ParentEdge represents one parent → child edge with the per-edge data
// (alias, with-values) supplied by the parent's dep declaration.
type ParentEdge struct {
	Source  string         // parent's NodeKey.Source
	Subpath string         // parent's NodeKey.Subpath
	As      string         // alias the parent gave the child
	With    map[string]any // Helm-style sub-flux values from the parent
}

// Node is a resolved mold in the graph: its mold.yaml plus the resolved
// version + commit, plus the edges that pulled it in.
type Node struct {
	Key     NodeKey
	Mold    *mold.Mold
	Ref     *foundry.Reference // resolved with concrete version
	Version string             // tag (or branch/sha when applicable)
	Commit  string
	Parents []ParentEdge   // empty = direct/root
	With    map[string]any // merged with-values from all incoming edges
}

// Graph is the resolved dep graph in topological (leaves-first) order.
type Graph struct {
	Nodes []Node
}

// Find returns the node with the given (source, subpath), or nil.
func (g *Graph) Find(source, subpath string) *Node {
	for i := range g.Nodes {
		if g.Nodes[i].Key.Source == source && g.Nodes[i].Key.Subpath == subpath {
			return &g.Nodes[i]
		}
	}
	return nil
}

// FetchResult is what a Fetcher returns for one (source, subpath, version).
type FetchResult struct {
	Mold    *mold.Mold
	Version string // resolved tag (e.g. "v1.2.3"); empty for branch/sha
	Commit  string // commit SHA
}

// TagInfo is one entry from a Fetcher.Tags listing: where the tag points and
// the version the mold declared at that tag.
type TagInfo struct {
	SHA         string // commit SHA the tag points at
	MoldVersion string // version declared in the mold's mold.yaml at the tag
}

// Fetcher abstracts the I/O the builder needs: resolving + loading a mold
// at a given reference, and listing the available semver tags for a source.
// Tests inject a fake; production wires this to foundry.ResolveWithMetadata
// + remote tag listing via git ls-remote.
type Fetcher interface {
	// Fetch resolves the reference (which may be a constraint, exact, branch,
	// or SHA) to a concrete version and returns the parsed mold.yaml.
	Fetch(ref *foundry.Reference) (FetchResult, error)
	// Tags returns the available semver tags for the source. Used by the
	// constraint solver to pick the highest version satisfying every
	// accumulated constraint when more than one was encountered. Each tag
	// carries its mold.yaml version so constraints resolve correctly on
	// release-train monorepos.
	Tags(source, subpath string) (map[string]TagInfo, error)
}

// Builder builds dep graphs.
type Builder struct {
	Fetcher Fetcher
}

// New constructs a Builder.
func New(f Fetcher) *Builder {
	return &Builder{Fetcher: f}
}

// Build walks the dep graph rooted at the given mold and returns it in
// leaves-first topological order.
//
// The algorithm:
//
//  1. DFS from root. For each visited (source, subpath) key, accumulate the
//     version constraints that referenced it and remember the dep edges.
//     Detect cycles via the visiting stack.
//  2. For each unique key, intersect its accumulated constraints (semver) and
//     pick the highest available tag satisfying all. For non-semver pins
//     (branch / SHA), require equality across all references.
//  3. If the chosen version differs from what was used during the discovery
//     fetch, re-fetch the node so its declared deps are accurate. We allow
//     up to maxResolveIterations passes — typical graphs converge in 1.
func (b *Builder) Build(root *mold.Mold, rootRef *foundry.Reference) (*Graph, error) {
	if b.Fetcher == nil {
		return nil, fmt.Errorf("depgraph: nil Fetcher")
	}
	if root == nil {
		return nil, fmt.Errorf("depgraph: nil root mold")
	}
	if rootRef == nil {
		return nil, fmt.Errorf("depgraph: nil root reference")
	}

	state := &buildState{
		fetcher:       b.Fetcher,
		nodes:         map[NodeKey]*Node{},
		constraints:   map[NodeKey][]constraintRef{},
		order:         nil,
		visiting:      map[NodeKey]bool{},
		visitingOrder: nil,
		nonSemverPins: map[NodeKey][]nonSemverRef{},
	}

	rootKey := NodeKey{Source: rootRef.CacheKey(), Subpath: rootRef.Subpath}
	rootNode := &Node{
		Key:     rootKey,
		Mold:    root,
		Ref:     rootRef,
		Version: rootRef.Version,
		Commit:  "",
	}
	state.nodes[rootKey] = rootNode
	state.order = append(state.order, rootKey)

	if err := state.walkChildren(rootKey, root); err != nil {
		return nil, err
	}

	// Resolve each non-root node to its highest-compatible version.
	if err := state.resolveAll(rootKey); err != nil {
		return nil, err
	}

	// Topo-sort: leaves first. We built `order` in DFS-pre order; reverse for
	// leaves-first because root was pushed first and leaves last.
	graph := &Graph{Nodes: make([]Node, 0, len(state.order))}
	// Build a children map to do a proper post-order traversal.
	children := map[NodeKey][]NodeKey{}
	for _, k := range state.order {
		n := state.nodes[k]
		seen := map[NodeKey]bool{}
		for _, dep := range n.Mold.Dependencies {
			if kind, _ := dep.Kind(); kind != "mold" {
				continue
			}
			ref, err := foundry.ParseReference(dep.Mold)
			if err != nil {
				continue
			}
			ck := NodeKey{Source: ref.CacheKey(), Subpath: ref.Subpath}
			if seen[ck] {
				continue
			}
			seen[ck] = true
			children[k] = append(children[k], ck)
		}
	}
	visited := map[NodeKey]bool{}
	var post func(k NodeKey)
	post = func(k NodeKey) {
		if visited[k] {
			return
		}
		visited[k] = true
		for _, c := range children[k] {
			post(c)
		}
		if n, ok := state.nodes[k]; ok {
			graph.Nodes = append(graph.Nodes, *n)
		}
	}
	post(rootKey)
	return graph, nil
}

// constraintRef pairs a parent's identity with the constraint it expressed.
type constraintRef struct {
	parent NodeKey // empty parent = direct/root edge
	value  string  // the raw `version:` field from the dep entry
}

// nonSemverRef records a non-semver pin (branch or SHA) so we can verify
// that all encountered references agree.
type nonSemverRef struct {
	parent NodeKey
	kind   foundry.RefType
	value  string
}

type buildState struct {
	fetcher       Fetcher
	nodes         map[NodeKey]*Node
	constraints   map[NodeKey][]constraintRef
	order         []NodeKey // pre-order discovery (root first)
	visiting      map[NodeKey]bool
	visitingOrder []NodeKey
	nonSemverPins map[NodeKey][]nonSemverRef
}

func (s *buildState) walkChildren(parentKey NodeKey, parent *mold.Mold) error {
	s.visiting[parentKey] = true
	s.visitingOrder = append(s.visitingOrder, parentKey)
	defer func() {
		s.visiting[parentKey] = false
		if len(s.visitingOrder) > 0 {
			s.visitingOrder = s.visitingOrder[:len(s.visitingOrder)-1]
		}
	}()

	for _, dep := range parent.Dependencies {
		kind, kerr := dep.Kind()
		if kerr != nil {
			return fmt.Errorf("dependency on %s: %w", parentKey, kerr)
		}
		if kind != "mold" {
			continue
		}
		ref, err := foundry.ParseReference(dep.Mold)
		if err != nil {
			return fmt.Errorf("parsing mold dep %q in %s: %w", dep.Mold, parentKey, err)
		}
		// `version:` field on the dep entry is the constraint; it overrides any
		// version embedded in the `mold:` reference itself.
		if dep.Version != "" {
			ref.Version = dep.Version
			ref.Type = classifyConstraint(dep.Version)
		}

		childKey := NodeKey{Source: ref.CacheKey(), Subpath: ref.Subpath}

		// Cycle: child is currently in the visiting stack.
		if s.visiting[childKey] {
			return s.cycleError(childKey)
		}

		// Record the constraint or non-semver pin.
		switch ref.Type {
		case foundry.Constraint, foundry.Exact, foundry.Latest:
			s.constraints[childKey] = append(s.constraints[childKey], constraintRef{
				parent: parentKey,
				value:  dep.Version,
			})
		case foundry.Branch, foundry.SHA:
			s.nonSemverPins[childKey] = append(s.nonSemverPins[childKey], nonSemverRef{
				parent: parentKey,
				kind:   ref.Type,
				value:  ref.Version,
			})
		}

		// Record / merge the parent edge on the child (whether or not it's new).
		s.recordParentEdge(childKey, parentKey, dep)

		// Already discovered? Skip the recursive fetch — its subtree was
		// already walked. Constraints/pins still accumulate via the records
		// above. (Re-walk is unnecessary because the same mold yields the same
		// dep declarations regardless of which path led to it.)
		// Note: recordParentEdge may have pre-allocated a Node with nil Mold
		// just to attach the edge, so check Mold != nil rather than presence.
		if existing, seen := s.nodes[childKey]; seen && existing.Mold != nil {
			continue
		}

		// Discover: fetch with the current constraint just to learn the child's
		// transitive deps. The chosen version may be replaced during resolveAll
		// when more constraints are intersected.
		fr, err := s.fetcher.Fetch(ref)
		if err != nil {
			return fmt.Errorf("fetching %s: %w", childKey, err)
		}
		// If recordParentEdge pre-created the Node, fill it in; otherwise add a
		// fresh one. Either way append to the order list once.
		child := s.nodes[childKey]
		if child == nil {
			child = &Node{Key: childKey}
			s.nodes[childKey] = child
		}
		child.Mold = fr.Mold
		child.Ref = ref
		child.Version = fr.Version
		child.Commit = fr.Commit
		s.order = append(s.order, childKey)

		if err := s.walkChildren(childKey, fr.Mold); err != nil {
			return err
		}
	}
	return nil
}

func (s *buildState) recordParentEdge(child, parent NodeKey, dep mold.Dependency) {
	n := s.nodes[child]
	if n == nil {
		// Pre-create so the edge can land before we fetch (used for cycle case).
		n = &Node{Key: child}
		s.nodes[child] = n
	}
	for _, e := range n.Parents {
		if e.Source == parent.Source && e.Subpath == parent.Subpath {
			// Already recorded (re-encountered same edge). Merge with-values.
			if dep.With != nil {
				if n.With == nil {
					n.With = map[string]any{}
				}
				mergeMap(n.With, dep.With)
			}
			return
		}
	}
	n.Parents = append(n.Parents, ParentEdge{
		Source:  parent.Source,
		Subpath: parent.Subpath,
		As:      dep.As,
		With:    cloneMap(dep.With),
	})
	if dep.With != nil {
		if n.With == nil {
			n.With = map[string]any{}
		}
		mergeMap(n.With, dep.With)
	}
}

func (s *buildState) cycleError(reentry NodeKey) error {
	// visitingOrder has the path from root to the parent; reentry closes it.
	var pathParts []string
	for _, k := range s.visitingOrder {
		pathParts = append(pathParts, k.String())
	}
	pathParts = append(pathParts, reentry.String())
	return fmt.Errorf("dependency cycle detected: %s", strings.Join(pathParts, " → "))
}

// resolveAll pins each non-root node to a concrete version satisfying every
// accumulated constraint. Direct cast (root) keeps its own ref as-is.
func (s *buildState) resolveAll(rootKey NodeKey) error {
	var errs []string
	for _, key := range s.order {
		if key == rootKey {
			continue
		}

		// Non-semver pins: every reference must agree on the same value.
		if pins := s.nonSemverPins[key]; len(pins) > 0 {
			first := pins[0].value
			for _, p := range pins[1:] {
				if p.value != first {
					errs = append(errs, fmt.Sprintf(
						"%s: incompatible non-semver pins (%q vs %q)", key, first, p.value,
					))
				}
			}
			// If we also have semver constraints alongside non-semver pins,
			// that's a hard conflict — we can't satisfy both at once.
			if len(s.constraints[key]) > 0 {
				errs = append(errs, fmt.Sprintf(
					"%s: pinned to non-semver %q but also has semver constraints", key, first,
				))
			}
			// The discovery fetch already used the pin; nothing further to do.
			continue
		}

		// Semver constraints: intersect and pick highest matching tag.
		raws := s.constraints[key]
		if len(raws) == 0 {
			continue
		}
		// Aggregate raw constraint strings into a slice; "" or "latest" mean
		// no constraint for purposes of intersection.
		var compiled []*semver.Constraints
		var rawValues []string
		for _, c := range raws {
			rawValues = append(rawValues, c.value)
			if c.value == "" || c.value == "latest" {
				continue
			}
			cc, err := semver.NewConstraint(c.value)
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s: invalid constraint %q from %s: %v", key, c.value, c.parent, err))
				continue
			}
			compiled = append(compiled, cc)
		}

		tags, err := s.fetcher.Tags(key.Source, key.Subpath)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: cannot list tags: %v", key, err))
			continue
		}

		bestTag, bestSHA, ok := highestSatisfying(tags, compiled)
		if !ok {
			errs = append(errs, fmt.Sprintf(
				"%s: no version satisfies all accumulated constraints (%s)", key, strings.Join(rawValues, ", "),
			))
			continue
		}

		// If the chosen version differs from what we fetched during discovery,
		// re-fetch so the mold.yaml reflects the actual installed version.
		// (Single-pass: if the new version's transitive deps differ, the user
		// will need to recast — but for the typical case where dep declarations
		// are stable across patch versions, this single re-fetch is enough.)
		n := s.nodes[key]
		if n.Version != bestTag && n.Version != strings.TrimPrefix(bestTag, "v") {
			ref := *n.Ref
			ref.Version = bestTag
			ref.Type = foundry.Exact
			fr, ferr := s.fetcher.Fetch(&ref)
			if ferr != nil {
				errs = append(errs, fmt.Sprintf("%s: re-fetching at %s: %v", key, bestTag, ferr))
				continue
			}
			n.Mold = fr.Mold
			n.Ref = &ref
			n.Version = bestTag
			n.Commit = bestSHA
		} else {
			// Use the SHA from the tag listing so we have an authoritative pin.
			n.Commit = bestSHA
			n.Version = bestTag
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("dependency resolution failed:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

// classifyConstraint mirrors foundry.classifyVersion (which is unexported)
// for the depgraph's needs.
func classifyConstraint(v string) foundry.RefType {
	if v == "" || v == "latest" {
		return foundry.Latest
	}
	// Constraint operators.
	switch v[0] {
	case '^', '~', '>', '<', '=', '!':
		return foundry.Constraint
	}
	// SHA-ish (hex 7-40).
	if isHex(v) && len(v) >= 7 && len(v) <= 40 {
		return foundry.SHA
	}
	// Semver-ish (starts with a digit or v<digit>).
	if (v[0] >= '0' && v[0] <= '9') || (len(v) > 1 && v[0] == 'v' && v[1] >= '0' && v[1] <= '9') {
		return foundry.Exact
	}
	return foundry.Branch
}

func isHex(s string) bool {
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		case r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
}

// highestSatisfying picks the highest-versioned tag that satisfies every
// supplied constraint. Candidates are ranked by their mold.yaml version when
// known, falling back to the tag-embedded semver. Returns (tag, sha, true) on
// success.
func highestSatisfying(tags map[string]TagInfo, constraints []*semver.Constraints) (string, string, bool) {
	type entry struct {
		tag    string
		sha    string
		ver    *semver.Version // rank version (mold version when known)
		tagVer *semver.Version // tag-embedded version, for tie-breaking
	}
	var candidates []entry
	for tag, info := range tags {
		v, ok := foundry.RankVersion(tag, info.MoldVersion)
		if !ok {
			continue
		}
		for _, c := range constraints {
			if !c.Check(v) {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		tagVer, _ := foundry.RankVersion(tag, "")
		candidates = append(candidates, entry{tag: tag, sha: info.SHA, ver: v, tagVer: tagVer})
	}
	if len(candidates) == 0 {
		return "", "", false
	}
	// Order by rank version, then by tag-embedded version as a tie-breaker:
	// release-train monorepos share one mold version across many tags, so the
	// newest tag is the right checkout pointer.
	sort.Slice(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		if !a.ver.Equal(b.ver) {
			return a.ver.LessThan(b.ver)
		}
		if a.tagVer != nil && b.tagVer != nil && !a.tagVer.Equal(b.tagVer) {
			return a.tagVer.LessThan(b.tagVer)
		}
		return a.tag < b.tag
	})
	best := candidates[len(candidates)-1]
	return best.tag, best.sha, true
}

func cloneMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// mergeMap shallow-copies entries from src into dst, src wins on key collision.
func mergeMap(dst, src map[string]any) {
	for k, v := range src {
		dst[k] = v
	}
}
