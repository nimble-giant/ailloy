package merge

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
)

// loadYAML parses YAML bytes into an order-preserving *node tree.
// Uses yaml.UseOrderedMap so map keys come back as yaml.MapSlice with
// declared order intact.
func loadYAML(data []byte) (*node, error) {
	var raw any
	if err := yaml.UnmarshalWithOptions(data, &raw, yaml.UseOrderedMap()); err != nil {
		return nil, err
	}
	return yamlToNode(raw)
}

func yamlToNode(v any) (*node, error) {
	switch x := v.(type) {
	case nil:
		return &node{kind: kindScalar, scalar: nil}, nil
	case yaml.MapSlice:
		n := &node{kind: kindMap, fields: map[string]*node{}}
		for _, item := range x {
			ks, ok := item.Key.(string)
			if !ok {
				return nil, fmt.Errorf("non-string YAML key: %v (only string keys are supported)", item.Key)
			}
			child, err := yamlToNode(item.Value)
			if err != nil {
				return nil, err
			}
			if _, exists := n.fields[ks]; !exists {
				n.keys = append(n.keys, ks)
			}
			n.fields[ks] = child
		}
		return n, nil
	case []any:
		n := &node{kind: kindSeq}
		for _, item := range x {
			child, err := yamlToNode(item)
			if err != nil {
				return nil, err
			}
			n.seq = append(n.seq, child)
		}
		return n, nil
	case map[string]any:
		// UseOrderedMap should always produce yaml.MapSlice. Hitting this
		// branch means key order has already been lost, which would make
		// merge output non-deterministic across runs.
		return nil, fmt.Errorf("yaml decoded as unordered map; UseOrderedMap not applied")
	default:
		return &node{kind: kindScalar, scalar: x}, nil
	}
}

// dumpYAML serializes a *node tree to YAML bytes via yaml.Marshal on a
// reconstructed MapSlice/[]any tree (which preserves insertion order).
func dumpYAML(n *node) ([]byte, error) {
	v, err := nodeToYAMLValue(n)
	if err != nil {
		return nil, err
	}
	return yaml.MarshalWithOptions(v, yaml.Indent(2))
}

// yamlAmbiguousStringScalar is a string wrapper whose MarshalYAML emits the
// value in double-quoted form. goccy/go-yaml's default marshaler leaves a few
// strings bare that should be quoted — notably the document-separator tokens
// "---" / "...", whitespace-only strings, and strings containing newlines —
// which means re-parsing the dump yields a different value (often nil or "")
// rather than the original string. We wrap such values to force quoting.
type yamlAmbiguousStringScalar string

func (s yamlAmbiguousStringScalar) MarshalYAML() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", string(s))), nil
}

// yamlForceQuotedItem and yamlForceQuotedMap let us emit a YAML mapping where
// specific keys that goccy/go-yaml would otherwise emit bare (and which would
// then fail to re-parse — e.g., the merge key "<<") are force-quoted. Goccy's
// MapSlice encoder requires string keys and won't honor a custom MarshalYAML
// at the key level, so we hand off the entire mapping to a custom marshaler.
type yamlForceQuotedItem struct {
	key   string
	value any
}

type yamlForceQuotedMap struct {
	items []yamlForceQuotedItem
}

func (m yamlForceQuotedMap) MarshalYAML() ([]byte, error) {
	if len(m.items) == 0 {
		return []byte("{}"), nil
	}
	var buf bytes.Buffer
	for i, it := range m.items {
		if i > 0 {
			buf.WriteByte('\n')
		}
		if needsForceQuoteAsYAMLKey(it.key) {
			buf.WriteString(fmt.Sprintf("%q", it.key))
		} else {
			// Defer to goccy for non-problematic keys.
			kb, err := yaml.Marshal(it.key)
			if err != nil {
				return nil, err
			}
			// Strip trailing newline.
			kb = bytes.TrimRight(kb, "\n")
			buf.Write(kb)
		}
		buf.WriteString(": ")
		vb, err := yaml.MarshalWithOptions(it.value, yaml.Indent(2))
		if err != nil {
			return nil, err
		}
		// If the value is a multi-line block, indent continuation lines.
		vbStr := string(bytes.TrimRight(vb, "\n"))
		if strings.Contains(vbStr, "\n") {
			// Place value on next line, indented.
			buf.WriteString("\n  ")
			vbStr = strings.ReplaceAll(vbStr, "\n", "\n  ")
		}
		buf.WriteString(vbStr)
	}
	return buf.Bytes(), nil
}

// needsForceQuoteForYAML reports whether a string scalar would round-trip as a
// different value (or fail to parse) when emitted bare by goccy/go-yaml. We
// determine this empirically: marshal the value, re-parse it, and compare.
// Any inequality means we must force quoting.
func needsForceQuoteForYAML(s string) bool {
	out, err := yaml.Marshal(s)
	if err != nil {
		return true
	}
	var v any
	if err := yaml.Unmarshal(out, &v); err != nil {
		return true
	}
	got, ok := v.(string)
	if !ok {
		return true
	}
	return got != s
}

// needsForceQuoteAsYAMLKey reports whether a string used as a mapping key
// would fail to round-trip when emitted bare by goccy/go-yaml. Some tokens
// that are valid as values are special as keys — notably the merge key "<<",
// which goccy emits bare but then refuses to re-parse against a null value.
func needsForceQuoteAsYAMLKey(s string) bool {
	if needsForceQuoteForYAML(s) {
		return true
	}
	out, err := yaml.Marshal(yaml.MapSlice{yaml.MapItem{Key: s, Value: nil}})
	if err != nil {
		return true
	}
	var v any
	if err := yaml.UnmarshalWithOptions(out, &v, yaml.UseOrderedMap()); err != nil {
		return true
	}
	ms, ok := v.(yaml.MapSlice)
	if !ok || len(ms) != 1 {
		return true
	}
	got, ok := ms[0].Key.(string)
	if !ok {
		return true
	}
	return got != s
}

func nodeToYAMLValue(n *node) (any, error) {
	switch n.kind {
	case kindScalar:
		if s, ok := n.scalar.(string); ok && needsForceQuoteForYAML(s) {
			return yamlAmbiguousStringScalar(s), nil
		}
		return n.scalar, nil
	case kindMap:
		out := make(yaml.MapSlice, 0, len(n.keys))
		needsCustom := false
		for _, k := range n.keys {
			child, err := nodeToYAMLValue(n.fields[k])
			if err != nil {
				return nil, err
			}
			if needsForceQuoteAsYAMLKey(k) {
				needsCustom = true
			}
			out = append(out, yaml.MapItem{Key: k, Value: child})
		}
		if needsCustom {
			// One or more keys would not round-trip when emitted bare by
			// goccy/go-yaml (e.g., the merge key "<<"), so we hand off to a
			// custom marshaler that force-quotes problematic keys.
			items := make([]yamlForceQuotedItem, 0, len(out))
			for _, it := range out {
				items = append(items, yamlForceQuotedItem{key: it.Key.(string), value: it.Value})
			}
			return yamlForceQuotedMap{items: items}, nil
		}
		return out, nil
	case kindSeq:
		out := make([]any, 0, len(n.seq))
		for _, item := range n.seq {
			v, err := nodeToYAMLValue(item)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	}
	return nil, fmt.Errorf("unknown node kind %v", n.kind)
}
