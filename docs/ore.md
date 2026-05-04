# Ore

Ore are reusable flux partials — structured data objects in the flux namespace that mold authors can opt into. Where [ingots](ingots.md) are reusable *template* partials (chunks of blank content), ore are reusable *flux* partials (chunks of values schema). Together they let you share both the prose and the data shapes that drive it.

A typical ore is a named group of related flux fields under `ore.<name>.*`. For example, `ore.status` describes the "Status" data model with an `enabled` toggle, a `field_id`, a `field_mapping`, and a map of `options`. Blanks consume an ore via conditionals (`{{if .ore.status.enabled}}…{{end}}`) and dotted access (`{{.ore.status.field_id}}`).

Ore are typically **optional** (`enabled: false` by default) and **shareable**: many molds can adopt the same ore schema so a values file or anneal session configured for one mold drops cleanly into another.

## When to Use Ore

| Need | Use |
|------|-----|
| Reusable prose / instruction blocks | [Ingot](ingots.md) |
| Reusable structured data shape (with `enabled` toggle) | **Ore** |
| Single value, one-off | Plain [flux variable](flux.md) |

Pick ore when:

- Multiple blanks (or multiple molds) need the same data shape — not just the same value
- The data represents an **opt-in capability** that some users will turn on and others will leave off
- The fields wrap an external system whose IDs/options can't be hardcoded (e.g., GitHub Project field IDs)

Pick an ingot instead when the reusable thing is text — a preamble, a CLI cheat-sheet, a coding-standards block.

## Anatomy of an Ore

The [official mold](https://github.com/nimble-giant/nimble-mold) defines three ore for GitHub Projects integration: `ore.status`, `ore.priority`, and `ore.iteration`. Here's `ore.status` end-to-end.

### Defaults in `flux.yaml`

```yaml
ore:
  status:
    enabled: false
    field_id: ""
    field_mapping: ""
    options:
      ready:
        id: ""
        label: Ready
      in_progress:
        id: ""
        label: In Progress
      in_review:
        id: ""
        label: In Review
      done:
        id: ""
        label: Done
```

### Schema in `flux.schema.yaml`

```yaml
# --- Ore Models ---
- name: ore.status.enabled
  type: bool
  description: "Enable Status ore model (track issue lifecycle)"
  default: "false"

- name: ore.status.field_id
  type: string
  description: "GitHub Project field ID for Status"
  discover:
    command: |
      gh api graphql -f query='...' -f org='{{.project.organization}}' -F number={{.project.number}}
    parse: |
      {{- range .data.organization.projectV2.fields.nodes -}}
      {{ .name }} ({{ .fieldType }})|{{ .id }}
      {{ end -}}
    prompt: select
```

The `# --- Ore Models ---` section header is the convention that groups ore entries together in the schema file.

### Consumption in a blank

```markdown
{{if .ore.status.enabled}}
## Status Tracking

After each step, update the Status field on the GitHub Project board.

- Field: `{{.ore.status.field_id}}`
- Available values:
{{range $key, $opt := .ore.status.options}}
  - `{{$opt.label}}` (id: `{{$opt.id}}`)
{{end}}
{{end}}
```

When `ore.status.enabled` is `false` (the default), the entire block is omitted from the rendered blank. Users who want status tracking flip the toggle and fill in IDs via [`ailloy anneal`](anneal.md).

## Authoring Conventions

### Naming

- **Lowercase, snake_case** ore names: `ore.status`, not `ore.Status` or `ore.statusField`
- **Name the concept, not the source system**: `ore.status` (concept) over `ore.github_status` (source-bound). This keeps the schema portable if a mold later adopts a different SCM or project tool.
- **One ore per business concept**: `ore.status` and `ore.priority` are siblings, not nested under a common parent. Don't lump unrelated fields together.
- **Always include `enabled: bool` (default `false`)**. The toggle is part of the contract — every consumer gates on it.
- **Plural sub-keys for collections**: `ore.status.options` (a map of named choices), not `ore.status.option_list` or `ore.status.choices`.
- **Mirror upstream vocabulary inside the ore**: if the external system calls them "fields" and "options", use those words. Don't invent new terms; match what users already see in the source system's UI.

### Structure

Each ore should provide, at minimum:

| Field | Type | Purpose |
|-------|------|---------|
| `enabled` | bool | Master toggle. Defaults to `false`. Blanks always gate on this. |
| `field_id` (or similar) | string | The primary external identifier this ore wraps. |
| `options` (when applicable) | map | Named entries for enumerated values, each typically `{id, label}`. |

Add more fields as the concept demands, but resist piling unrelated config into the same ore.

### Discovery patterns

Discovery is a natural fit for ore — most ore wrap an external system whose IDs you can't reasonably ask a user to paste by hand. Use `discover:` blocks in `flux.schema.yaml` to populate dropdowns at anneal time:

```yaml
- name: ore.status.field_id
  type: string
  description: "GitHub Project field ID for Status"
  discover:
    command: |
      gh api graphql -f query='
        query($org: String!, $number: Int!) {
          organization(login: $org) {
            projectV2(number: $number) {
              fields(first: 50) {
                nodes {
                  ... on ProjectV2SingleSelectField { id name dataType }
                }
              }
            }
          }
        }
      ' -f org='{{.project.organization}}' -F number={{.project.number}}
    parse: |
      {{- range .data.organization.projectV2.fields.nodes -}}
      {{ .name }} ({{ .dataType }})|{{ .id }}
      {{ end -}}
    prompt: select
```

Patterns to follow:

- **Reference parent flux values** in the discovery `command` — chain off project-level config (e.g. `{{.project.organization}}`, `{{.project.number}}`) so each ore field's prompt has the context it needs.
- **Use `also_sets:`** to cascade a single selection into sibling fields (e.g. selecting a Status field also populates the option IDs underneath).
- **Run discovery at the most useful level**. For maps of options, you may want one discovery for the parent field ID and a separate discovery (or hand-fill) for each option's `id`.
- **Fail soft**: if discovery's required values aren't populated yet, the wizard skips the prompt with a hint rather than erroring. Order schema entries so dependencies resolve first.

See the [Anneal guide](anneal.md) for the full discovery field reference.

## Ore Package Structure

An ore package is a directory containing three files:

```
my-status-ore/
├── ore.yaml
├── flux.schema.yaml
└── flux.yaml
```

`ore.yaml` (manifest):

```yaml
apiVersion: v1
kind: ore
name: status                      # claimed namespace: ore.status.*
version: 1.0.0
description: "GitHub Project status field tracking"
author:
  name: Nimble Giant
  url: https://github.com/nimble-giant
requires:
  ailloy: ">=0.7.0"
```

`flux.schema.yaml` — entries are **unprefixed**; the ailloy loader prepends `ore.<name>.` (or `ore.<alias>.` if installed `--as`):

```yaml
- name: enabled
  type: bool
  description: "Enable Status ore"
  default: "false"

- name: field_id
  type: string
  description: "GitHub Project field ID for Status"
  discover:
    command: |
      gh api graphql -f query='...'
    parse: |
      ...
    prompt: select
```

`flux.yaml` — defaults, also unprefixed; the loader wraps them under `ore.<name>:` at merge time:

```yaml
enabled: false
field_id: ""
options:
  ready: { id: "", label: Ready }
```

## Creating an Ore

Use the scaffolder:

```bash
ailloy ore new my-status-ore
```

This creates the directory layout above with placeholder content. Edit the files, then commit + tag.

### Manifest Fields

| Field | Required | Description |
|-------|----------|-------------|
| `apiVersion` | Yes | Always `v1` |
| `kind` | Yes | Always `ore` |
| `name` | Yes | Snake_case identifier (the package name) |
| `namespace` | No | Snake_case flux namespace (`ore.<namespace>.*`); falls back to `name` when omitted |
| `version` | Yes | Semver |
| `description` | No | Human-readable description |
| `author` | No | `{name, url}` |
| `requires.ailloy` | No | Minimum ailloy version |

### Namespace Precedence

The flux namespace an ore lands at — the `<X>` in `ore.<X>.*` — is resolved with this precedence chain (highest wins):

1. **`as:` in the consuming mold's `dependencies[]` entry** — per-cast override.
2. **`--as <alias>` at install time** — recorded in `installed.yaml` and pins the on-disk install dir name.
3. **`namespace:` in `ore.yaml`** — publisher-declared canonical namespace.
4. **`name:` in `ore.yaml`** — fallback when none of the above is set.

Layers 1–2 also control the on-disk install dir name; layers 3–4 are layered on top by the resolver. Set `namespace:` when the package's external name differs from the canonical flux key (e.g. publish a package called `status_ore` that lands at `ore.status.*`). Omit it when the two are the same — temper warns about redundant `namespace:` fields.

## Resolution Order

When merging ore deps into a mold's flux schema and defaults, ailloy walks search paths in priority order:

1. **Mold-local** — `./ores/<name>/` (if the mold ships its own ore overlay)
2. **Project** — `.ailloy/ores/<name>/` (cast-time install destination)
3. **Global** — `~/.ailloy/ores/<name>/` (user-scope, `ore add --global`)

First match wins. The mold's own `flux.schema.yaml` always wins over an installed ore on collision.

## Installing Remote Ores

```bash
ailloy ore add github.com/nimble-giant/status-ore
ailloy ore add github.com/nimble-giant/status-ore --as github_status
ailloy ore add github.com/nimble-giant/status-ore --global
ailloy ore get github.com/nimble-giant/status-ore  # download to cache without installing
```

The bidirectional verb forms also work: `ailloy add ore <ref>`, `ailloy get ore <ref>`.

## Declaring Dependencies

Molds can declare ore dependencies in `mold.yaml`:

```yaml
apiVersion: v1
kind: mold
name: my-mold
version: 1.0.0
dependencies:
  - ore: github.com/nimble-giant/status-ore
    version: "^1.0.0"
  - ore: github.com/other-org/status-ore
    version: "^2.0.0"
    as: github_status            # alias to avoid namespace collision
```

`ailloy cast` and `ailloy recast` auto-install declared deps. `ailloy forge` and `ailloy temper` resolve declared deps ephemerally (no on-disk side effects).

For CI, pass `--frozen` to `cast` (or `recast`) to fail loudly on any declared dep that isn't already installed:

```bash
ailloy cast my-mold --frozen
```

With `--frozen` set, a typo or unpinned bump in `mold.yaml` becomes an error referencing the missing dep instead of a silent network fetch + `installed.yaml` / `ailloy.lock` mutation. When every declared dep is already installed, `--frozen` is a no-op and cast proceeds normally.

## Validating Ores

```bash
ailloy temper ./my-status-ore
```

Validates manifest fields, semver, snake_case name, that schema entries are unprefixed, that `flux.yaml` doesn't have a top-level `ore` key, that an `enabled: bool` schema entry exists, and reports orphan defaults as warnings. See the [Validation guide](temper.md) for the full rule list.

`ailloy anneal` continues to enforce type rules at wizard time, and `ailloy forge --debug` shows resolved ore values plus any missing dependencies in discovery commands.

## Distributing Ores

Publish via plain git tag:

```bash
cd my-status-ore/
git init && git add -A && git commit -m "initial ore"
git remote add origin git@github.com:nimble-giant/status-ore.git
git push -u origin main
git tag v1.0.0 && git push --tags
```

Consumers install with `ailloy ore add github.com/nimble-giant/status-ore@v1.0.0`.

## Removing Ores

```bash
ailloy ore remove status              # project scope
ailloy ore remove status --global     # ~/.ailloy/ores
ailloy ore remove status --force      # bypass dependents check
```

`ailloy uninstall <mold>` cascade-removes any ores whose only remaining dependent was the uninstalled mold. User-direct installs (via `ailloy ore add ...`) are never auto-removed — see the [Cascade Uninstall](ingots.md#cascade-uninstall) section in the ingots doc for the shared semantics.

## Migrating an In-Tree Ore to a Package

If you have an ore section embedded in a mold's `flux.schema.yaml` and you want to lift it out into a standalone, versioned package, follow the [Ore Migration Guide](ore-migration.md).

## See Also

- [Flux Variables](flux.md) — the variable system ore is built on
- [Ingots](ingots.md) — the sibling concept for reusable template partials
- [Anneal](anneal.md) — the wizard that configures ore values interactively
- [Helm Users](helm-users.md) — concept map for newcomers
