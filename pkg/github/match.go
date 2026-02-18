package github

import "strings"

// MatchFieldByName finds a field whose name matches the given name.
// It tries exact case-insensitive match first, then falls back to substring containment.
// Returns nil if no match is found.
func MatchFieldByName(fields []Field, name string) *Field {
	lower := strings.ToLower(name)

	// Exact case-insensitive match
	for i := range fields {
		if strings.EqualFold(fields[i].Name, name) {
			return &fields[i]
		}
	}

	// Fuzzy fallback: field name contains the search term
	for i := range fields {
		if strings.Contains(strings.ToLower(fields[i].Name), lower) {
			return &fields[i]
		}
	}

	return nil
}

// MatchOptionByName finds an option whose name matches the given label (case-insensitive).
// Returns nil if no match is found.
func MatchOptionByName(options []Option, label string) *Option {
	for i := range options {
		if strings.EqualFold(options[i].Name, label) {
			return &options[i]
		}
	}
	return nil
}

// AutoMapModel attempts to match a model's field name and option labels
// to the discovered project fields. Returns the matched field (or nil) and
// a map of concept keys to matched options.
func AutoMapModel(
	fields []Field,
	modelFieldName string,
	modelOptions map[string]string, // concept key -> label
) (matchedField *Field, matchedOptions map[string]*Option) {
	matchedField = MatchFieldByName(fields, modelFieldName)
	matchedOptions = make(map[string]*Option)

	if matchedField != nil && len(matchedField.Options) > 0 {
		for conceptKey, label := range modelOptions {
			if opt := MatchOptionByName(matchedField.Options, label); opt != nil {
				matchedOptions[conceptKey] = opt
			}
		}
	}
	return
}
