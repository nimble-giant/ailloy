package plugin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupPluginForUpdate(t *testing.T) string {
	t.Helper()

	// First generate a full plugin
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "plugin")

	g := NewGenerator(outputDir)
	g.Config = &Config{
		Name:        "update-test",
		Version:     "1.0.0",
		Description: "Plugin for update testing",
		Author: Author{
			Name: "Test",
		},
	}

	if err := g.Generate(); err != nil {
		t.Fatalf("failed to generate initial plugin: %v", err)
	}

	return outputDir
}

func TestNewUpdater(t *testing.T) {
	u := NewUpdater("/tmp/plugin")
	if u == nil {
		t.Fatal("expected non-nil updater")
	}
	if u.PluginPath != "/tmp/plugin" {
		t.Errorf("expected path '/tmp/plugin', got '%s'", u.PluginPath)
	}
	if !strings.HasPrefix(u.BackupPath, "/tmp/plugin.backup.") {
		t.Errorf("expected backup path to start with '/tmp/plugin.backup.', got '%s'", u.BackupPath)
	}
}

func TestUpdater_Backup(t *testing.T) {
	pluginDir := setupPluginForUpdate(t)
	u := NewUpdater(pluginDir)

	err := u.Backup()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify backup directory exists
	info, err := os.Stat(u.BackupPath)
	if err != nil {
		t.Fatalf("backup directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected backup to be a directory")
	}

	// Verify backup contains expected files
	entries, err := os.ReadDir(u.BackupPath)
	if err != nil {
		t.Fatalf("failed to read backup dir: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected backup to contain files")
	}

	// Verify commands directory was backed up
	backupCmds := filepath.Join(u.BackupPath, "commands")
	if _, err := os.Stat(backupCmds); err != nil {
		t.Error("expected commands directory in backup")
	}
}

func TestUpdater_Restore(t *testing.T) {
	pluginDir := setupPluginForUpdate(t)
	u := NewUpdater(pluginDir)

	// Create backup
	if err := u.Backup(); err != nil {
		t.Fatalf("failed to backup: %v", err)
	}

	// Modify the original plugin (add a file)
	markerPath := filepath.Join(pluginDir, "commands", "custom-command.md")
	if err := os.WriteFile(markerPath, []byte("# Custom Command"), 0644); err != nil {
		t.Fatalf("failed to write marker file: %v", err)
	}

	// Restore from backup
	err := u.Restore()
	if err != nil {
		t.Fatalf("unexpected error restoring: %v", err)
	}

	// Verify the custom file is gone (restored from backup)
	if _, err := os.Stat(markerPath); !os.IsNotExist(err) {
		t.Error("expected custom file to be removed after restore")
	}

	// Verify original files still exist
	commandsDir := filepath.Join(pluginDir, "commands")
	entries, err := os.ReadDir(commandsDir)
	if err != nil {
		t.Fatalf("failed to read commands after restore: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected commands to exist after restore")
	}
}

func TestUpdater_Update(t *testing.T) {
	pluginDir := setupPluginForUpdate(t)
	u := NewUpdater(pluginDir)

	err := u.Update()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if u.UpdatedFiles == 0 && u.NewCommands == 0 {
		t.Error("expected some files to be updated or new commands added")
	}

	// Verify commands still exist
	entries, err := os.ReadDir(filepath.Join(pluginDir, "commands"))
	if err != nil {
		t.Fatalf("failed to read commands after update: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected commands to exist after update")
	}
}

func TestUpdater_Update_PreservesCustomCommands(t *testing.T) {
	pluginDir := setupPluginForUpdate(t)

	// Add a custom command
	customPath := filepath.Join(pluginDir, "commands", "custom-team-cmd.md")
	if err := os.WriteFile(customPath, []byte("# Custom Team Command"), 0644); err != nil {
		t.Fatalf("failed to write custom command: %v", err)
	}

	u := NewUpdater(pluginDir)
	err := u.Update()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Custom command should still exist
	if _, err := os.Stat(customPath); err != nil {
		t.Error("expected custom command to be preserved after update")
	}

	if u.PreservedFiles == 0 {
		t.Error("expected at least one preserved file")
	}
}

func TestUpdater_CopyDir(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := filepath.Join(t.TempDir(), "copy-dest")

	// Create source structure
	subDir := filepath.Join(srcDir, "sub")
	if err := os.MkdirAll(subDir, 0750); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("content1"), 0644); err != nil {
		t.Fatalf("failed to write file1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "file2.txt"), []byte("content2"), 0644); err != nil {
		t.Fatalf("failed to write file2: %v", err)
	}

	u := &Updater{}
	err := u.copyDir(srcDir, dstDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify destination
	data1, err := os.ReadFile(filepath.Join(dstDir, "file1.txt"))
	if err != nil {
		t.Fatalf("failed to read copied file1: %v", err)
	}
	if string(data1) != "content1" {
		t.Errorf("expected 'content1', got '%s'", string(data1))
	}

	data2, err := os.ReadFile(filepath.Join(dstDir, "sub", "file2.txt"))
	if err != nil {
		t.Fatalf("failed to read copied file2: %v", err)
	}
	if string(data2) != "content2" {
		t.Errorf("expected 'content2', got '%s'", string(data2))
	}
}

func TestUpdater_CopyFile(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	srcPath := filepath.Join(srcDir, "source.txt")
	dstPath := filepath.Join(dstDir, "dest.txt")

	if err := os.WriteFile(srcPath, []byte("hello world"), 0644); err != nil {
		t.Fatalf("failed to write source: %v", err)
	}

	u := &Updater{}
	err := u.copyFile(srcPath, dstPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("failed to read destination: %v", err)
	}
	if string(data) != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", string(data))
	}
}

func TestUpdater_BackupAndUpdate_FullCycle(t *testing.T) {
	pluginDir := setupPluginForUpdate(t)
	u := NewUpdater(pluginDir)

	// Backup
	if err := u.Backup(); err != nil {
		t.Fatalf("backup failed: %v", err)
	}

	// Update
	if err := u.Update(); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	// Validate the updated plugin
	v := NewValidator(pluginDir)
	result, err := v.Validate()
	if err != nil {
		t.Fatalf("validation failed: %v", err)
	}
	if !result.IsValid {
		t.Errorf("updated plugin is not valid: errors=%v, warnings=%v", result.Errors, result.Warnings)
	}
}
