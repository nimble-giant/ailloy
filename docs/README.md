<div align="center">

<img src="../.assets/Friendly Ailloy with Glowing Orb.png" alt="Ailloy Documentation" width="150">

# Ailloy Documentation

Comprehensive guides for the package manager for AI instructions

</div>

These guides teach you how to install, create, package, and share AI workflow packages with Ailloy. For a quick overview of the project, see the [main README](../README.md). Every page here is also available offline with `ailloy docs <topic>`.

## Pick your path

| You want to… | Start here | Then |
|--------------|------------|------|
| **Use** a mold someone else wrote | [Getting Started](getting-started.md) | [Glossary](glossary.md) → [Configuration Wizard](anneal.md) |
| **Write** your own mold | [Your First Mold](topics/tutorials/first-mold.md) | [Blanks](blanks.md) → [Flux Variables](flux.md) |
| **Share** molds with a team | [Remote Molds](foundry.md) | [Packaging Molds](smelt.md) → [Mold Dependencies](mold-dependencies.md) |
| Map Ailloy onto tools you know | [Coming from Helm?](helm-users.md) | [Glossary](glossary.md) |

New to the metalworking vocabulary? The [Glossary](glossary.md) translates every term (mold, flux, cast, forge, …) into plain English on one page.

## 1. Learn the basics

- [Getting Started](getting-started.md) — Install ailloy and cast your first mold in five minutes
- [Glossary](glossary.md) — Every Ailloy term and command in plain English, with npm and Helm equivalents
- [Your First Mold](topics/tutorials/first-mold.md) — Build, preview, and cast a mold from scratch
- [Coming from Helm?](helm-users.md) — Concept map, command equivalences, and what's new

## 2. Author molds

- [Blanks](blanks.md) — What blanks are and how to create commands, skills, and workflows
- [AGENTS.md](agents-md.md) — Tool-agnostic agent instructions in molds
- [Flux Variables](flux.md) — Configure blanks with variables, schemas, and value layering
- [Ingots](ingots.md) — Create and use reusable template components
- [Ore](ore.md) — Reusable flux partials for opt-in business-logic data structures
- [Migrating to Ore](ore-migration.md) — Move inline flux schemas into reusable ores

## 3. Share and distribute

- [Remote Molds](foundry.md) — Resolve molds from git repositories ([reference format](foundry.md#mold-references)), manage foundry indexes, and use the [interactive `ailloy foundries` TUI](foundry.md#interactive-tui)
- [Mold Dependencies](mold-dependencies.md) — Compose molds; transitive resolution, conflict handling, lockfile, uninstall cascade
- [Packaging Molds](smelt.md) — Package molds into distributable tarballs or binaries
- [Plugins](plugin.md) — Generate plugins from molds (currently Claude Code)
- [Cast as a Claude Code Plugin](cast-claude-plugin.md) — Install a mold straight into `.claude/plugins/`

## 4. Operate and maintain

- [Uninstalling a casted mold](foundry.md#uninstalling-a-casted-mold) — Safely remove what `cast` wrote using the install manifest
- [Configuration Wizard](anneal.md) — Interactive wizard for flux variable configuration
- [Validation](temper.md) — Lint and validate mold and ingot packages
- [Linting](assay.md) — Lint rendered AI instruction files against best practices
- [Cache Management](cache.md) — Clear cached molds and foundry indexes
- [Extensions](topics/extensions/users.md) — Optional add-ons such as the rich docs browser
