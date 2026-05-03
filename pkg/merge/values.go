package merge

import "reflect"

// MergeYAMLValues deep-merges overlay maps into base, overlay-wins. Nil maps
// are treated as empty. Returns a new map; inputs are not mutated.
//
// Semantics (value parity with mergeNodes used by `ailloy cast --strategy=merge`):
//   - kind mismatch (e.g., scalar vs map): overlay wins.
//   - scalar vs scalar: overlay wins.
//   - map vs map: recursive deep-merge. Base keys retain their values;
//     overlay-only keys are added.
//   - sequence vs sequence: concat base+overlay, dedupe by deep equality.
//
// Note: because the result is a map[string]any, key *iteration order* is
// not preserved (parity is on values, not on iteration order). Callers that
// need byte-stable YAML output should sort keys at marshal time.
//
// Scalar types are preserved as-is (no YAML round-trip), so callers get back
// the same Go types they put in.
func MergeYAMLValues(base map[string]any, overlays ...map[string]any) map[string]any {
	cur := cloneMap(base)
	if cur == nil {
		cur = map[string]any{}
	}
	for _, ov := range overlays {
		cur = mergeMap(cur, ov)
	}
	return cur
}

func mergeMap(base, overlay map[string]any) map[string]any {
	if base == nil {
		base = map[string]any{}
	}
	out := make(map[string]any, len(base)+len(overlay))
	for k, v := range base {
		out[k] = cloneValue(v)
	}
	for k, ov := range overlay {
		bv, present := out[k]
		if !present {
			out[k] = cloneValue(ov)
			continue
		}
		out[k] = mergeValue(bv, ov)
	}
	return out
}

func mergeValue(base, overlay any) any {
	bm, baseIsMap := base.(map[string]any)
	om, overlayIsMap := overlay.(map[string]any)
	if baseIsMap && overlayIsMap {
		return mergeMap(bm, om)
	}
	bs, baseIsSeq := base.([]any)
	os, overlayIsSeq := overlay.([]any)
	if baseIsSeq && overlayIsSeq {
		return mergeSeq(bs, os)
	}
	return cloneValue(overlay)
}

func mergeSeq(base, overlay []any) []any {
	out := make([]any, 0, len(base)+len(overlay))
	for _, v := range base {
		out = append(out, cloneValue(v))
	}
	for _, ov := range overlay {
		dup := false
		for _, ex := range out {
			if valueEqual(ex, ov) {
				dup = true
				break
			}
		}
		if !dup {
			out = append(out, cloneValue(ov))
		}
	}
	return out
}

func cloneMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = cloneValue(v)
	}
	return out
}

func cloneValue(v any) any {
	switch x := v.(type) {
	case map[string]any:
		return cloneMap(x)
	case []any:
		out := make([]any, len(x))
		for i, item := range x {
			out[i] = cloneValue(item)
		}
		return out
	default:
		return v
	}
}

func valueEqual(a, b any) bool {
	am, aIsMap := a.(map[string]any)
	bm, bIsMap := b.(map[string]any)
	if aIsMap && bIsMap {
		if len(am) != len(bm) {
			return false
		}
		for k, av := range am {
			bv, ok := bm[k]
			if !ok || !valueEqual(av, bv) {
				return false
			}
		}
		return true
	}
	as, aIsSeq := a.([]any)
	bs, bIsSeq := b.([]any)
	if aIsSeq && bIsSeq {
		if len(as) != len(bs) {
			return false
		}
		for i := range as {
			if !valueEqual(as[i], bs[i]) {
				return false
			}
		}
		return true
	}
	return reflect.DeepEqual(a, b)
}
