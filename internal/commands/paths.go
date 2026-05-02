package commands

import (
	"os"
	"path/filepath"

	"github.com/nimble-giant/ailloy/pkg/foundry"
)

// projectLockPath returns the lock path inside the current project.
func projectLockPath() string {
	return foundry.LockFileName
}

// projectManifestPath returns the installed-manifest path inside the current project.
func projectManifestPath() string {
	return foundry.InstalledManifestPath
}

// globalLockPath returns the lock path under the user's home directory.
// Mirrors `cast --global`'s install location.
func globalLockPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return foundry.LockFileName
	}
	return filepath.Join(home, foundry.LockFileName)
}

// globalManifestPath returns the installed-manifest path under the user's home.
func globalManifestPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return foundry.InstalledManifestPath
	}
	return filepath.Join(home, foundry.InstalledManifestPath)
}

// lockPathFor returns the project or global lock path based on the global flag.
func lockPathFor(global bool) string {
	if global {
		return globalLockPath()
	}
	return projectLockPath()
}

// manifestPathFor returns the project or global manifest path based on the global flag.
func manifestPathFor(global bool) string {
	if global {
		return globalManifestPath()
	}
	return projectManifestPath()
}
