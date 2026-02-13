package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Validator validates Claude Code plugin structure
type Validator struct {
	PluginPath string
}

// ValidationResult contains the results of plugin validation
type ValidationResult struct {
	IsValid      bool
	HasManifest  bool
	HasCommands  bool
	HasREADME    bool
	CommandCount int
	Warnings     []string
	Errors       []string
}

// NewValidator creates a new plugin validator
func NewValidator(pluginPath string) *Validator {
	return &Validator{
		PluginPath: pluginPath,
	}
}

// Validate performs validation on the plugin structure
func (v *Validator) Validate() (*ValidationResult, error) {
	result := &ValidationResult{
		IsValid:  true,
		Warnings: []string{},
		Errors:   []string{},
	}

	// Check if plugin directory exists
	if _, err := os.Stat(v.PluginPath); err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Plugin directory not found: %s", v.PluginPath))
		return result, fmt.Errorf("plugin directory not found: %s", v.PluginPath)
	}

	// Validate manifest
	v.validateManifest(result)

	// Validate commands
	v.validateCommands(result)

	// Validate README
	v.validateREADME(result)

	// Validate hooks (optional)
	v.validateHooks(result)

	// Validate scripts (optional)
	v.validateScripts(result)

	// Set overall validity
	if len(result.Errors) > 0 {
		result.IsValid = false
	}

	return result, nil
}

func (v *Validator) validateManifest(result *ValidationResult) {
	manifestPath := filepath.Join(v.PluginPath, ".claude-plugin", "plugin.json")

	data, err := os.ReadFile(manifestPath) // #nosec G304 -- CLI tool validates plugin files
	if err != nil {
		result.HasManifest = false
		result.Errors = append(result.Errors, "Missing plugin manifest (.claude-plugin/plugin.json)")
		return
	}

	result.HasManifest = true

	// Validate manifest JSON
	var manifest map[string]interface{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Invalid manifest JSON: %v", err))
		return
	}

	// Check required fields
	requiredFields := []string{"name", "version", "description"}
	for _, field := range requiredFields {
		if _, ok := manifest[field]; !ok {
			result.Errors = append(result.Errors, fmt.Sprintf("Manifest missing required field: %s", field))
		}
	}

	// Check version format
	if version, ok := manifest["version"].(string); ok {
		if !isValidVersion(version) {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Invalid version format: %s", version))
		}
	}
}

func (v *Validator) validateCommands(result *ValidationResult) {
	commandsPath := filepath.Join(v.PluginPath, "commands")

	entries, err := os.ReadDir(commandsPath)
	if err != nil {
		result.HasCommands = false
		result.Errors = append(result.Errors, "Commands directory not found or not accessible")
		return
	}

	commandCount := 0
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".md" {
			commandCount++

			// Validate command file
			cmdPath := filepath.Join(commandsPath, entry.Name())
			v.validateCommandFile(cmdPath, result)
		}
	}

	result.CommandCount = commandCount
	if commandCount > 0 {
		result.HasCommands = true
	} else {
		result.HasCommands = false
		result.Errors = append(result.Errors, "No command files found in commands directory")
	}
}

func (v *Validator) validateCommandFile(cmdPath string, result *ValidationResult) {
	content, err := os.ReadFile(cmdPath) // #nosec G304 -- CLI tool validates plugin command files
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Cannot read command file: %s", filepath.Base(cmdPath)))
		return
	}

	// Check for required elements
	contentStr := string(content)
	cmdName := filepath.Base(cmdPath)

	// Check for command name header
	if !hasCommandHeader(contentStr) {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Command %s missing proper header", cmdName))
	}

	// Check for description
	if !hasDescription(contentStr) {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Command %s missing description", cmdName))
	}

	// Check for instructions
	if !hasInstructions(contentStr) {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Command %s missing instructions for Claude", cmdName))
	}
}

func (v *Validator) validateREADME(result *ValidationResult) {
	readmePath := filepath.Join(v.PluginPath, "README.md")

	if _, err := os.Stat(readmePath); err != nil {
		result.HasREADME = false
		result.Warnings = append(result.Warnings, "README.md not found (recommended for documentation)")
	} else {
		result.HasREADME = true
	}
}

func (v *Validator) validateHooks(result *ValidationResult) {
	hooksPath := filepath.Join(v.PluginPath, "hooks", "hooks.json")

	if _, err := os.Stat(hooksPath); err == nil {
		// Hooks file exists, validate it
		data, err := os.ReadFile(hooksPath) // #nosec G304 -- CLI tool validates plugin hooks files
		if err == nil {
			var hooks map[string]interface{}
			if err := json.Unmarshal(data, &hooks); err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Invalid hooks.json: %v", err))
			}
		}
	}
}

func (v *Validator) validateScripts(result *ValidationResult) {
	scriptsPath := filepath.Join(v.PluginPath, "scripts")

	if _, err := os.Stat(scriptsPath); err == nil {
		// Check for install script
		installScript := filepath.Join(scriptsPath, "install.sh")
		if _, err := os.Stat(installScript); err != nil {
			result.Warnings = append(result.Warnings, "Missing install.sh script (recommended)")
		}
	}
}

// Helper functions

func isValidVersion(version string) bool {
	// Simple semantic version check (x.y.z)
	return len(version) > 0 && (version[0] >= '0' && version[0] <= '9')
}

func hasCommandHeader(content string) bool {
	return len(content) > 0 && content[0] == '#'
}

func hasDescription(content string) bool {
	return containsPattern(content, "description:")
}

func hasInstructions(content string) bool {
	return containsPattern(content, "## Instructions") || containsPattern(content, "When this command")
}

func containsPattern(content, pattern string) bool {
	return len(content) > len(pattern) && (findSubstring(content, pattern) != -1)
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}