# Authoring Your First Mold

A short tutorial that walks through scaffolding a mold, defining flux
variables, writing a blank with template syntax, and casting it into a
project.

## 1. Scaffold

```bash
ailloy mold new my-first-mold
cd my-first-mold
```

`mold new` creates the directory layout, a starter `mold.yaml`, an
empty `flux.yaml`, and a `blanks/` directory ready for content.

## 2. Define a Flux Variable

Open `flux.schema.yaml` and add one variable:

```yaml
- name: project.name
  type: string
  required: true
  description: "Display name shown to users"
```

Set a default in `flux.yaml`:

```yaml
project:
  name: My Project
```

## 3. Write a Blank

Create `blanks/commands/hello.md`:

```markdown
# /hello

Greet the team for {{ .project.name }}.
```

## 4. Preview and Cast

```bash
ailloy forge .                    # see the rendered output
ailloy cast . --set project.name=Atlas
```

For deeper coverage, see `ailloy docs blanks` and `ailloy docs flux`.
