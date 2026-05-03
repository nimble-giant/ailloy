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

## Sharing Ore Across Molds

Ore is currently a **flux-namespace convention**, not a packaged artifact. Sharing happens by:

- **Copying schema entries** between molds that want the same ore (the `# --- Ore Models ---` section is designed to be portable)
- **Anneal-produced values files** (`ore.yaml`) that round-trip across any mold sharing the schema — `ailloy anneal mold-a -o ore.yaml` then `ailloy cast mold-b -f ore.yaml`

This is intentional simplicity, not a long-term answer. A future packaging story (an `ores/` directory, an `ore.yaml` manifest, `ailloy ore add`, mold dependencies, lockfile pinning) is tracked in [issue #178](https://github.com/nimble-giant/ailloy/issues/178). Until that ships, treat ore schemas as conventions you copy, not packages you install.

## Validation

Ore today is validated as part of the surrounding flux schema:

- `ailloy temper` checks `flux.schema.yaml` and `flux.yaml` shape, including ore entries
- `ailloy anneal` enforces type rules at wizard time
- `ailloy forge --debug` shows resolved ore values and surfaces missing dependencies in discovery commands

See the [Validation guide](temper.md) and [Anneal guide](anneal.md) for details.

## See Also

- [Flux Variables](flux.md) — the variable system ore is built on
- [Ingots](ingots.md) — the sibling concept for reusable template partials
- [Anneal](anneal.md) — the wizard that configures ore values interactively
- [Helm Users](helm-users.md) — concept map for newcomers
