package commands

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/nimble-giant/ailloy/pkg/styles"
)

type dependency struct {
	name        string
	binary      string
	description string
	installHelp map[string]string // keyed by GOOS
}

var runtimeDeps = []dependency{
	{
		name:        "Git",
		binary:      "git",
		description: "Version control",
		installHelp: map[string]string{
			"darwin":  "brew install git",
			"linux":   "sudo apt install git  # or: sudo dnf install git",
			"windows": "winget install Git.Git",
		},
	},
	{
		name:        "GitHub CLI",
		binary:      "gh",
		description: "GitHub integration for PR and issue blanks",
		installHelp: map[string]string{
			"darwin":  "brew install gh",
			"linux":   "sudo apt install gh  # see: https://cli.github.com",
			"windows": "winget install GitHub.cli",
		},
	},
	{
		name:        "Claude Code",
		binary:      "claude",
		description: "AI-powered development assistant",
		installHelp: map[string]string{
			"darwin":  "npm install -g @anthropic-ai/claude-code",
			"linux":   "npm install -g @anthropic-ai/claude-code",
			"windows": "npm install -g @anthropic-ai/claude-code",
		},
	},
}

// checkDependencies checks for runtime dependencies and prints styled results.
// All checks are warnings only - this never blocks execution.
func checkDependencies() {
	fmt.Println(styles.InfoStyle.Render("🔍 Checking dependencies..."))
	fmt.Println()

	allFound := true

	for _, dep := range runtimeDeps {
		found, version := checkBinary(dep.binary)
		if found {
			label := dep.name
			if version != "" {
				label += " (" + version + ")"
			}
			fmt.Println(styles.SuccessStyle.Render("  ✅ "+label) +
				styles.SubtleStyle.Render(" - "+dep.description))
		} else {
			allFound = false
			fmt.Println(styles.WarningStyle.Render("  ⚠️  "+dep.name) +
				styles.SubtleStyle.Render(" - Not found"))

			installCmd := dep.installHelp[runtime.GOOS]
			if installCmd == "" {
				installCmd = dep.installHelp["linux"] // fallback
			}
			if installCmd != "" {
				fmt.Println("     Install with: " + styles.CodeStyle.Render(installCmd))
			}
		}
	}

	fmt.Println()
	if allFound {
		fmt.Println(styles.SuccessStyle.Render("  All dependencies found!"))
	} else {
		fmt.Println(styles.SubtleStyle.Render("  Missing dependencies are optional but recommended for full functionality."))
	}
	fmt.Println()
}

// checkBinary checks if a binary exists on PATH and attempts to get its version.
func checkBinary(name string) (bool, string) {
	path, err := exec.LookPath(name)
	if err != nil || path == "" {
		return false, ""
	}

	version := getBinaryVersion(name)
	return true, version
}

// getBinaryVersion attempts to get a version string from a binary.
func getBinaryVersion(name string) string {
	cmd := exec.Command(name, "--version") // #nosec G204 -- binary name is from hardcoded list
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	raw := strings.TrimSpace(string(output))

	// Parse version from common output formats
	switch name {
	case "git":
		// "git version 2.43.0" -> "v2.43.0"
		raw = strings.TrimPrefix(raw, "git version ")
		if i := strings.Index(raw, "\n"); i != -1 {
			raw = raw[:i]
		}
		return "v" + raw
	case "gh":
		// "gh version 2.40.1 (2024-01-01)\n..." -> "v2.40.1"
		raw = strings.TrimPrefix(raw, "gh version ")
		if i := strings.Index(raw, " "); i != -1 {
			raw = raw[:i]
		}
		return "v" + raw
	case "claude":
		// Output format varies, just take first line
		if i := strings.Index(raw, "\n"); i != -1 {
			raw = raw[:i]
		}
		return raw
	default:
		if i := strings.Index(raw, "\n"); i != -1 {
			raw = raw[:i]
		}
		return raw
	}
}
