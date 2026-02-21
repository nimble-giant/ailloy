package plugin

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nimble-giant/ailloy/pkg/templates"
)

// Updater handles updating existing Claude Code plugins
type Updater struct {
	PluginPath     string
	BackupPath     string
	reader         *templates.MoldReader
	UpdatedFiles   int
	NewCommands    int
	PreservedFiles int
}

// NewUpdater creates a new plugin updater
func NewUpdater(pluginPath string, reader *templates.MoldReader) *Updater {
	return &Updater{
		PluginPath: pluginPath,
		BackupPath: pluginPath + ".backup." + time.Now().Format("20060102-150405"),
		reader:     reader,
	}
}

// Update updates an existing plugin with new templates
func (u *Updater) Update() error {
	// Create a generator for the new content
	generator := NewGenerator(u.PluginPath, u.reader)
	generator.Config = &Config{
		Name:        "ailloy",
		Version:     "1.0.0",
		Description: "AI-assisted development workflows and structured templates for Claude Code",
		Author: Author{
			Name:  "Ailloy Team",
			Email: "support@ailloy.dev",
			URL:   "https://github.com/nimble-giant/ailloy",
		},
	}

	// Load templates
	if err := generator.loadTemplates(); err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}

	// Update commands
	if err := u.updateCommands(generator); err != nil {
		return fmt.Errorf("failed to update commands: %w", err)
	}

	// Update manifest
	if err := generator.generateManifest(); err != nil {
		return fmt.Errorf("failed to update manifest: %w", err)
	}
	u.UpdatedFiles++

	// Update README
	if err := generator.generateREADME(); err != nil {
		return fmt.Errorf("failed to update README: %w", err)
	}
	u.UpdatedFiles++

	return nil
}

// Backup creates a backup of the existing plugin
func (u *Updater) Backup() error {
	// Create backup directory
	if err := os.MkdirAll(u.BackupPath, 0750); err != nil { // #nosec G301 -- Backup directory needs group read access
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Copy all files to backup
	return u.copyDir(u.PluginPath, u.BackupPath)
}

// Restore restores from backup
func (u *Updater) Restore() error {
	// Remove current plugin
	if err := os.RemoveAll(u.PluginPath); err != nil {
		return fmt.Errorf("failed to remove current plugin: %w", err)
	}

	// Restore from backup
	return u.copyDir(u.BackupPath, u.PluginPath)
}

func (u *Updater) updateCommands(generator *Generator) error {
	commandsPath := filepath.Join(u.PluginPath, "commands")

	// Get list of existing commands
	existingCommands := make(map[string]bool)
	if entries, err := os.ReadDir(commandsPath); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && filepath.Ext(entry.Name()) == ".md" {
				name := strings.TrimSuffix(entry.Name(), ".md")
				existingCommands[name] = true
			}
		}
	}

	// Transform and update commands
	transformer := NewTransformer()
	for _, tmpl := range generator.commands {
		// Transform template
		command, err := transformer.Transform(tmpl)
		if err != nil {
			return fmt.Errorf("failed to transform template %s: %w", tmpl.Name, err)
		}

		// Write command file
		cmdPath := filepath.Join(commandsPath, tmpl.Name+".md")
		//#nosec G306 -- Command files need to be readable
		if err := os.WriteFile(cmdPath, command, 0644); err != nil {
			return fmt.Errorf("failed to write command %s: %w", tmpl.Name, err)
		}

		// Track updates
		if existingCommands[tmpl.Name] {
			u.UpdatedFiles++
		} else {
			u.NewCommands++
		}

		// Remove from existing commands map
		delete(existingCommands, tmpl.Name)
	}

	// Count preserved custom commands
	for cmdName := range existingCommands {
		if !strings.HasPrefix(cmdName, "ailloy-") {
			u.PreservedFiles++
		}
	}

	return nil
}

func (u *Updater) copyDir(src, dst string) error {
	// Get source info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create destination with same permissions
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	// Read directory contents
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectories
			if err := u.copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			if err := u.copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func (u *Updater) copyFile(src, dst string) error {
	// Open source file
	srcFile, err := os.Open(src) // #nosec G304 -- CLI tool copies plugin files during backup/update
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	// Get source file info
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	// Create destination file
	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode()) // #nosec G304 G703 -- CLI tool creates backup files
	if err != nil {
		return err
	}
	defer func() { _ = dstFile.Close() }()

	// Copy content
	_, err = io.Copy(dstFile, srcFile)
	return err
}
