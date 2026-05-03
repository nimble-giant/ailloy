package merge

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// loadJSON parses JSON bytes into an order-preserving *node tree.
// Integers are preserved as json.Number to avoid float coercion.
func loadJSON(data []byte) (*node, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	n, err := decodeJSONValue(dec)
	if err != nil {
		return nil, err
	}
	// Anything other than EOF means there's trailing content.
	if tok, err := dec.Token(); err == nil {
		return nil, fmt.Errorf("trailing data after JSON value: %v", tok)
	} else if !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("trailing data after JSON value: %w", err)
	}
	return n, nil
}

func decodeJSONValue(dec *json.Decoder) (*node, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	return decodeJSONFromToken(dec, tok)
}

func decodeJSONFromToken(dec *json.Decoder, tok json.Token) (*node, error) {
	switch v := tok.(type) {
	case json.Delim:
		switch v {
		case '{':
			n := &node{kind: kindMap, fields: map[string]*node{}}
			for dec.More() {
				keyTok, err := dec.Token()
				if err != nil {
					return nil, err
				}
				key, ok := keyTok.(string)
				if !ok {
					return nil, fmt.Errorf("non-string JSON key: %v", keyTok)
				}
				val, err := decodeJSONValue(dec)
				if err != nil {
					return nil, err
				}
				n.keys = append(n.keys, key)
				n.fields[key] = val
			}
			// Consume closing '}'.
			if _, err := dec.Token(); err != nil {
				return nil, err
			}
			return n, nil
		case '[':
			n := &node{kind: kindSeq}
			for dec.More() {
				val, err := decodeJSONValue(dec)
				if err != nil {
					return nil, err
				}
				n.seq = append(n.seq, val)
			}
			// Consume closing ']'.
			if _, err := dec.Token(); err != nil {
				return nil, err
			}
			return n, nil
		default:
			return nil, fmt.Errorf("unexpected delim %v", v)
		}
	default:
		// Scalar: string, bool, nil, or json.Number.
		return &node{kind: kindScalar, scalar: tok}, nil
	}
}

// dumpJSON serializes a *node tree as 2-space-indented JSON, preserving
// map key order and trailing newline.
func dumpJSON(n *node) ([]byte, error) {
	var buf bytes.Buffer
	if err := writeJSONNode(&buf, n, 0); err != nil {
		return nil, err
	}
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

func writeJSONNode(buf *bytes.Buffer, n *node, indent int) error {
	switch n.kind {
	case kindScalar:
		return writeJSONScalar(buf, n.scalar)
	case kindMap:
		if len(n.keys) == 0 {
			buf.WriteString("{}")
			return nil
		}
		buf.WriteString("{\n")
		for i, k := range n.keys {
			writeIndent(buf, indent+1)
			if err := writeJSONString(buf, k); err != nil {
				return err
			}
			buf.WriteString(": ")
			if err := writeJSONNode(buf, n.fields[k], indent+1); err != nil {
				return err
			}
			if i < len(n.keys)-1 {
				buf.WriteByte(',')
			}
			buf.WriteByte('\n')
		}
		writeIndent(buf, indent)
		buf.WriteByte('}')
		return nil
	case kindSeq:
		if len(n.seq) == 0 {
			buf.WriteString("[]")
			return nil
		}
		buf.WriteString("[\n")
		for i, item := range n.seq {
			writeIndent(buf, indent+1)
			if err := writeJSONNode(buf, item, indent+1); err != nil {
				return err
			}
			if i < len(n.seq)-1 {
				buf.WriteByte(',')
			}
			buf.WriteByte('\n')
		}
		writeIndent(buf, indent)
		buf.WriteByte(']')
		return nil
	}
	return fmt.Errorf("unknown node kind %v", n.kind)
}

func writeIndent(buf *bytes.Buffer, depth int) {
	buf.WriteString(strings.Repeat("  ", depth))
}

func writeJSONScalar(buf *bytes.Buffer, v any) error {
	switch s := v.(type) {
	case nil:
		buf.WriteString("null")
		return nil
	case bool:
		if s {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
		return nil
	case json.Number:
		buf.WriteString(string(s))
		return nil
	case string:
		return writeJSONString(buf, s)
	default:
		// Fallback: defer to encoding/json (covers float64, int — only
		// reached if loaders ever produce non-Number numerics).
		out, err := json.Marshal(v)
		if err != nil {
			return err
		}
		buf.Write(out)
		return nil
	}
}

func writeJSONString(buf *bytes.Buffer, s string) error {
	var tmp bytes.Buffer
	enc := json.NewEncoder(&tmp)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(s); err != nil {
		return err
	}
	// Encoder.Encode appends a trailing newline; strip it.
	out := tmp.Bytes()
	if len(out) > 0 && out[len(out)-1] == '\n' {
		out = out[:len(out)-1]
	}
	buf.Write(out)
	return nil
}
