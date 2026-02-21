package commands

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/nimble-giant/ailloy/pkg/config"
	"github.com/nimble-giant/ailloy/pkg/github"
	"github.com/nimble-giant/ailloy/pkg/styles"
)

// runWizardAnneal runs the 5-section huh wizard for interactive annealing
func runWizardAnneal(cfg *config.Config) error {
	scope := "project"
	if globalCustomize {
		scope = "global"
	}

	// Snapshot original config for diff summary
	origVars := snapshotVars(cfg)
	origOre := snapshotOre(cfg)

	// Wizard state
	var (
		projectName string
		orgName     string

		enableGitHub  bool
		selectedBoard string // "number:id" format for select value

		enabledOreModels []string

		statusField    string // field ID
		priorityField  string // field ID
		iterationField string // field ID

		customVarsRaw string

		confirmSave bool
	)

	// Pre-populate from existing config
	if v, ok := cfg.Templates.Flux["default_board"].(string); ok {
		projectName = v
	}
	if v, ok := cfg.Templates.Flux["organization"].(string); ok {
		orgName = v
	}

	// Pre-populate enabled ore models
	if cfg.Ore.Status.Enabled {
		enabledOreModels = append(enabledOreModels, "status")
	}
	if cfg.Ore.Priority.Enabled {
		enabledOreModels = append(enabledOreModels, "priority")
	}
	if cfg.Ore.Iteration.Enabled {
		enabledOreModels = append(enabledOreModels, "iteration")
	}

	// Pre-populate field IDs
	statusField = cfg.Ore.Status.FieldID
	priorityField = cfg.Ore.Priority.FieldID
	iterationField = cfg.Ore.Iteration.FieldID

	// Pre-populate custom vars
	customVarsRaw = buildCustomVarsText(cfg)

	// GitHub discovery client (lazy, cached)
	ghClient := github.NewClient()

	// Welcome banner
	fmt.Println(styles.WorkingBanner("Interactive template annealing (" + scope + ")"))
	fmt.Println()

	// --- Section 1: Project Basics ---
	group1 := huh.NewGroup(
		huh.NewInput().
			Title("Project board name").
			Description("Default GitHub project board name").
			Placeholder("e.g., Engineering").
			Value(&projectName),
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

	// Field mapping for Status — hidden unless status is enabled AND GitHub integration is on
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
			Description("One per line, format: key=value. Leave empty to skip.").
			Placeholder("default_reviewer=alice\nslack_channel=#eng").
			Value(&customVarsRaw).
			Lines(6),
	).Title("Section 4: Custom Flux Variables").
		Description("Add freeform key-value pairs for template rendering")

	// --- Section 5: Review & Save ---
	group5 := huh.NewGroup(
		huh.NewNote().
			Title("Review Changes").
			DescriptionFunc(func() string {
				return buildSummaryDiff(
					origVars, origOre,
					projectName, orgName,
					enabledOreModels,
					statusField, priorityField, iterationField,
					customVarsRaw,
				)
			}, &customVarsRaw).
			Next(true).
			NextLabel("Continue"),
		huh.NewConfirm().
			Title("Save these changes?").
			Description("Write configuration to "+scope+" config file").
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

	// Apply wizard results to config
	applyWizardResults(cfg, projectName, orgName, enabledOreModels,
		statusField, priorityField, iterationField,
		customVarsRaw, enableGitHub, selectedBoard, ghClient)

	if err := config.SaveConfig(cfg, globalCustomize); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println()
	fmt.Println(styles.SuccessBanner("Template annealing saved to " + scope + " configuration"))
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

	// Try smart matching to determine which to pre-select
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

// applyWizardResults applies wizard state to the config
func applyWizardResults(
	cfg *config.Config,
	projectName, orgName string,
	enabledOreModels []string,
	statusFieldID, priorityFieldID, iterationFieldID string,
	customVarsRaw string,
	enableGitHub bool,
	selectedBoard string,
	ghClient *github.Client,
) {
	// Basic flux variables
	if projectName != "" {
		cfg.Templates.Flux["default_board"] = projectName
	}
	if orgName != "" {
		cfg.Templates.Flux["organization"] = orgName
	}

	// Store project ID if board was selected
	if enableGitHub && selectedBoard != "" {
		boardID := parseBoardID(selectedBoard)
		if boardID != "" {
			cfg.Templates.Flux["project_id"] = boardID
		}
	}

	// Ore configuration
	cfg.Ore.Status.Enabled = contains(enabledOreModels, "status")
	cfg.Ore.Priority.Enabled = contains(enabledOreModels, "priority")
	cfg.Ore.Iteration.Enabled = contains(enabledOreModels, "iteration")

	// Field mapping
	if statusFieldID != "" {
		cfg.Ore.Status.FieldID = statusFieldID
		applyFieldMapping(cfg, ghClient, orgName, selectedBoard, &cfg.Ore.Status, statusFieldID)
	}
	if priorityFieldID != "" {
		cfg.Ore.Priority.FieldID = priorityFieldID
		applyFieldMapping(cfg, ghClient, orgName, selectedBoard, &cfg.Ore.Priority, priorityFieldID)
	}
	if iterationFieldID != "" {
		cfg.Ore.Iteration.FieldID = iterationFieldID
		applyFieldMapping(cfg, ghClient, orgName, selectedBoard, &cfg.Ore.Iteration, iterationFieldID)
	}

	// Custom flux variables
	customVars := parseCustomVars(customVarsRaw)
	for k, v := range customVars {
		cfg.Templates.Flux[k] = v
	}
}

// applyFieldMapping looks up the field name and auto-maps options
func applyFieldMapping(cfg *config.Config, ghClient *github.Client, org, boardValue string, ore *config.OreConfig, fieldID string) {
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
			ore.FieldMapping = f.Name

			// Auto-map options by label matching
			if ore.Options != nil {
				for conceptKey, opt := range ore.Options {
					if matched := github.MatchOptionByName(f.Options, opt.Label); matched != nil {
						opt.ID = matched.ID
						ore.Options[conceptKey] = opt
					}
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

// buildCustomVarsText converts existing custom vars to "key=value\n" text,
// excluding well-known variables that are managed by the wizard
func buildCustomVarsText(cfg *config.Config) string {
	managed := map[string]bool{
		"default_board":      true,
		"default_priority":   true,
		"default_status":     true,
		"organization":       true,
		"project_id":         true,
		"status_field_id":    true,
		"priority_field_id":  true,
		"iteration_field_id": true,
	}

	var lines []string
	for k, v := range cfg.Templates.Flux {
		if managed[k] {
			continue
		}
		if s, ok := v.(string); ok {
			lines = append(lines, k+"="+s)
		}
	}
	return strings.Join(lines, "\n")
}

// snapshotVars captures current flux variable state for diff comparison
func snapshotVars(cfg *config.Config) map[string]string {
	snap := make(map[string]string, len(cfg.Templates.Flux))
	for k, v := range cfg.Templates.Flux {
		if s, ok := v.(string); ok {
			snap[k] = s
		}
	}
	return snap
}

type oreSnapshot struct {
	StatusEnabled    bool
	PriorityEnabled  bool
	IterationEnabled bool
	StatusFieldID    string
	PriorityFieldID  string
	IterationFieldID string
}

func snapshotOre(cfg *config.Config) oreSnapshot {
	return oreSnapshot{
		StatusEnabled:    cfg.Ore.Status.Enabled,
		PriorityEnabled:  cfg.Ore.Priority.Enabled,
		IterationEnabled: cfg.Ore.Iteration.Enabled,
		StatusFieldID:    cfg.Ore.Status.FieldID,
		PriorityFieldID:  cfg.Ore.Priority.FieldID,
		IterationFieldID: cfg.Ore.Iteration.FieldID,
	}
}

// buildSummaryDiff creates a styled diff of changes for the review section
func buildSummaryDiff(
	origVars map[string]string,
	origOre oreSnapshot,
	projectName, orgName string,
	enabledOreModels []string,
	statusFieldID, priorityFieldID, iterationFieldID string,
	customVarsRaw string,
) string {
	var b strings.Builder

	b.WriteString("Changes to apply:\n\n")

	// Basic variables
	diffVar(&b, "default_board", origVars["default_board"], projectName)
	diffVar(&b, "organization", origVars["organization"], orgName)

	b.WriteString("\n")

	// Ore models
	diffBool(&b, "Status ore", origOre.StatusEnabled, contains(enabledOreModels, "status"))
	diffBool(&b, "Priority ore", origOre.PriorityEnabled, contains(enabledOreModels, "priority"))
	diffBool(&b, "Iteration ore", origOre.IterationEnabled, contains(enabledOreModels, "iteration"))

	// Field IDs
	if statusFieldID != "" || origOre.StatusFieldID != "" {
		diffVar(&b, "Status field ID", origOre.StatusFieldID, statusFieldID)
	}
	if priorityFieldID != "" || origOre.PriorityFieldID != "" {
		diffVar(&b, "Priority field ID", origOre.PriorityFieldID, priorityFieldID)
	}
	if iterationFieldID != "" || origOre.IterationFieldID != "" {
		diffVar(&b, "Iteration field ID", origOre.IterationFieldID, iterationFieldID)
	}

	// Custom variables
	customVars := parseCustomVars(customVarsRaw)
	if len(customVars) > 0 {
		b.WriteString("\nCustom variables:\n")
		for k, v := range customVars {
			b.WriteString(fmt.Sprintf("  %s = %s\n", k, v))
		}
	}

	return b.String()
}

func diffVar(b *strings.Builder, name, old, new string) {
	if old == new {
		if old != "" {
			fmt.Fprintf(b, "  %s: %s (unchanged)\n", name, old)
		}
		return
	}
	if old == "" {
		fmt.Fprintf(b, "  + %s: %s\n", name, new)
	} else if new == "" {
		fmt.Fprintf(b, "  - %s: %s\n", name, old)
	} else {
		fmt.Fprintf(b, "  ~ %s: %s -> %s\n", name, old, new)
	}
}

func diffBool(b *strings.Builder, name string, old, new bool) {
	if old == new {
		state := "disabled"
		if old {
			state = "enabled"
		}
		fmt.Fprintf(b, "  %s: %s (unchanged)\n", name, state)
		return
	}
	if new {
		fmt.Fprintf(b, "  + %s: enabled\n", name)
	} else {
		fmt.Fprintf(b, "  - %s: disabled\n", name)
	}
}

// --- helpers ---

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func parseBoardNumber(boardValue string) int {
	// boardValue format is "number:id"
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
