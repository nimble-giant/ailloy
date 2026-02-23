# Ingots

Ingots are reusable template components that can be included in blanks via the `{{ingot "name"}}` template function. They work like partial templates — define common instruction blocks once, then include them across multiple blanks.

## When to Use Ingots

Ingots are useful when multiple blanks share the same instructions. For example:

- A standard preamble that every command should include
- Common GitHub CLI patterns used across PR and issue blanks
- Team-specific coding standards referenced by multiple skills

Instead of duplicating content across blanks, extract it into an ingot and include it with `{{ingot "name"}}`.

## Ingot Structure

Ingots come in two forms:

### Manifest-based (directory)

A directory with an `ingot.yaml` manifest listing the content files:

```
ingots/
└── team-preamble/
    ├── ingot.yaml
    ├── intro.md
    └── conventions.md
```

```yaml
# ingot.yaml
apiVersion: v1
kind: ingot
name: team-preamble
version: 1.0.0
description: "Standard team preamble for all commands"
files:
  - intro.md
  - conventions.md
```

When resolved, all files listed in `files:` are concatenated in order.

### Bare file

A single Markdown file in the `ingots/` directory:

```
ingots/
└── team-preamble.md
```

Bare files are simpler but limited to a single file. The resolver tries the manifest-based form first, then falls back to bare files.

## Creating an Ingot

### 1. Create the ingot directory

Inside your mold, create a directory under `ingots/`:

```bash
mkdir -p ingots/github-patterns
```

### 2. Write `ingot.yaml`

```yaml
apiVersion: v1
kind: ingot
name: github-patterns
version: 1.0.0
description: "Common GitHub CLI patterns for PR and issue blanks"
files:
  - pr-helpers.md
  - issue-helpers.md
```

**Manifest fields:**

| Field | Required | Description |
|-------|----------|-------------|
| `apiVersion` | Yes | Always `v1` |
| `kind` | Yes | Always `ingot` |
| `name` | Yes | Unique identifier |
| `version` | Yes | Semver version (e.g., `1.0.0`) |
| `description` | No | Human-readable description |
| `files` | No | List of content files to concatenate |
| `requires` | No | Minimum ailloy version (e.g., `ailloy: ">=0.2.0"`) |

### 3. Write the content files

```markdown
<!-- pr-helpers.md -->
## PR Patterns

Use `{{ scm.cli }} pr list` to find open pull requests.
Use `{{ scm.cli }} pr view <number>` to see PR details.
```

```markdown
<!-- issue-helpers.md -->
## Issue Patterns

Use `{{ scm.cli }} issue list` to find open issues.
Use `{{ scm.cli }} issue view <number>` to see issue details.
```

Ingot content files support the same template syntax as blanks — flux variables, conditionals, ranges, and even nested `{{ingot "name"}}` calls.

### 4. Use in a blank

```markdown
# Create Issue

{{ingot "github-patterns"}}

## Instructions

Create a new issue for {{ project.organization }}.
```

## Resolution Order

When `{{ingot "name"}}` is called during rendering, the resolver searches these paths in order:

1. **Mold-local** — `./ingots/` (the mold's own ingots directory)
2. **Project** — `.ailloy/ingots/` (ingots installed in the current project)
3. **Global** — `~/.ailloy/ingots/` (user-level shared ingots)

The first match wins. This allows molds to bundle their own ingots while also pulling in shared ingots from the project or user level.

## Template Processing

Ingot content is rendered through the same Go template engine with the same flux context as the including blank. This means:

- Ingots can reference flux variables: `{{ project.organization }}`
- Ingots can use conditionals: `{{if .scm.provider}}...{{end}}`
- Ingots can include other ingots: `{{ingot "other-ingot"}}`

Circular references are detected and reported as errors (e.g., ingot A includes ingot B which includes ingot A).

## Installing Remote Ingots

Ingots can be published as standalone git repositories and installed into your project:

```bash
# Download to local cache (inspect before installing)
ailloy ingot get github.com/my-org/my-ingot@v1.0.0

# Download and install into .ailloy/ingots/
ailloy ingot add github.com/my-org/my-ingot@v1.0.0
```

After `ingot add`, the ingot files are copied to `.ailloy/ingots/<name>/` where the template engine can resolve them during `cast` and `forge`.

Bidirectional command forms also work:

```bash
ailloy get ingot github.com/my-org/my-ingot@v1.0.0
ailloy add ingot github.com/my-org/my-ingot@v1.0.0
```

## Declaring Dependencies

Molds can declare ingot dependencies in `mold.yaml`:

```yaml
apiVersion: v1
kind: mold
name: my-mold
version: 1.0.0
dependencies:
  - ingot: github.com/my-org/my-ingot
    version: "^1.0.0"
```

## Validating Ingots

Use `ailloy temper` to validate an ingot's structure:

```bash
ailloy temper ./my-ingot
```

This checks:

- `ingot.yaml` is present and parseable
- Required manifest fields (`apiVersion`, `kind`, `name`, `version`) are set
- Version is valid semver
- All files listed in `files:` exist
- All `.md` files have valid template syntax

See the [Validation guide](temper.md) for more details.

## Distributing Ingots

Publish an ingot the same way you publish a mold — push to a git repository and tag with semver:

```bash
cd my-ingot/
git init && git add -A && git commit -m "initial ingot"
git remote add origin git@github.com:my-org/my-ingot.git
git push -u origin main
git tag v1.0.0 && git push --tags
```

Others can then install it with:

```bash
ailloy ingot add github.com/my-org/my-ingot@v1.0.0
```

For more on remote resolution, versioning, and caching, see the [Remote Molds guide](foundry.md).
