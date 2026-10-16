package merge

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

// loadTOML parses TOML bytes into an order-preserving *node tree.
//
// BurntSushi/toml decodes into unordered Go maps, so key order is rebuilt
// from MetaData.Keys(), which lists every key in document order. Ranks are
// tracked per table path with array-of-tables indices dropped, so every
// element of an array of tables shares one key order.
//
// Scalars keep their TOML types: int64, float64, string, bool, and
// time.Time (local dates/times carry BurntSushi's sentinel locations so they
// re-encode in their original form).
func loadTOML(data []byte) (*node, error) {
	var raw map[string]any
	md, err := toml.Decode(string(data), &raw)
	if err != nil {
		return nil, err
	}
	order := map[string]map[string]int{}
	for _, k := range md.Keys() {
		for i := range k {
			parent := strings.Join(k[:i], "\x00")
			ranks, ok := order[parent]
			if !ok {
				ranks = map[string]int{}
				order[parent] = ranks
			}
			if _, seen := ranks[k[i]]; !seen {
				ranks[k[i]] = len(ranks)
			}
		}
	}
	return tomlToNode(raw, nil, order), nil
}

func tomlToNode(v any, path []string, order map[string]map[string]int) *node {
	switch x := v.(type) {
	case map[string]any:
		n := &node{kind: kindMap, fields: make(map[string]*node, len(x))}
		for k := range x {
			n.keys = append(n.keys, k)
		}
		ranks := order[strings.Join(path, "\x00")]
		sort.SliceStable(n.keys, func(i, j int) bool {
			ri, iok := ranks[n.keys[i]]
			rj, jok := ranks[n.keys[j]]
			switch {
			case iok && jok:
				return ri < rj
			case iok != jok:
				return iok // ranked keys first
			default:
				return n.keys[i] < n.keys[j]
			}
		})
		for _, k := range n.keys {
			n.fields[k] = tomlToNode(x[k], append(append([]string(nil), path...), k), order)
		}
		return n
	case []map[string]any:
		n := &node{kind: kindSeq}
		for _, item := range x {
			n.seq = append(n.seq, tomlToNode(item, path, order))
		}
		return n
	case []any:
		n := &node{kind: kindSeq}
		for _, item := range x {
			n.seq = append(n.seq, tomlToNode(item, path, order))
		}
		return n
	default:
		return &node{kind: kindScalar, scalar: x}
	}
}

// dumpTOML serializes a *node tree as TOML, preserving key order within each
// table. TOML requires a table's plain key/value pairs to precede its
// sub-tables, so within a table those pairs are emitted first; sub-tables and
// arrays of tables then follow in their original relative order. Non-empty arrays
// whose elements are all tables become arrays of tables ([[name]]); any
// other array is written inline. Comments are not preserved.
func dumpTOML(n *node) ([]byte, error) {
	if n.kind != kindMap {
		return nil, fmt.Errorf("toml document root must be a table")
	}
	var buf bytes.Buffer
	if err := writeTOMLTable(&buf, n, nil); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// isTOMLTableArray reports whether a sequence should be written as an array
// of tables: it is non-empty and every element is a map.
func isTOMLTableArray(n *node) bool {
	if n.kind != kindSeq || len(n.seq) == 0 {
		return false
	}
	for _, e := range n.seq {
		if e.kind != kindMap {
			return false
		}
	}
	return true
}

func writeTOMLTable(buf *bytes.Buffer, n *node, path []string) error {
	// Plain key/value pairs.
	for _, k := range n.keys {
		v := n.fields[k]
		if v.kind == kindMap || isTOMLTableArray(v) {
			continue
		}
		buf.WriteString(tomlKey(k))
		buf.WriteString(" = ")
		if err := writeTOMLInline(buf, v); err != nil {
			return fmt.Errorf("key %s: %w", strings.Join(append(path, k), "."), err)
		}
		buf.WriteByte('\n')
	}
	// Sub-tables and arrays of tables.
	for _, k := range n.keys {
		v := n.fields[k]
		child := append(append([]string(nil), path...), k)
		switch {
		case v.kind == kindMap:
			if tomlNeedsHeader(v) {
				writeTOMLHeader(buf, child, false)
			}
			if err := writeTOMLTable(buf, v, child); err != nil {
				return err
			}
		case isTOMLTableArray(v):
			for _, e := range v.seq {
				writeTOMLHeader(buf, child, true)
				if err := writeTOMLTable(buf, e, child); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// tomlNeedsHeader reports whether a table must be introduced with its own
// [header]. A table holding only sub-tables is created implicitly by its
// children's headers, so the header is omitted to keep output canonical.
func tomlNeedsHeader(n *node) bool {
	if len(n.keys) == 0 {
		return true
	}
	for _, k := range n.keys {
		v := n.fields[k]
		if v.kind != kindMap && !isTOMLTableArray(v) {
			return true
		}
	}
	return false
}

func writeTOMLHeader(buf *bytes.Buffer, path []string, array bool) {
	if buf.Len() > 0 {
		buf.WriteByte('\n')
	}
	parts := make([]string, len(path))
	for i, p := range path {
		parts[i] = tomlKey(p)
	}
	if array {
		buf.WriteString("[[" + strings.Join(parts, ".") + "]]\n")
	} else {
		buf.WriteString("[" + strings.Join(parts, ".") + "]\n")
	}
}

func writeTOMLInline(buf *bytes.Buffer, n *node) error {
	switch n.kind {
	case kindScalar:
		s, err := tomlScalar(n.scalar)
		if err != nil {
			return err
		}
		buf.WriteString(s)
	case kindSeq:
		buf.WriteByte('[')
		for i, e := range n.seq {
			if i > 0 {
				buf.WriteString(", ")
			}
			if err := writeTOMLInline(buf, e); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	case kindMap:
		if len(n.keys) == 0 {
			buf.WriteString("{}")
			return nil
		}
		buf.WriteString("{ ")
		for i, k := range n.keys {
			if i > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(tomlKey(k))
			buf.WriteString(" = ")
			if err := writeTOMLInline(buf, n.fields[k]); err != nil {
				return err
			}
		}
		buf.WriteString(" }")
	}
	return nil
}

// tomlScalar formats one scalar value by delegating to BurntSushi's encoder,
// which already handles TOML string escaping, float forms (1.0, inf, nan)
// and offset/local date-times.
func tomlScalar(v any) (string, error) {
	if v == nil {
		return "", fmt.Errorf("toml has no null value")
	}
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(map[string]any{"v": v}); err != nil {
		return "", err
	}
	s := strings.TrimSuffix(buf.String(), "\n")
	out, ok := strings.CutPrefix(s, "v = ")
	if !ok || strings.Contains(out, "\n") {
		return "", fmt.Errorf("unsupported toml scalar %T", v)
	}
	return out, nil
}

var tomlBareKey = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func tomlKey(k string) string {
	if tomlBareKey.MatchString(k) {
		return k
	}
	s, err := tomlScalar(k)
	if err != nil {
		// Strings always encode; keep a safe fallback anyway.
		return fmt.Sprintf("%q", k)
	}
	return s
}
