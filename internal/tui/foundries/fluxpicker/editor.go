package fluxpicker

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

// buildEditorForm returns a huh.Form for editing one FluxVar. The form's
// state is driven by `value` (string-typed) for most fields and `boolVal`
// for booleans. On submit, commitEditorValue parses the appropriate field
// back into the typed override.
func buildEditorForm(fv mold.FluxVar, value *string, boolVal *bool) *huh.Form {
	switch fv.Type {
	case "bool":
		return huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(fv.Name).
					Description(fv.Description).
					Affirmative("Yes").
					Negative("No").
					Value(boolVal),
			),
		)
	case "int":
		return huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title(fv.Name).
					Description(fv.Description).
					Value(value).
					Validate(func(s string) error {
						if s == "" {
							return nil
						}
						if _, err := strconv.Atoi(s); err != nil {
							return fmt.Errorf("must be an integer")
						}
						return nil
					}),
			),
		)
	case "list":
		return huh.NewForm(
			huh.NewGroup(
				huh.NewText().
					Title(fv.Name).
					Description(fv.Description).
					Lines(4).
					Placeholder("comma- or newline-separated").
					Value(value),
			),
		)
	case "select":
		opts := make([]huh.Option[string], 0, len(fv.Options))
		for _, o := range fv.Options {
			opts = append(opts, huh.NewOption(o.Label, o.Value))
		}
		return huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title(fv.Name).
					Description(fv.Description).
					Options(opts...).
					Value(value),
			),
		)
	default: // string
		return huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title(fv.Name).
					Description(fv.Description).
					Value(value),
			),
		)
	}
}

// commitEditorValue parses raw user input into the typed override and stores
// it on the model. Empty input clears the override. Validation failures are
// recorded on m.err and the override is left unchanged.
func commitEditorValue(m Model, fv mold.FluxVar, raw string) Model {
	if m.overrides == nil {
		m.overrides = map[string]any{}
	}
	if strings.TrimSpace(raw) == "" {
		delete(m.overrides, fv.Name)
		m.err = nil
		return m
	}
	switch fv.Type {
	case "int":
		n, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil {
			m.err = fmt.Errorf("%s: must be an integer", fv.Name)
			return m
		}
		m.overrides[fv.Name] = n
	case "bool":
		switch strings.ToLower(strings.TrimSpace(raw)) {
		case "true", "yes", "y", "1":
			m.overrides[fv.Name] = true
		case "false", "no", "n", "0":
			m.overrides[fv.Name] = false
		default:
			m.err = fmt.Errorf("%s: must be true/false", fv.Name)
			return m
		}
	case "list":
		var parts []string
		for _, p := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == '\n' }) {
			t := strings.TrimSpace(p)
			if t == "" {
				continue
			}
			parts = append(parts, t)
		}
		m.overrides[fv.Name] = parts
	case "select":
		v := strings.TrimSpace(raw)
		if len(fv.Options) > 0 {
			ok := false
			for _, o := range fv.Options {
				if o.Value == v {
					ok = true
					break
				}
			}
			if !ok {
				m.err = fmt.Errorf("%s: %q is not a valid option", fv.Name, v)
				return m
			}
		}
		m.overrides[fv.Name] = v
	default:
		m.overrides[fv.Name] = raw
	}
	m.err = nil
	return m
}

// ErrUnknownVar is returned when the editor is asked to commit a variable
// not present in the schema.
var ErrUnknownVar = errors.New("unknown flux variable")

// findVar searches a schema by name.
func findVar(schema []mold.FluxVar, name string) (mold.FluxVar, bool) {
	for _, v := range schema {
		if v.Name == name {
			return v, true
		}
	}
	return mold.FluxVar{}, false
}
