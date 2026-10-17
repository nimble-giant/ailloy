# Project State

Ailloy keeps its own bookkeeping out of your project. Only a mold's rendered
output (blanks, workflows, plugin files) lands in the project tree; everything
ailloy needs to track that output lives in a per-project **state directory**
under your home directory:

```
~/.ailloy/projects/<project-name>-<hash>/
├── installed.yaml   # installed manifest: provenance for recast, quench, uninstall
├── ailloy.lock      # opt-in lock (created by `ailloy quench`)
├── state.yaml       # where blanks/workflows were installed
├── flux/            # persisted project-scoped flux values (anneal / TUI)
├── ores/            # ores installed for this project
├── ingots/          # ingots installed for this project
└── project.yaml     # the project root this directory belongs to
```

`<hash>` comes from the project's absolute path, so every checkout gets its own
state directory and the same checkout always maps to the same one.

## Finding it

```bash
ailloy state path      # print this project's state directory
ailloy state list      # every project with ailloy state, and where it lives
```

## Migrating existing projects

Projects cast with an older ailloy have `.ailloy/` and `ailloy.lock` in the
project tree. The first ailloy command you run in such a project moves
`.ailloy/{installed.yaml,state.yaml,flux,ores,ingots}` and `ailloy.lock` into
the state directory and prints one line saying so. `ailloy state migrate` does
the same on demand.

Migration never overwrites: if the state directory already has a file, the
legacy copy is left in place with a warning. Unknown files in `.ailloy/` are
left alone, and the directory is removed only once empty. Nothing is migrated
when ailloy runs in your home directory, where `~/.ailloy/` and `~/ailloy.lock`
are the global state.

## Keeping state in the project

Set `AILLOY_PROJECT_STATE_DIR` to choose the location. A relative value
resolves against the project root, so

```bash
export AILLOY_PROJECT_STATE_DIR=.ailloy
```

keeps everything, the lock included (as `.ailloy/ailloy.lock`), inside the
project, for teams that commit the lock and run `ailloy quench --verify` in CI.

## Global scope is unchanged

`--global` operations still use `~/.ailloy/installed.yaml`, `~/ailloy.lock`,
`~/.ailloy/ores/`, `~/.ailloy/ingots/` and `~/.ailloy/flux/`.
