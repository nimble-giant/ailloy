// Package extensions wraps the SDK's host package with ailloy-specific
// concerns: a huh-driven consent prompt, async background update kick-
// off, and a single Manager instance the cobra commands share.
//
// The SDK does the protocol-level lifting (manifest, install, exec,
// version resolution); this package handles UX (when to prompt, how to
// surface progress, how to mirror NO_COLOR into the protocol env).
package extensions

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/charmbracelet/huh"
	"github.com/nimble-giant/ailloy-extensions-sdk/pkg/host"
	"github.com/nimble-giant/ailloy-extensions-sdk/pkg/manifest"
	"github.com/nimble-giant/ailloy-extensions-sdk/pkg/registry"
)

// Manager is ailloy's view of the extension subsystem. Construct once
// per CLI invocation via NewManager; the underlying *host.Host is
// concurrency-safe for the read-mostly workflows the host executes.
type Manager struct {
	host *host.Host

	// Stdout/Stderr/Stdin are wired from the cobra command at construction
	// time so prompts respect command-level redirection in tests.
	Stdout io.Writer
	Stderr io.Writer
	Stdin  io.Reader

	mu sync.Mutex
}

// NewManager builds a Manager rooted at the user's ~/.ailloy unless
// overridden via the AILLOY_CONFIG_DIR env var.
func NewManager(ailloyVersion string) (*Manager, error) {
	cfg := os.Getenv("AILLOY_CONFIG_DIR")
	h, err := host.New(ailloyVersion, cfg)
	if err != nil {
		return nil, err
	}
	return &Manager{
		host:   h,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Stdin:  os.Stdin,
	}, nil
}

// Host exposes the underlying SDK host (for tests and advanced callers).
func (m *Manager) Host() *host.Host { return m.host }

// IsInstalled returns whether the named extension has a binary on disk
// recorded in the manifest.
func (m *Manager) IsInstalled(name string) bool {
	mn, err := m.host.LoadManifest()
	if err != nil {
		return false
	}
	e, ok := mn.Get(name)
	return ok && e.BinaryPath != ""
}

// IsDeclined returns whether the user has previously declined to
// install this extension. The host re-prompts after `ailloy ext reset`.
func (m *Manager) IsDeclined(name string) bool {
	mn, err := m.host.LoadManifest()
	if err != nil {
		return false
	}
	e, ok := mn.Get(name)
	return ok && e.Declined && !e.ConsentGranted
}

// PromptInstall asks the user whether to install the extension. The
// answer is recorded in the manifest so the prompt only fires once.
// Returns the user's decision.
func (m *Manager) PromptInstall(name string) (bool, error) {
	if !registry.IsKnown(name) {
		return false, fmt.Errorf("unknown extension %q", name)
	}
	if !isInteractive() {
		// Non-interactive callers (CI, pipes) don't get a prompt; they
		// must opt in with --yes or `ailloy ext install` explicitly.
		return false, nil
	}

	var approve bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(consentTitle(name)).
				Description(consentBody(name)).
				Affirmative("Install").
				Negative("Not now").
				Value(&approve),
		),
	).WithTheme(huh.ThemeBase())

	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return false, m.recordDeclined(name)
		}
		return false, err
	}
	if !approve {
		return false, m.recordDeclined(name)
	}
	return true, nil
}

// Install installs the extension named (or referenced) by ref. ref may
// be a registry name ("docs") or a github.com/owner/repo URL.
func (m *Manager) Install(ref string) error {
	src, err := host.Resolve(ref)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	warnFn := func(msg string) {
		_, _ = fmt.Fprintln(m.Stderr, "  ⚠ "+msg)
	}
	_, _ = fmt.Fprintf(m.Stdout, "📦 Installing %s from %s…\n", src.Name, src.Source)
	ext, err := m.host.Install(src, host.InstallOptions{WarnFn: warnFn})
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(m.Stdout, "✅ Installed %s %s\n", ext.Name, ext.InstalledVersion)
	return nil
}

// Update checks for a newer release of the named extension and installs
// it if available.
func (m *Manager) Update(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	old, neu, err := m.host.Update(name, host.InstallOptions{
		WarnFn: func(msg string) { _, _ = fmt.Fprintln(m.Stderr, "  ⚠ "+msg) },
	})
	if err != nil {
		return err
	}
	if old == neu {
		_, _ = fmt.Fprintf(m.Stdout, "✓ %s is already up to date (%s)\n", name, neu)
		return nil
	}
	_, _ = fmt.Fprintf(m.Stdout, "⬆ Updated %s: %s → %s\n", name, old, neu)
	return nil
}

// Remove deletes an installed extension's binary and manifest entry.
func (m *Manager) Remove(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.host.Remove(name); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(m.Stdout, "🗑 Removed %s\n", name)
	return nil
}

// Reset clears the manifest. consentOnly=true keeps installed binaries
// but clears the consent/declined flags so prompts re-fire.
func (m *Manager) Reset(consentOnly bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.host.Reset(consentOnly); err != nil {
		return err
	}
	if consentOnly {
		_, _ = fmt.Fprintln(m.Stdout, "🔄 Cleared extension consent records (binaries kept).")
	} else {
		_, _ = fmt.Fprintln(m.Stdout, "🔄 Reset all extensions.")
	}
	return nil
}

// Run execs the named extension. Returns its exit code.
func (m *Manager) Run(name string, args []string) (int, error) {
	env := []string{}
	if os.Getenv("NO_COLOR") != "" {
		env = append(env, "AILLOY_NO_COLOR=1")
	}
	return m.host.Run(name, args, env)
}

// List returns every extension recorded in the manifest, plus available
// (not-yet-installed) extensions from the registry.
func (m *Manager) List() ([]ListEntry, error) {
	mn, err := m.host.LoadManifest()
	if err != nil {
		return nil, err
	}
	known := registry.All()
	out := make([]ListEntry, 0, len(known))
	for _, k := range known {
		entry := ListEntry{
			Name:        k.Name,
			Source:      k.Source,
			Description: k.Description,
		}
		if e, ok := mn.Get(k.Name); ok {
			entry.Installed = e.BinaryPath != ""
			entry.InstalledVersion = e.InstalledVersion
			entry.AilloyDocsVersion = e.AilloyDocsVersion
			entry.Declined = e.Declined && !e.ConsentGranted
		}
		out = append(out, entry)
	}
	// Surface any non-registry extensions that are installed.
	for _, name := range mn.Names() {
		if registry.IsKnown(name) {
			continue
		}
		e, _ := mn.Get(name)
		out = append(out, ListEntry{
			Name:              name,
			Source:            e.Source,
			Installed:         e.BinaryPath != "",
			InstalledVersion:  e.InstalledVersion,
			AilloyDocsVersion: e.AilloyDocsVersion,
		})
	}
	return out, nil
}

// Show returns full manifest entry detail for one extension.
func (m *Manager) Show(name string) (manifest.Extension, bool, error) {
	mn, err := m.host.LoadManifest()
	if err != nil {
		return manifest.Extension{}, false, err
	}
	e, ok := mn.Get(name)
	return e, ok, nil
}

// ListEntry is a list-and-status view used by `ailloy ext list`.
type ListEntry struct {
	Name              string
	Source            string
	Description       string
	Installed         bool
	InstalledVersion  string
	AilloyDocsVersion string
	Declined          bool
}

// recordDeclined marks the extension as declined so the host doesn't
// re-prompt on every run. The user can clear this with `ailloy ext
// reset --consent-only`.
func (m *Manager) recordDeclined(name string) error {
	mn, err := m.host.LoadManifest()
	if err != nil {
		return err
	}
	e, _ := mn.Get(name)
	e.Name = name
	if reg, ok := registry.Lookup(name); ok {
		e.Source = reg.Source
	}
	e.Declined = true
	e.ConsentGranted = false
	mn.Set(e)
	return m.host.SaveManifest(mn)
}

func consentTitle(name string) string {
	if reg, ok := registry.Lookup(name); ok {
		return fmt.Sprintf("Install the %s extension?", reg.Name)
	}
	return fmt.Sprintf("Install the %s extension?", name)
}

func consentBody(name string) string {
	reg, ok := registry.Lookup(name)
	src := name
	desc := ""
	if ok {
		src = reg.Source
		desc = reg.Description
	}
	return fmt.Sprintf(
		"%s\n\nailloy will download a binary release from\n  %s\n\n"+
			"Releases are checksummed; the binary is stored under\n"+
			"~/.ailloy/extensions/. You can manage extensions anytime\n"+
			"with `ailloy extensions`.",
		desc, src,
	)
}
