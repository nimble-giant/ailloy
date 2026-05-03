package merge

import (
	"fmt"

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
				ks = fmt.Sprintf("%v", item.Key)
			}
			child, err := yamlToNode(item.Value)
			if err != nil {
				return nil, err
			}
			n.keys = append(n.keys, ks)
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
		// Fallback for unordered map (shouldn't happen with UseOrderedMap,
		// but be defensive — keys appear in arbitrary order).
		n := &node{kind: kindMap, fields: map[string]*node{}}
		for k, vv := range x {
			child, err := yamlToNode(vv)
			if err != nil {
				return nil, err
			}
			n.keys = append(n.keys, k)
			n.fields[k] = child
		}
		return n, nil
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

func nodeToYAMLValue(n *node) (any, error) {
	switch n.kind {
	case kindScalar:
		return n.scalar, nil
	case kindMap:
		out := make(yaml.MapSlice, 0, len(n.keys))
		for _, k := range n.keys {
			child, err := nodeToYAMLValue(n.fields[k])
			if err != nil {
				return nil, err
			}
			out = append(out, yaml.MapItem{Key: k, Value: child})
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
