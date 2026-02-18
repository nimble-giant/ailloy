package github

import (
	"testing"
)

func TestMatchFieldByName_ExactCaseInsensitive(t *testing.T) {
	fields := []Field{
		{ID: "f1", Name: "Status", Type: FieldTypeSingleSelect},
		{ID: "f2", Name: "Priority", Type: FieldTypeSingleSelect},
	}

	got := MatchFieldByName(fields, "status")
	if got == nil {
		t.Fatal("expected match, got nil")
	}
	if got.ID != "f1" {
		t.Errorf("expected f1, got %s", got.ID)
	}
}

func TestMatchFieldByName_ExactMatch(t *testing.T) {
	fields := []Field{
		{ID: "f1", Name: "Status", Type: FieldTypeSingleSelect},
	}

	got := MatchFieldByName(fields, "Status")
	if got == nil {
		t.Fatal("expected match, got nil")
	}
	if got.ID != "f1" {
		t.Errorf("expected f1, got %s", got.ID)
	}
}

func TestMatchFieldByName_FuzzySubstring(t *testing.T) {
	fields := []Field{
		{ID: "f1", Name: "Team Status Board", Type: FieldTypeSingleSelect},
		{ID: "f2", Name: "Priority", Type: FieldTypeSingleSelect},
	}

	got := MatchFieldByName(fields, "status")
	if got == nil {
		t.Fatal("expected fuzzy match, got nil")
	}
	if got.ID != "f1" {
		t.Errorf("expected f1, got %s", got.ID)
	}
}

func TestMatchFieldByName_NoMatch(t *testing.T) {
	fields := []Field{
		{ID: "f1", Name: "Status", Type: FieldTypeSingleSelect},
	}

	got := MatchFieldByName(fields, "iteration")
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestMatchFieldByName_PrefersExactOverFuzzy(t *testing.T) {
	fields := []Field{
		{ID: "f1", Name: "Team Status", Type: FieldTypeSingleSelect},
		{ID: "f2", Name: "Status", Type: FieldTypeSingleSelect},
	}

	got := MatchFieldByName(fields, "Status")
	if got == nil {
		t.Fatal("expected match, got nil")
	}
	if got.ID != "f2" {
		t.Errorf("expected exact match f2, got %s", got.ID)
	}
}

func TestMatchOptionByName_ExactCaseInsensitive(t *testing.T) {
	options := []Option{
		{ID: "o1", Name: "In Progress"},
		{ID: "o2", Name: "Done"},
	}

	got := MatchOptionByName(options, "in progress")
	if got == nil {
		t.Fatal("expected match, got nil")
	}
	if got.ID != "o1" {
		t.Errorf("expected o1, got %s", got.ID)
	}
}

func TestMatchOptionByName_NoMatch(t *testing.T) {
	options := []Option{
		{ID: "o1", Name: "In Progress"},
	}

	got := MatchOptionByName(options, "Ready")
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestAutoMapModel_FullMatch(t *testing.T) {
	fields := []Field{
		{
			ID:   "f1",
			Name: "Status",
			Type: FieldTypeSingleSelect,
			Options: []Option{
				{ID: "o1", Name: "Ready"},
				{ID: "o2", Name: "In Progress"},
				{ID: "o3", Name: "Done"},
			},
		},
		{ID: "f2", Name: "Title", Type: FieldTypeText},
	}

	modelOptions := map[string]string{
		"ready":       "Ready",
		"in_progress": "In Progress",
		"done":        "Done",
		"in_review":   "In Review", // no GitHub match
	}

	field, opts := AutoMapModel(fields, "Status", modelOptions)

	if field == nil {
		t.Fatal("expected field match, got nil")
	}
	if field.ID != "f1" {
		t.Errorf("expected f1, got %s", field.ID)
	}

	if len(opts) != 3 {
		t.Errorf("expected 3 matched options, got %d", len(opts))
	}
	if opts["ready"] == nil || opts["ready"].ID != "o1" {
		t.Error("expected ready -> o1")
	}
	if opts["in_progress"] == nil || opts["in_progress"].ID != "o2" {
		t.Error("expected in_progress -> o2")
	}
	if opts["done"] == nil || opts["done"].ID != "o3" {
		t.Error("expected done -> o3")
	}
	if opts["in_review"] != nil {
		t.Error("expected in_review to have no match")
	}
}

func TestAutoMapModel_NoFieldMatch(t *testing.T) {
	fields := []Field{
		{ID: "f1", Name: "Title", Type: FieldTypeText},
	}

	field, opts := AutoMapModel(fields, "Status", map[string]string{"ready": "Ready"})

	if field != nil {
		t.Errorf("expected no field match, got %v", field)
	}
	if len(opts) != 0 {
		t.Errorf("expected 0 option matches, got %d", len(opts))
	}
}

func TestAutoMapModel_FieldMatchNoOptions(t *testing.T) {
	fields := []Field{
		{ID: "f1", Name: "Status", Type: FieldTypeText}, // text field, no options
	}

	field, opts := AutoMapModel(fields, "Status", map[string]string{"ready": "Ready"})

	if field == nil {
		t.Fatal("expected field match, got nil")
	}
	if len(opts) != 0 {
		t.Errorf("expected 0 option matches for text field, got %d", len(opts))
	}
}
