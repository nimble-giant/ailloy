package assay

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

// AssayResult holds the outcome of an assay run.
type AssayResult struct {
	Diagnostics   []mold.Diagnostic
	FilesScanned  int
	Platforms     []Platform
	ContextStats  []FileContextStat
	ContextWindow int  // context window size in tokens used for threshold calculations
	Verbose       bool // when true, formatters show all context stats even without threshold breaches
}

// HasErrors returns true if any diagnostic is an error.
func (r *AssayResult) HasErrors() bool {
	for _, d := range r.Diagnostics {
		if d.Severity == mold.SeverityError {
			return true
		}
	}
	return false
}

// Errors returns only error-severity diagnostics.
func (r *AssayResult) Errors() []mold.Diagnostic {
	return filterBySeverity(r.Diagnostics, mold.SeverityError)
}

// Warnings returns only warning-severity diagnostics.
func (r *AssayResult) Warnings() []mold.Diagnostic {
	return filterBySeverity(r.Diagnostics, mold.SeverityWarning)
}

// Suggestions returns only suggestion-severity diagnostics.
func (r *AssayResult) Suggestions() []mold.Diagnostic {
	return filterBySeverity(r.Diagnostics, mold.SeveritySuggestion)
}

// HasFailures returns true if any diagnostic meets or exceeds the given severity threshold.
func (r *AssayResult) HasFailures(failOn mold.DiagSeverity) bool {
	for _, d := range r.Diagnostics {
		if d.Severity <= failOn {
			return true
		}
	}
	return false
}

// HasContextFindings returns true if any diagnostic from the context-usage rule exists.
func (r *AssayResult) HasContextFindings() bool {
	for _, d := range r.Diagnostics {
		if d.Rule == "context-usage" {
			return true
		}
	}
	return false
}

func filterBySeverity(diags []mold.Diagnostic, sev mold.DiagSeverity) []mold.Diagnostic {
	var filtered []mold.Diagnostic
	for _, d := range diags {
		if d.Severity == sev {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

// Assay runs all applicable rules against the AI instruction files in rootDir.
func Assay(rootDir string, cfg *Config) (*AssayResult, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Convert configured platform strings to Platform type
	var platforms []Platform
	for _, p := range cfg.Platforms {
		platforms = append(platforms, Platform(p))
	}

	// Detect AI instruction files
	files, err := Detect(rootDir, platforms)
	if err != nil {
		return nil, fmt.Errorf("detecting files: %w", err)
	}

	// Also detect Claude schema files (agents, commands, settings)
	schemaFiles := detectClaudeSchemaFilesFS(rootDir)

	// Detect Claude plugin directories
	pluginFiles := detectPluginFiles(rootDir)
	for _, pf := range pluginFiles {
		found := false
		for _, f := range files {
			if f.Path == pf.Path {
				found = true
				break
			}
		}
		if !found {
			files = append(files, pf)
		}
	}
	for _, path := range schemaFiles {
		// Skip if already detected
		found := false
		for _, f := range files {
			if f.Path == path {
				found = true
				break
			}
		}
		if found {
			continue
		}
		content, err := os.ReadFile(filepath.Join(rootDir, path)) //#nosec G304
		if err != nil {
			continue
		}
		files = append(files, DetectedFile{
			Path:     path,
			Platform: PlatformClaude,
			Content:  content,
		})
	}

	// Apply ignore patterns
	if len(cfg.Ignore) > 0 {
		files = filterIgnored(files, cfg.Ignore)
	}

	// Collect unique platforms
	platformSet := make(map[Platform]bool)
	for _, f := range files {
		platformSet[f.Platform] = true
	}
	var detectedPlatforms []Platform
	for p := range platformSet {
		detectedPlatforms = append(detectedPlatforms, p)
	}

	result := &AssayResult{
		FilesScanned: len(files),
		Platforms:    detectedPlatforms,
	}

	// Run rules
	ctx := &RuleContext{
		RootDir: rootDir,
		Files:   files,
		Config:  cfg,
	}

	for _, rule := range AllRules() {
		if !cfg.IsRuleEnabled(rule.Name()) {
			continue
		}

		// Filter by platform if the rule is platform-specific
		rulePlatforms := rule.Platforms()
		if len(rulePlatforms) > 0 {
			applicable := false
			for _, rp := range rulePlatforms {
				if platformSet[rp] {
					applicable = true
					break
				}
			}
			if !applicable {
				continue
			}
		}

		diags := rule.Check(ctx)
		result.Diagnostics = append(result.Diagnostics, diags...)
	}

	result.ContextStats = ctx.ContextStats
	result.ContextWindow = ctx.ContextWindow
	if result.ContextWindow == 0 {
		result.ContextWindow = defaultContextWindow(PlatformClaude)
	}
	return result, nil
}

// detectClaudeSchemaFilesFS finds Claude config files using the real filesystem.
func detectClaudeSchemaFilesFS(rootDir string) []string {
	var paths []string
	patterns := []string{
		filepath.Join(".claude", "agents", "*.yml"),
		filepath.Join(".claude", "agents", "*.yaml"),
		filepath.Join(".claude", "commands", "*.md"),
		filepath.Join(".claude", "settings.json"),
	}
	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(rootDir, pattern))
		if err != nil {
			continue
		}
		for _, m := range matches {
			rel, err := filepath.Rel(rootDir, m)
			if err != nil {
				continue
			}
			paths = append(paths, rel)
		}
	}
	return paths
}

// detectPluginFiles walks rootDir looking for Claude plugin directories
// (identified by a .claude-plugin/plugin.json manifest) and returns all
// relevant files (manifest, commands, agents) as DetectedFile entries with
// PluginDir set to the plugin's root directory relative to rootDir.
func detectPluginFiles(rootDir string) []DetectedFile {
	var files []DetectedFile

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == "node_modules" || name == "vendor" || name == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		// Locate .claude-plugin/plugin.json to identify a plugin root
		if d.Name() == "plugin.json" && filepath.Base(filepath.Dir(path)) == ".claude-plugin" {
			pluginRoot := filepath.Dir(filepath.Dir(path))
			pluginDirRel, err := filepath.Rel(rootDir, pluginRoot)
			if err != nil {
				return nil
			}

			// Include the manifest itself
			manifestRel, _ := filepath.Rel(rootDir, path)
			if content, err := os.ReadFile(path); err == nil { //#nosec G304 G122
				files = append(files, DetectedFile{
					Path:      manifestRel,
					Platform:  PlatformClaude,
					Content:   content,
					PluginDir: pluginDirRel,
				})
			}

			// Include all lintable plugin subdirectories
			addPluginSubdirFiles(rootDir, pluginRoot, pluginDirRel, "commands", []string{".md"}, &files)
			addPluginSubdirFiles(rootDir, pluginRoot, pluginDirRel, "skills", []string{".md"}, &files)
			addPluginSubdirFiles(rootDir, pluginRoot, pluginDirRel, "rules", []string{".md"}, &files)
			addPluginSubdirFiles(rootDir, pluginRoot, pluginDirRel, "agents", []string{".yml", ".yaml"}, &files)
			addPluginSubdirFiles(rootDir, pluginRoot, pluginDirRel, "hooks", []string{".json"}, &files)
		}
		return nil
	})
	_ = err
	return files
}

// addPluginSubdirFiles recursively appends files from a plugin sub-directory.
func addPluginSubdirFiles(rootDir, pluginRoot, pluginDirRel, subdir string, exts []string, files *[]DetectedFile) {
	subdirPath := filepath.Join(pluginRoot, subdir)
	extSet := make(map[string]bool, len(exts))
	for _, ext := range exts {
		extSet[ext] = true
	}

	_ = filepath.WalkDir(subdirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !extSet[filepath.Ext(path)] {
			return nil
		}
		rel, err := filepath.Rel(rootDir, path)
		if err != nil {
			return nil
		}
		content, err := os.ReadFile(path) //#nosec G304 G122
		if err != nil {
			return nil
		}
		*files = append(*files, DetectedFile{
			Path:      rel,
			Platform:  PlatformClaude,
			Content:   content,
			PluginDir: pluginDirRel,
		})
		return nil
	})
}

// filterIgnored removes files matching any of the ignore glob patterns.
func filterIgnored(files []DetectedFile, patterns []string) []DetectedFile {
	var result []DetectedFile
	for _, f := range files {
		ignored := false
		for _, pattern := range patterns {
			if ok, _ := filepath.Match(pattern, f.Path); ok {
				ignored = true
				break
			}
		}
		if !ignored {
			result = append(result, f)
		}
	}
	return result
}
