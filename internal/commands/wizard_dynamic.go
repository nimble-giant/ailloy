package commands

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/nimble-giant/ailloy/pkg/mold"
	"github.com/nimble-giant/ailloy/pkg/styles"
)

// orderedGroup holds a group name and its flux variables in schema order.
type orderedGroup struct {
	name string
	vars []mold.FluxVar
}

// dynamicWizard builds and runs a huh form from a flux schema.
type dynamicWizard struct {
	schema          []mold.FluxVar
	flux            map[string]any
	discovery       *mold.DiscoverExecutor
	values          map[string]*string               // bound string/int/select values
	boolVals        map[string]*bool                 // bound bool values
	textVals        map[string]*string               // bound list (multi-line text) values
	discoverResults map[string][]mold.DiscoverResult // last discovery results per field name
}

// newDynamicWizard creates a wizard from schema and existing flux values.
func newDynamicWizard(schema []mold.FluxVar, flux map[string]any) *dynamicWizard {
	w := &dynamicWizard{
		schema:          schema,
		flux:            flux,
		discovery:       mold.NewDiscoverExecutor(),
		values:          make(map[string]*string),
		boolVals:        make(map[string]*bool),
		textVals:        make(map[string]*string),
		discoverResults: make(map[string][]mold.DiscoverResult),
	}

	// Pre-populate bound values from existing flux
	for _, fv := range schema {
		switch fv.Type {
		case "bool":
			b := false
			if v, ok := getFluxBool(flux, fv.Name); ok {
				b = v
			} else if fv.Default != "" {
				b = strings.EqualFold(fv.Default, "true")
			}
			w.boolVals[fv.Name] = &b
		case "list":
			s := ""
			if v, ok := getFluxString(flux, fv.Name); ok {
				s = v
			} else if fv.Default != "" {
				s = fv.Default
			}
			w.textVals[fv.Name] = &s
		default: // string, int, select
			s := ""
			if v, ok := getFluxString(flux, fv.Name); ok {
				s = v
			} else if fv.Default != "" {
				s = fv.Default
			}
			w.values[fv.Name] = &s
		}
	}

	return w
}

// groupPrefix extracts the group name from a dotted variable name.
// "project.organization" -> "project", "ore.status.enabled" -> "ore.status"
// "simple" -> "general"
func groupPrefix(name string) string {
	idx := strings.LastIndex(name, ".")
	if idx < 0 {
		return "general"
	}
	return name[:idx]
}

// groupTitle formats a group prefix as a human-readable title.
// "project" -> "Project", "ore.status" -> "Ore > Status"
func groupTitle(prefix string) string {
	parts := strings.Split(prefix, ".")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " > ")
}

// collectGroups groups FluxVars by their dotted-path prefix, preserving schema order.
func collectGroups(schema []mold.FluxVar) []orderedGroup {
	var groups []orderedGroup
	seen := make(map[string]int) // prefix -> index in groups

	for _, fv := range schema {
		prefix := groupPrefix(fv.Name)
		if idx, ok := seen[prefix]; ok {
			groups[idx].vars = append(groups[idx].vars, fv)
		} else {
			seen[prefix] = len(groups)
			groups = append(groups, orderedGroup{name: prefix, vars: []mold.FluxVar{fv}})
		}
	}
	return groups
}

// buildGroups constructs huh.Group slices grouped by dotted-path prefix.
// Fields that have a sibling "enabled" bool are split into their own group
// with a WithHideFunc so they're skipped when disabled.
func (w *dynamicWizard) buildGroups() []*huh.Group {
	groups := collectGroups(w.schema)
	var huhGroups []*huh.Group

	sectionNum := 0
	for _, g := range groups {
		var mainFields []huh.Field
		var conditionalFields []huh.Field

		for _, fv := range g.vars {
			field := w.buildField(fv)
			if w.siblingEnabledHideFunc(fv.Name) != nil {
				conditionalFields = append(conditionalFields, field)
			} else {
				mainFields = append(mainFields, field)
			}
		}

		if len(mainFields) > 0 {
			sectionNum++
			title := fmt.Sprintf("Section %d: %s", sectionNum, groupTitle(g.name))
			huhGroups = append(huhGroups, huh.NewGroup(mainFields...).Title(title))
		}

		if len(conditionalFields) > 0 {
			sectionNum++
			title := fmt.Sprintf("Section %d: %s Configuration", sectionNum, groupTitle(g.name))
			group := huh.NewGroup(conditionalFields...).Title(title)
			// Hide the entire group when the sibling "enabled" bool is false
			enabledKey := g.name + ".enabled"
			if ptr, ok := w.boolVals[enabledKey]; ok {
				group.WithHideFunc(func() bool { return !*ptr })
			}
			huhGroups = append(huhGroups, group)
		}
	}

	return huhGroups
}

// buildField generates the appropriate huh.Field for a FluxVar.
func (w *dynamicWizard) buildField(fv mold.FluxVar) huh.Field {
	switch fv.Type {
	case "bool":
		return w.buildBoolField(fv)
	case "int":
		return w.buildIntField(fv)
	case "list":
		return w.buildListField(fv)
	case "select":
		return w.buildSelectField(fv)
	default: // string
		if fv.Discover != nil {
			return w.buildDiscoverField(fv)
		}
		return w.buildStringField(fv)
	}
}

// buildStringField creates a huh.Input for string variables.
func (w *dynamicWizard) buildStringField(fv mold.FluxVar) huh.Field {
	input := huh.NewInput().
		Title(fieldTitle(fv)).
		Description(fv.Description).
		Value(w.values[fv.Name])

	if fv.Default != "" {
		input.Placeholder(fv.Default)
	}

	if fv.Required {
		input.Validate(func(s string) error {
			if s == "" {
				return fmt.Errorf("%s is required", fv.Name)
			}
			return nil
		})
	}

	return input
}

// buildBoolField creates a huh.Confirm for bool variables.
func (w *dynamicWizard) buildBoolField(fv mold.FluxVar) huh.Field {
	return huh.NewConfirm().
		Title(fieldTitle(fv)).
		Description(fv.Description).
		Affirmative("Yes").
		Negative("No").
		Value(w.boolVals[fv.Name])
}

// buildIntField creates a huh.Input with integer validation for int variables.
func (w *dynamicWizard) buildIntField(fv mold.FluxVar) huh.Field {
	input := huh.NewInput().
		Title(fieldTitle(fv)).
		Description(fv.Description).
		Value(w.values[fv.Name]).
		Validate(func(s string) error {
			if s == "" {
				if fv.Required {
					return fmt.Errorf("%s is required", fv.Name)
				}
				return nil
			}
			if _, err := strconv.Atoi(s); err != nil {
				return fmt.Errorf("must be an integer")
			}
			return nil
		})

	if fv.Default != "" {
		input.Placeholder(fv.Default)
	}

	return input
}

// buildListField creates a huh.Text (multi-line) for list variables.
func (w *dynamicWizard) buildListField(fv mold.FluxVar) huh.Field {
	text := huh.NewText().
		Title(fieldTitle(fv)).
		Description(fv.Description + " (comma-separated or one per line)").
		Value(w.textVals[fv.Name]).
		Lines(4)

	if fv.Default != "" {
		text.Placeholder(fv.Default)
	}

	return text
}

// buildSelectField creates a huh.Select for select-type variables with static options.
func (w *dynamicWizard) buildSelectField(fv mold.FluxVar) huh.Field {
	if fv.Discover != nil {
		return w.buildDiscoverField(fv)
	}

	opts := make([]huh.Option[string], 0, len(fv.Options))
	for _, o := range fv.Options {
		opts = append(opts, huh.NewOption(o.Label, o.Value))
	}

	sel := huh.NewSelect[string]().
		Title(fieldTitle(fv)).
		Description(fv.Description).
		Options(opts...).
		Value(w.values[fv.Name])

	return sel
}

// buildDiscoverField creates a huh.Select with lazy discovery via OptionsFunc.
func (w *dynamicWizard) buildDiscoverField(fv mold.FluxVar) huh.Field {
	return huh.NewSelect[string]().
		Title(fieldTitle(fv)).
		Description(fv.Description).
		OptionsFunc(func() []huh.Option[string] {
			return w.runDiscovery(fv)
		}, w.fluxDeps(fv)).
		Value(w.values[fv.Name]).
		Height(10)
}

// siblingEnabledHideFunc returns a function that hides a field when a sibling
// "enabled" bool in the same group prefix is false. Returns nil if no such
// sibling exists or if the field IS the enabled bool itself.
// e.g., for "ore.status.field_id", checks "ore.status.enabled"
func (w *dynamicWizard) siblingEnabledHideFunc(name string) func() bool {
	prefix := groupPrefix(name)
	enabledKey := prefix + ".enabled"
	// Don't hide the enabled toggle itself
	if name == enabledKey {
		return nil
	}
	ptr, ok := w.boolVals[enabledKey]
	if !ok || ptr == nil {
		return nil
	}
	return func() bool {
		return !*ptr
	}
}

// runDiscovery executes a discover spec and returns huh options.
// Falls back to a manual entry option on failure.
// If template dependencies (e.g. {{.project.organization}}) are not yet
// populated, returns a placeholder prompting the user to fill them in first.
func (w *dynamicWizard) runDiscovery(fv mold.FluxVar) []huh.Option[string] {
	if fv.Discover == nil {
		return []huh.Option[string]{huh.NewOption("(no discovery configured)", "")}
	}

	// Build current flux state from bound values for template expansion
	currentFlux := w.currentFlux()

	// Check if required template variables are populated before running
	if missing := missingTemplateDeps(fv.Discover.Command, currentFlux); len(missing) > 0 {
		hint := strings.Join(missing, ", ")
		return []huh.Option[string]{
			huh.NewOption(fmt.Sprintf("(waiting — fill in %s first)", hint), ""),
		}
	}

	results, err := w.discovery.Run(*fv.Discover, currentFlux)
	if err != nil {
		return []huh.Option[string]{
			huh.NewOption(fmt.Sprintf("(discovery failed: %s)", err), ""),
		}
	}

	if len(results) == 0 {
		return []huh.Option[string]{huh.NewOption("(no options discovered)", "")}
	}

	// Store results for also_sets lookup after selection
	w.discoverResults[fv.Name] = results

	opts := make([]huh.Option[string], 0, len(results)+1)
	opts = append(opts, huh.NewOption("(skip)", ""))
	for _, r := range results {
		opts = append(opts, huh.NewOption(r.Label, r.Value))
	}
	return opts
}

// missingTemplateDeps extracts {{.dotted.path}} references from a template string
// and returns those whose values are empty or missing in the flux map.
func missingTemplateDeps(tmplStr string, flux map[string]any) []string {
	var missing []string
	// Find all {{.some.path}} or {{ .some.path }} references
	// We scan for ".identifier" sequences inside {{ }} delimiters
	rest := tmplStr
	for {
		start := strings.Index(rest, "{{")
		if start < 0 {
			break
		}
		end := strings.Index(rest[start:], "}}")
		if end < 0 {
			break
		}
		expr := rest[start+2 : start+end]
		rest = rest[start+end+2:]

		// Look for dotted field access like .project.organization
		expr = strings.TrimSpace(expr)
		// Skip range, if, end, etc. — only look for standalone field refs
		// and field refs used as arguments (e.g., -f org='{{.project.organization}}')
		refs := extractDottedRefs(expr)
		for _, ref := range refs {
			if ref == "" {
				continue
			}
			val := lookupNestedString(flux, ref)
			if val == "" {
				missing = append(missing, ref)
			}
		}
	}
	return dedupStrings(missing)
}

// extractDottedRefs finds all .dotted.path references in a template expression.
func extractDottedRefs(expr string) []string {
	var refs []string
	for i := 0; i < len(expr); i++ {
		if expr[i] == '.' && (i == 0 || expr[i-1] == ' ' || expr[i-1] == '=' || expr[i-1] == '\'' || expr[i-1] == '"' || expr[i-1] == '(') {
			// Collect the dotted path: .word.word.word
			j := i + 1
			for j < len(expr) {
				ch := expr[j]
				if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '.' {
					j++
				} else {
					break
				}
			}
			ref := expr[i+1 : j]
			// Must have at least one dot to be a nested reference
			if strings.Contains(ref, ".") {
				refs = append(refs, ref)
			}
		}
	}
	return refs
}

// lookupNestedString traverses a nested map by dotted path and returns the string value.
func lookupNestedString(m map[string]any, path string) string {
	parts := strings.Split(path, ".")
	var current any = m
	for _, part := range parts {
		cm, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current, ok = cm[part]
		if !ok {
			return ""
		}
	}
	if s, ok := current.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", current)
}

// dedupStrings removes duplicate strings from a slice.
func dedupStrings(ss []string) []string {
	seen := make(map[string]bool, len(ss))
	var result []string
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// fluxDeps returns a binding value for huh's reactive OptionsFunc.
// When the binding's hash changes, huh re-evaluates the options function.
// Only includes pointers for variables actually referenced in the discover
// command template, so unrelated changes don't trigger re-queries.
func (w *dynamicWizard) fluxDeps(fv mold.FluxVar) any {
	if fv.Discover == nil {
		return nil
	}
	// Extract the specific template variable references from the command
	refs := discoverCommandRefs(fv.Discover.Command)
	if len(refs) == 0 {
		// No template deps (e.g., org query uses viewer API) — use a static binding
		return "static"
	}
	// Build a slice of only the referenced bound value pointers
	var deps []any
	for _, ref := range refs {
		if ptr, ok := w.values[ref]; ok {
			deps = append(deps, ptr)
		}
		if ptr, ok := w.boolVals[ref]; ok {
			deps = append(deps, ptr)
		}
		if ptr, ok := w.textVals[ref]; ok {
			deps = append(deps, ptr)
		}
	}
	if len(deps) == 0 {
		return "static"
	}
	return deps
}

// discoverCommandRefs extracts the dotted flux variable references from a
// discover command template (e.g., "{{.project.organization}}" -> ["project.organization"]).
func discoverCommandRefs(tmplStr string) []string {
	var refs []string
	rest := tmplStr
	for {
		start := strings.Index(rest, "{{")
		if start < 0 {
			break
		}
		end := strings.Index(rest[start:], "}}")
		if end < 0 {
			break
		}
		expr := strings.TrimSpace(rest[start+2 : start+end])
		rest = rest[start+end+2:]
		for _, ref := range extractDottedRefs(expr) {
			if ref != "" {
				refs = append(refs, ref)
			}
		}
	}
	return dedupStrings(refs)
}

// currentFlux builds a flux map from the current bound values.
func (w *dynamicWizard) currentFlux() map[string]any {
	flux := make(map[string]any)
	// Start with original flux as base
	for k, v := range w.flux {
		flux[k] = v
	}
	// Override with current bound values
	for name, ptr := range w.values {
		if ptr != nil && *ptr != "" {
			mold.SetNestedValue(flux, name, *ptr)
		}
	}
	for name, ptr := range w.boolVals {
		if ptr != nil {
			mold.SetNestedAny(flux, name, *ptr)
		}
	}
	for name, ptr := range w.textVals {
		if ptr != nil && *ptr != "" {
			mold.SetNestedValue(flux, name, *ptr)
		}
	}
	// Apply also_sets: propagate extra segments from discover results
	w.applyAlsoSets(flux)
	return flux
}

// applyAlsoSets looks up discover results for fields with also_sets and
// populates the extra flux variables from the selected option's extra segments.
func (w *dynamicWizard) applyAlsoSets(flux map[string]any) {
	for _, fv := range w.schema {
		if fv.Discover == nil || len(fv.Discover.AlsoSets) == 0 {
			continue
		}
		// Get the currently selected value for this field
		ptr, ok := w.values[fv.Name]
		if !ok || ptr == nil || *ptr == "" {
			continue
		}
		selected := *ptr

		// Find the matching discover result
		results := w.discoverResults[fv.Name]
		for _, r := range results {
			if r.Value == selected {
				// Apply each also_sets mapping
				for varName, idx := range fv.Discover.AlsoSets {
					// idx is 0-based into Extra (segment 2+ from parse output)
					if idx >= 0 && idx < len(r.Extra) {
						mold.SetNestedValue(flux, varName, r.Extra[idx])
					}
				}
				break
			}
		}
	}
}

// buildSummary creates a dynamic review preview from all bound values.
func (w *dynamicWizard) buildSummary() string {
	var b strings.Builder
	b.WriteString("Flux values to write:\n\n")

	// Build current flux to include also_sets values
	flux := w.currentFlux()

	for _, fv := range w.schema {
		val := w.getBoundValue(fv)
		if val != "" {
			fmt.Fprintf(&b, "  %s: %s\n", fv.Name, val)
		}

		// Show also_sets values derived from this field's discover selection
		if fv.Discover != nil {
			for varName := range fv.Discover.AlsoSets {
				if v, ok := mold.GetNestedAny(flux, varName); ok {
					if s, ok := v.(string); ok && s != "" {
						fmt.Fprintf(&b, "  %s: %s\n", varName, s)
					}
				}
			}
		}
	}

	return b.String()
}

// getBoundValue returns the current string representation of a bound value.
func (w *dynamicWizard) getBoundValue(fv mold.FluxVar) string {
	switch fv.Type {
	case "bool":
		if ptr, ok := w.boolVals[fv.Name]; ok && ptr != nil {
			return fmt.Sprintf("%v", *ptr)
		}
	case "list":
		if ptr, ok := w.textVals[fv.Name]; ok && ptr != nil {
			return *ptr
		}
	default:
		if ptr, ok := w.values[fv.Name]; ok && ptr != nil {
			return *ptr
		}
	}
	return ""
}

// run executes the full wizard form and returns the populated flux map.
func (w *dynamicWizard) run() (map[string]any, bool, error) {
	if len(w.schema) == 0 {
		return w.flux, false, fmt.Errorf("no flux variables found in schema")
	}

	// Welcome banner
	fmt.Println(styles.WorkingBanner("Interactive blank annealing"))
	fmt.Println()

	// Build schema-driven groups
	huhGroups := w.buildGroups()

	// Add review & save section
	var confirmSave bool
	reviewGroup := huh.NewGroup(
		huh.NewNote().
			Title("Review Changes").
			DescriptionFunc(func() string {
				return w.buildSummary()
			}, w.allDeps()).
			Next(true).
			NextLabel("Continue"),
		huh.NewConfirm().
			Title("Save these flux values?").
			Description("Save writes to flux file; Cancel prints to stdout").
			Affirmative("Save").
			Negative("Cancel").
			Value(&confirmSave),
	).Title("Review & Save")

	allGroups := append(huhGroups, reviewGroup)

	form := huh.NewForm(allGroups...).WithTheme(ailloyTheme())

	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Println(styles.InfoStyle.Render("Annealing cancelled."))
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("wizard error: %w", err)
	}

	// Collect values into flux map
	result := w.currentFlux()

	if !confirmSave {
		return result, false, nil
	}

	return result, true, nil
}

// allDeps returns a binding value for reactive huh updates.
func (w *dynamicWizard) allDeps() any {
	var deps []any
	for _, ptr := range w.values {
		deps = append(deps, ptr)
	}
	for _, ptr := range w.boolVals {
		deps = append(deps, ptr)
	}
	for _, ptr := range w.textVals {
		deps = append(deps, ptr)
	}
	return deps
}

// fieldTitle returns the display title for a flux variable.
// Uses the last segment of the dotted name, title-cased.
func fieldTitle(fv mold.FluxVar) string {
	name := fv.Name
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		name = name[idx+1:]
	}
	// Convert underscores to spaces and title-case
	name = strings.ReplaceAll(name, "_", " ")
	if len(name) > 0 {
		name = strings.ToUpper(name[:1]) + name[1:]
	}
	return name
}
