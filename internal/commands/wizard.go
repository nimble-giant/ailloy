package commands

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/nimble-giant/ailloy/pkg/github"
	"github.com/nimble-giant/ailloy/pkg/mold"
	"github.com/nimble-giant/ailloy/pkg/styles"
)

// runWizardAnneal runs the 5-section huh wizard for interactive annealing.
// It populates the given flux map and writes the output via writeFluxOutput.
func runWizardAnneal(flux map[string]any) error {
	// Wizard state
	var (
		projectBoard string
		orgName      string

		enableGitHub  bool
		selectedBoard string // "number:id" format for select value

		enabledOreModels []string

		statusField    string // field ID
		priorityField  string // field ID
		iterationField string // field ID

		customVarsRaw string

		confirmSave bool
	)

	// Pre-populate from existing flux if present
	if v, ok := getFluxString(flux, "project.board"); ok {
		projectBoard = v
	}
	if v, ok := getFluxString(flux, "project.organization"); ok {
		orgName = v
	}

	// Pre-populate enabled ore models
	if enabled, ok := getFluxBool(flux, "ore.status.enabled"); ok && enabled {
		enabledOreModels = append(enabledOreModels, "status")
	}
	if enabled, ok := getFluxBool(flux, "ore.priority.enabled"); ok && enabled {
		enabledOreModels = append(enabledOreModels, "priority")
	}
	if enabled, ok := getFluxBool(flux, "ore.iteration.enabled"); ok && enabled {
		enabledOreModels = append(enabledOreModels, "iteration")
	}

	// Pre-populate field IDs
	if v, ok := getFluxString(flux, "ore.status.field_id"); ok {
		statusField = v
	}
	if v, ok := getFluxString(flux, "ore.priority.field_id"); ok {
		priorityField = v
	}
	if v, ok := getFluxString(flux, "ore.iteration.field_id"); ok {
		iterationField = v
	}

	// GitHub discovery client (lazy, cached)
	ghClient := github.NewClient()

	// Welcome banner
	fmt.Println(styles.WorkingBanner("Interactive blank annealing"))
	fmt.Println()

	// --- Section 1: Project Basics ---
	group1 := huh.NewGroup(
		huh.NewInput().
			Title("Project board name").
			Description("Default GitHub project board name").
			Placeholder("e.g., Engineering").
			Value(&projectBoard),
		huh.NewInput().
			Title("Organization").
			Description("GitHub organization name").
			Placeholder("e.g., mycompany").
			Value(&orgName),
	).Title("Section 1: Project Basics").
		Description("Basic project configuration")

	// --- Section 2: GitHub Integration ---
	group2 := huh.NewGroup(
		huh.NewConfirm().
			Title("Enable GitHub Project integration?").
			Description("Auto-discover boards, fields, and options via the GitHub API").
			Affirmative("Yes").
			Negative("No").
			Value(&enableGitHub),
	).Title("Section 2: GitHub Integration").
		Description("Connect to GitHub Projects for automatic field discovery")

	// Board selection — hidden unless GitHub is enabled
	group2board := huh.NewGroup(
		huh.NewSelect[string]().
			Title("Select a project board").
			Description("Discovered from your organization's GitHub Projects").
			OptionsFunc(func() []huh.Option[string] {
				return discoverBoards(ghClient, orgName)
			}, &orgName).
			Value(&selectedBoard).
			Height(10),
	).WithHideFunc(func() bool {
		return !enableGitHub || orgName == ""
	})

	// --- Section 3: Ore Configuration ---
	group3 := huh.NewGroup(
		huh.NewMultiSelect[string]().
			Title("Which ore models to enable?").
			Description("Ore models map your workflow concepts to GitHub Project fields").
			Options(
				huh.NewOption("Status (track issue lifecycle)", "status").Selected(contains(enabledOreModels, "status")),
				huh.NewOption("Priority (P0-P3 ranking)", "priority").Selected(contains(enabledOreModels, "priority")),
				huh.NewOption("Iteration (sprint/cycle tracking)", "iteration").Selected(contains(enabledOreModels, "iteration")),
			).
			Value(&enabledOreModels),
	).Title("Section 3: Ore Configuration").
		Description("Configure semantic ore models for your workflow")

	// Field mapping for Status
	group3status := huh.NewGroup(
		huh.NewSelect[string]().
			Title("Map Status ore to GitHub field").
			Description("Select which GitHub Project field represents status").
			OptionsFunc(func() []huh.Option[string] {
				return discoverFieldOptions(ghClient, orgName, selectedBoard, "Status")
			}, &selectedBoard).
			Value(&statusField).
			Height(8),
	).WithHideFunc(func() bool {
		return !enableGitHub || !contains(enabledOreModels, "status") || selectedBoard == ""
	})

	// Field mapping for Priority
	group3priority := huh.NewGroup(
		huh.NewSelect[string]().
			Title("Map Priority ore to GitHub field").
			Description("Select which GitHub Project field represents priority").
			OptionsFunc(func() []huh.Option[string] {
				return discoverFieldOptions(ghClient, orgName, selectedBoard, "Priority")
			}, &selectedBoard).
			Value(&priorityField).
			Height(8),
	).WithHideFunc(func() bool {
		return !enableGitHub || !contains(enabledOreModels, "priority") || selectedBoard == ""
	})

	// Field mapping for Iteration
	group3iteration := huh.NewGroup(
		huh.NewSelect[string]().
			Title("Map Iteration ore to GitHub field").
			Description("Select which GitHub Project field represents iteration/sprint").
			OptionsFunc(func() []huh.Option[string] {
				return discoverFieldOptions(ghClient, orgName, selectedBoard, "Iteration")
			}, &selectedBoard).
			Value(&iterationField).
			Height(8),
	).WithHideFunc(func() bool {
		return !enableGitHub || !contains(enabledOreModels, "iteration") || selectedBoard == ""
	})

	// --- Section 4: Custom Flux Variables ---
	group4 := huh.NewGroup(
		huh.NewText().
			Title("Custom flux variables").
			Description("One per line, format: key=value (supports dotted paths like scm.provider=GitLab)").
			Placeholder("default_reviewer=alice\nslack_channel=#eng").
			Value(&customVarsRaw).
			Lines(6),
	).Title("Section 4: Custom Flux Variables").
		Description("Add freeform key-value pairs for blank rendering")

	// --- Section 5: Review & Save ---
	group5 := huh.NewGroup(
		huh.NewNote().
			Title("Review Changes").
			DescriptionFunc(func() string {
				return buildSummaryPreview(
					projectBoard, orgName,
					enabledOreModels,
					statusField, priorityField, iterationField,
					customVarsRaw,
				)
			}, &customVarsRaw).
			Next(true).
			NextLabel("Continue"),
		huh.NewConfirm().
			Title("Save these flux values?").
			Description("Write flux YAML to output").
			Affirmative("Save").
			Negative("Cancel").
			Value(&confirmSave),
	).Title("Section 5: Review & Save")

	form := huh.NewForm(
		group1,
		group2, group2board,
		group3, group3status, group3priority, group3iteration,
		group4,
		group5,
	).WithTheme(ailloyTheme())

	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Println(styles.InfoStyle.Render("Annealing cancelled."))
			return nil
		}
		return fmt.Errorf("wizard error: %w", err)
	}

	if !confirmSave {
		fmt.Println(styles.InfoStyle.Render("Changes discarded."))
		return nil
	}

	// Apply wizard results to flux map
	applyWizardResults(flux, projectBoard, orgName, enabledOreModels,
		statusField, priorityField, iterationField,
		customVarsRaw, enableGitHub, selectedBoard, ghClient)

	if err := writeFluxOutput(flux); err != nil {
		return err
	}

	fmt.Println()
	dest := "stdout"
	if annealOutput != "" {
		dest = annealOutput
	}
	fmt.Println(styles.SuccessBanner("Blank annealing saved to " + dest))
	return nil
}

// discoverBoards queries GitHub for project boards and returns huh options
func discoverBoards(client *github.Client, org string) []huh.Option[string] {
	if org == "" {
		return []huh.Option[string]{huh.NewOption("(enter organization name first)", "")}
	}

	if err := client.CheckAuth(); err != nil {
		return []huh.Option[string]{huh.NewOption("("+err.Error()+")", "")}
	}

	projects, err := client.ListProjects(org)
	if err != nil {
		return []huh.Option[string]{huh.NewOption("("+err.Error()+")", "")}
	}

	opts := make([]huh.Option[string], 0, len(projects))
	for _, p := range projects {
		if p.Closed {
			continue
		}
		label := fmt.Sprintf("%s (#%d)", p.Title, p.Number)
		value := fmt.Sprintf("%d:%s", p.Number, p.ID)
		opts = append(opts, huh.NewOption(label, value))
	}

	if len(opts) == 0 {
		return []huh.Option[string]{huh.NewOption("(no open projects found)", "")}
	}
	return opts
}

// discoverFieldOptions queries GitHub for project fields and returns huh options
func discoverFieldOptions(client *github.Client, org, boardValue, oreName string) []huh.Option[string] {
	if boardValue == "" {
		return []huh.Option[string]{huh.NewOption("(select a board first)", "")}
	}

	projectNumber := parseBoardNumber(boardValue)
	if projectNumber == 0 {
		return []huh.Option[string]{huh.NewOption("(invalid board selection)", "")}
	}

	result, err := client.GetProjectFields(org, projectNumber)
	if err != nil {
		return []huh.Option[string]{huh.NewOption("("+err.Error()+")", "")}
	}

	// Filter to relevant field types
	var relevantTypes []github.FieldType
	switch strings.ToLower(oreName) {
	case "iteration":
		relevantTypes = []github.FieldType{github.FieldTypeIteration, github.FieldTypeSingleSelect}
	default:
		relevantTypes = []github.FieldType{github.FieldTypeSingleSelect}
	}

	opts := []huh.Option[string]{huh.NewOption("(skip - don't map this ore)", "")}

	smartMatch := github.MatchFieldByName(result.Fields, oreName)

	for _, f := range result.Fields {
		if !isRelevantType(f.Type, relevantTypes) {
			continue
		}
		label := fmt.Sprintf("%s (%s)", f.Name, f.Type)
		if smartMatch != nil && f.ID == smartMatch.ID {
			label += " *suggested*"
		}
		opts = append(opts, huh.NewOption(label, f.ID))
	}

	return opts
}

// applyWizardResults applies wizard state to the flux map
func applyWizardResults(
	flux map[string]any,
	projectBoard, orgName string,
	enabledOreModels []string,
	statusFieldID, priorityFieldID, iterationFieldID string,
	customVarsRaw string,
	enableGitHub bool,
	selectedBoard string,
	ghClient *github.Client,
) {
	// Basic flux variables
	if projectBoard != "" {
		mold.SetNestedAny(flux, "project.board", projectBoard)
	}
	if orgName != "" {
		mold.SetNestedAny(flux, "project.organization", orgName)
	}

	// Store project ID if board was selected
	if enableGitHub && selectedBoard != "" {
		boardID := parseBoardID(selectedBoard)
		if boardID != "" {
			mold.SetNestedAny(flux, "project.id", boardID)
		}
	}

	// Ore configuration
	mold.SetNestedAny(flux, "ore.status.enabled", contains(enabledOreModels, "status"))
	mold.SetNestedAny(flux, "ore.priority.enabled", contains(enabledOreModels, "priority"))
	mold.SetNestedAny(flux, "ore.iteration.enabled", contains(enabledOreModels, "iteration"))

	// Field mapping
	if statusFieldID != "" {
		mold.SetNestedAny(flux, "ore.status.field_id", statusFieldID)
		applyFieldMapping(flux, ghClient, orgName, selectedBoard, "ore.status", statusFieldID)
	}
	if priorityFieldID != "" {
		mold.SetNestedAny(flux, "ore.priority.field_id", priorityFieldID)
		applyFieldMapping(flux, ghClient, orgName, selectedBoard, "ore.priority", priorityFieldID)
	}
	if iterationFieldID != "" {
		mold.SetNestedAny(flux, "ore.iteration.field_id", iterationFieldID)
		applyFieldMapping(flux, ghClient, orgName, selectedBoard, "ore.iteration", iterationFieldID)
	}

	// Custom flux variables
	customVars := parseCustomVars(customVarsRaw)
	for k, v := range customVars {
		mold.SetNestedValue(flux, k, v)
	}
}

// applyFieldMapping looks up the field name and auto-maps options using dotted paths.
func applyFieldMapping(flux map[string]any, ghClient *github.Client, org, boardValue, orePrefix, fieldID string) {
	if boardValue == "" || org == "" {
		return
	}
	projectNumber := parseBoardNumber(boardValue)
	if projectNumber == 0 {
		return
	}

	result, err := ghClient.GetProjectFields(org, projectNumber)
	if err != nil {
		return
	}

	// Find the field by ID and set the mapping name
	for _, f := range result.Fields {
		if f.ID == fieldID {
			mold.SetNestedAny(flux, orePrefix+".field_mapping", f.Name)

			// Auto-map options by looking at existing option keys in flux
			optionsVal, hasOptions := mold.GetNestedAny(flux, orePrefix+".options")
			if hasOptions {
				if optionsMap, ok := optionsVal.(map[string]any); ok {
					for conceptKey, optVal := range optionsMap {
						if optMap, ok := optVal.(map[string]any); ok {
							label, _ := optMap["label"].(string)
							if label == "" {
								continue
							}
							if matched := github.MatchOptionByName(f.Options, label); matched != nil {
								mold.SetNestedAny(flux, orePrefix+".options."+conceptKey+".id", matched.ID)
							}
						}
					}
				}
			} else {
				// No existing options in flux — map directly from field options
				for _, opt := range f.Options {
					key := strings.ToLower(strings.ReplaceAll(opt.Name, " ", "_"))
					mold.SetNestedAny(flux, orePrefix+".options."+key+".label", opt.Name)
					mold.SetNestedAny(flux, orePrefix+".options."+key+".id", opt.ID)
				}
			}
			break
		}
	}
}

// parseCustomVars parses "key=value\n" text into a map
func parseCustomVars(raw string) map[string]string {
	vars := make(map[string]string)
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key != "" {
			vars[key] = value
		}
	}
	return vars
}

// buildSummaryPreview creates a styled preview for the review section
func buildSummaryPreview(
	projectBoard, orgName string,
	enabledOreModels []string,
	statusFieldID, priorityFieldID, iterationFieldID string,
	customVarsRaw string,
) string {
	var b strings.Builder

	b.WriteString("Flux values to write:\n\n")

	if projectBoard != "" {
		fmt.Fprintf(&b, "  project.board: %s\n", projectBoard)
	}
	if orgName != "" {
		fmt.Fprintf(&b, "  project.organization: %s\n", orgName)
	}

	b.WriteString("\n")

	// Ore models
	for _, model := range []string{"status", "priority", "iteration"} {
		enabled := contains(enabledOreModels, model)
		state := "disabled"
		if enabled {
			state = "enabled"
		}
		fmt.Fprintf(&b, "  ore.%s.enabled: %s\n", model, state)
	}

	// Field IDs
	if statusFieldID != "" {
		fmt.Fprintf(&b, "  ore.status.field_id: %s\n", statusFieldID)
	}
	if priorityFieldID != "" {
		fmt.Fprintf(&b, "  ore.priority.field_id: %s\n", priorityFieldID)
	}
	if iterationFieldID != "" {
		fmt.Fprintf(&b, "  ore.iteration.field_id: %s\n", iterationFieldID)
	}

	// Custom variables
	customVars := parseCustomVars(customVarsRaw)
	if len(customVars) > 0 {
		b.WriteString("\nCustom variables:\n")
		for k, v := range customVars {
			fmt.Fprintf(&b, "  %s: %s\n", k, v)
		}
	}

	return b.String()
}

// --- helpers ---

// getFluxString retrieves a string value from nested flux by dotted path.
func getFluxString(flux map[string]any, path string) (string, bool) {
	val, ok := mold.GetNestedAny(flux, path)
	if !ok {
		return "", false
	}
	s, ok := val.(string)
	return s, ok
}

// getFluxBool retrieves a bool value from nested flux by dotted path.
func getFluxBool(flux map[string]any, path string) (bool, bool) {
	val, ok := mold.GetNestedAny(flux, path)
	if !ok {
		return false, false
	}
	b, ok := val.(bool)
	return b, ok
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func parseBoardNumber(boardValue string) int {
	parts := strings.SplitN(boardValue, ":", 2)
	if len(parts) < 1 {
		return 0
	}
	n, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0
	}
	return n
}

func parseBoardID(boardValue string) string {
	parts := strings.SplitN(boardValue, ":", 2)
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}

func isRelevantType(ft github.FieldType, relevant []github.FieldType) bool {
	for _, r := range relevant {
		if ft == r {
			return true
		}
	}
	return false
}
