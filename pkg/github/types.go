package github

// FieldType classifies a ProjectV2 field
type FieldType string

const (
	FieldTypeText         FieldType = "TEXT"
	FieldTypeNumber       FieldType = "NUMBER"
	FieldTypeDate         FieldType = "DATE"
	FieldTypeSingleSelect FieldType = "SINGLE_SELECT"
	FieldTypeIteration    FieldType = "ITERATION"
	FieldTypeUnknown      FieldType = "UNKNOWN"
)

// Project represents a GitHub ProjectV2 board
type Project struct {
	ID     string
	Number int
	Title  string
	URL    string
	Closed bool
}

// Field represents a single field on a ProjectV2 board
type Field struct {
	ID      string
	Name    string
	Type    FieldType
	Options []Option // Populated only for SingleSelect fields
}

// Option represents one choice within a single-select or iteration field
type Option struct {
	ID   string
	Name string
}

// DiscoveryResult holds the full result of discovering a project's fields
type DiscoveryResult struct {
	Project Project
	Fields  []Field
}
