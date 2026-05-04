# Ore Migration Guide

This guide walks through lifting an in-tree ore out of a mold and publishing it as a standalone ore package. The steps are mechanical and can be performed one ore at a time — Q3-A precedence (mold-local always wins) means in-progress migrations don't break consumers.

## When to Migrate

Migrate when:
- Multiple molds need the same ore schema.
- You want to version + lockfile-pin the ore.
- Your ore section in `flux.schema.yaml` has grown large enough to deserve its own home.

Stay in-tree when the ore is mold-specific and not reusable.

## The Migration Steps

### 1. Identify the ore section

Find the `# --- Ore Models ---` section (or equivalent) in your mold's `flux.schema.yaml`. Note all entries with the prefix `ore.<name>.`.

### 2. Scaffold the ore package

```bash
mkdir -p ../my-org-ores
cd ../my-org-ores
ailloy ore new <name>
cd <name>
```

This creates:

```
<name>/
├── ore.yaml
├── flux.schema.yaml
└── flux.yaml
```

### 3. Lift the schema entries

Cut the `ore.<name>.*` entries from your mold's `flux.schema.yaml` and paste them into the ore package's `flux.schema.yaml`. **Strip the `ore.<name>.` prefix from every `name:` field.** The ailloy loader will add it back at install time.

For example, this in the mold:

```yaml
- name: ore.status.enabled
  type: bool
  default: "false"

- name: ore.status.field_id
  type: string
```

Becomes this in the ore package:

```yaml
- name: enabled
  type: bool
  default: "false"

- name: field_id
  type: string
```

### 4. Lift the defaults

Cut the `ore: <name>: ...` block from your mold's `flux.yaml` and paste **the contents** (without the `ore: <name>:` wrapper) into the ore package's `flux.yaml`.

For example, this in the mold:

```yaml
ore:
  status:
    enabled: false
    field_id: ""
    options:
      ready: { id: "", label: Ready }
```

Becomes this in the ore package:

```yaml
enabled: false
field_id: ""
options:
  ready: { id: "", label: Ready }
```

### 5. Validate

```bash
ailloy temper .
```

Should pass with zero errors. Common issues:

- Schema entries still prefixed → strip the `ore.<name>.` prefix.
- Missing `enabled: bool` → ore convention requires it.
- Top-level `ore` key in `flux.yaml` → strip the wrapper.

### 6. Publish

```bash
git init && git add -A && git commit -m "initial ore"
git remote add origin git@github.com:my-org/<name>-ore.git
git push -u origin main
git tag v1.0.0 && git push --tags
```

### 7. Update the consuming mold

In `mold.yaml`, add the dependency:

```yaml
dependencies:
  - ore: github.com/my-org/<name>-ore
    version: "^1.0.0"
```

Remove the `ore.<name>.*` entries from the mold's `flux.schema.yaml` (the lifted ones).

Remove the `ore: <name>:` block from the mold's `flux.yaml`.

### 8. Verify byte-identical output

Cast a test project before and after the migration:

```bash
# Before migration (with the in-tree ore)
ailloy cast my-mold -o /tmp/before/

# After migration (with the packaged dep)
ailloy cast my-mold -o /tmp/after/

# Diff
diff -r /tmp/before /tmp/after
```

The output should be identical. If it isn't, check:
- Did you strip the prefix from every entry?
- Did you preserve all default values?
- Did your discover blocks reference the correct parent flux variables?

## Note: `namespace:` field

Newer ailloy versions support an optional `namespace:` field in `ore.yaml` that decouples the package name from the canonical flux namespace. **Existing ores without `namespace:` keep working unchanged** — when the field is absent the namespace falls back to `name:`, matching the pre-`namespace:` behavior. Add `namespace:` only when you want the package's external name to differ from the flux key (e.g. publishing `status_ore` that lands at `ore.status.*`). See [docs/ore.md](ore.md) for the full precedence chain.

## See Also

- [Ore](ore.md) — concept and authoring guide
- [Ingots](ingots.md) — sibling packaging story
