package merge

type nodeKind int

const (
	kindScalar nodeKind = iota
	kindMap
	kindSeq
)

// node is a format-agnostic ordered representation used by both
// the JSON and YAML backends. Maps preserve insertion order via `keys`.
type node struct {
	kind   nodeKind
	scalar any              // for kindScalar (string, bool, nil, json.Number, int64, float64)
	keys   []string         // ordered keys for kindMap
	fields map[string]*node // map[string]*node for kindMap
	seq    []*node          // ordered entries for kindSeq
}

// mergeNodes returns the result of merging overlay into base.
//
// Rules:
//   - kind mismatch: overlay wins (returned as-is).
//   - both kindScalar: overlay wins.
//   - both kindMap: recursive deep-merge. Existing base keys keep their
//     position; overlay keys not in base are appended at the end in
//     overlay's declared order.
//   - both kindSeq: concat overlay onto base, then dedupe by deep value
//     equality. Order is preserved (base first, then overlay-only).
//
// mergeNodes does not mutate its inputs; the returned tree may share
// child pointers with base/overlay.
func mergeNodes(base, overlay *node) *node {
	if base == nil {
		return overlay
	}
	if overlay == nil {
		return base
	}
	if base.kind != overlay.kind {
		return overlay
	}
	switch base.kind {
	case kindScalar:
		return overlay
	case kindMap:
		out := &node{
			kind:   kindMap,
			fields: make(map[string]*node, len(base.fields)+len(overlay.fields)),
		}
		// Existing keys (in base order), recursively merged.
		for _, k := range base.keys {
			bv := base.fields[k]
			if ov, ok := overlay.fields[k]; ok {
				out.keys = append(out.keys, k)
				out.fields[k] = mergeNodes(bv, ov)
			} else {
				out.keys = append(out.keys, k)
				out.fields[k] = bv
			}
		}
		// Overlay-only keys, appended.
		for _, k := range overlay.keys {
			if _, ok := base.fields[k]; ok {
				continue
			}
			out.keys = append(out.keys, k)
			out.fields[k] = overlay.fields[k]
		}
		return out
	case kindSeq:
		out := &node{kind: kindSeq, seq: make([]*node, 0, len(base.seq)+len(overlay.seq))}
		out.seq = append(out.seq, base.seq...)
		for _, oe := range overlay.seq {
			dup := false
			for _, be := range out.seq {
				if nodeEqual(be, oe) {
					dup = true
					break
				}
			}
			if !dup {
				out.seq = append(out.seq, oe)
			}
		}
		return out
	}
	return overlay
}

// nodeEqual is recursive structural equality.
//   - scalars compare via Go ==
//   - maps are equal if they have the same key set with recursive-equal values
//     (order-insensitive — equality is a set property)
//   - sequences are equal only if same length and element-wise equal
//     (order-sensitive)
func nodeEqual(a, b *node) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.kind != b.kind {
		return false
	}
	switch a.kind {
	case kindScalar:
		return a.scalar == b.scalar
	case kindMap:
		if len(a.fields) != len(b.fields) {
			return false
		}
		for k, av := range a.fields {
			bv, ok := b.fields[k]
			if !ok {
				return false
			}
			if !nodeEqual(av, bv) {
				return false
			}
		}
		return true
	case kindSeq:
		if len(a.seq) != len(b.seq) {
			return false
		}
		for i := range a.seq {
			if !nodeEqual(a.seq[i], b.seq[i]) {
				return false
			}
		}
		return true
	}
	return false
}
