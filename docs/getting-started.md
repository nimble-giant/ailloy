# Getting Started

Ailloy is the package manager for AI instructions. **Molds** are versioned,
configurable packages of AI workflow files (commands, skills, workflows) that
can be installed into any project — much like Helm charts for Kubernetes.

This guide walks you through installing ailloy and casting your first mold.

## 1. Install

**Homebrew (macOS, Linux)**

```bash
brew install nimble-giant/tap/ailloy
```

**Quick install script**

```bash
curl -fsSL https://raw.githubusercontent.com/nimble-giant/ailloy/main/install.sh | bash
```

Verify the install:

```bash
ailloy --version
```

## 2. Cast a Mold

Casting installs a mold into your project — rendering its blanks (templates)
with your flux (values) into the destinations declared by the mold's
`output:` mapping.

```bash
# Cast the official mold from GitHub
ailloy cast github.com/nimble-giant/nimble-mold

# Or cast a local mold directory
ailloy cast ./my-mold
```

A successful cast writes a manifest to `.ailloy/installed.yaml` so the
files can be tracked, refreshed (`ailloy recast`), or removed
(`ailloy uninstall`).

## 3. Configure with Anneal

Most molds expose flux variables — the per-project values that customize the
rendered output. Run `ailloy anneal` for a guided wizard:

```bash
ailloy anneal ./my-mold -o team-values.yaml
ailloy cast ./my-mold -f team-values.yaml
```

Run `ailloy docs flux` for the full flux variable reference.

## 4. Explore the Pipeline

| Step      | Command         | What it does                                    |
|-----------|-----------------|-------------------------------------------------|
| Author    | —               | Write instruction blanks with Go templates       |
| Configure | `ailloy anneal` | Interactive wizard for flux variables            |
| Preview   | `ailloy forge`  | Dry-run render to stdout or a directory          |
| Install   | `ailloy cast`   | Render and install blanks into a project         |
| Package   | `ailloy smelt`  | Bundle a mold into a tarball or binary           |
| Validate  | `ailloy temper` | Validate mold structure and templates            |
| Lint      | `ailloy assay`  | Lint AI instruction files for best practices     |

## In-CLI Documentation

- `ailloy docs` — opens the rich documentation TUI when the docs
  extension is installed; otherwise offers to install it (or falls back
  to a plain glamour-rendered table)
- `ailloy docs <topic>` — render a topic in the terminal
- `ailloy <command> --docs` — render the command's associated topic
- `ailloy docs --no-extension` — force the in-binary fallback

For example:

```bash
ailloy docs flux
ailloy cast --docs
```

The richer experience ships as a separate **extension** binary so the
main `ailloy` install stays small for CI and other lean environments.
Extensions are downloaded on demand and verified by SHA-256 checksum;
manage them with `ailloy extensions` (alias `ext`). Install ailloy with
the docs extension preloaded via:

```bash
curl -fsSL https://raw.githubusercontent.com/nimble-giant/ailloy/main/install.sh | bash -s -- --with-docs
brew install nimble-giant/tap/ailloy-with-docs
```

See `ailloy docs extensions/users` for the full extensions guide.
