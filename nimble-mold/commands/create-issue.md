# Create Github Issue

## Purpose

This command is used to instruct Claude to generate {{scm.provider}} issues with a clean, actionable, well-structured format. It is optimized for creating consistent {{scm.provider}} issues for engineering planning, with consistent formatting across `feat`, `fix`, `chore`, `docs`, and `epic` types.

## Command Name

`create-issue`

## Invocation Syntax

You can invoke this command with optional flags:

```bash
/create-github-issue [flags] <description>
```

### Flags

- `-b, --board <name>`: Specify project board (default: "{{project.board}}")
- `-l, --label <label>`: Add labels to the issue (can be used multiple times)
- `--prompt`: Enable interactive mode (prompts for board and labels if supported)

### Examples

```bash
# Basic issue creation (no board or labels)
/create-github-issue feat(web): add control family status card

# With project board
/create-github-issue --board "Tech Debt" fix(api): memory leak in cache

# With labels 
/create-github-issue -l bug -l priority:high fix(security): patch vulnerability

# Multiple labels and board
/create-github-issue -b "Backend" -l enhancement -l good-first-issue feat(api): add new endpoint

# Interactive mode for teams with complex configurations
/create-github-issue --prompt feat(web): new feature requiring custom setup
```

Claude must:

1. Immediately enter plan mode when this command is invoked
2. Parse the user input to extract flags and issue description
3. Format the {{scm.provider}} issue using the exact markdown structure below
4. Use the ExitPlanMode tool to present the formatted issue as the plan
5. Wait for user approval before proceeding
6. After approval:
   - If `--prompt` flag is present: Use interactive mode (ask for board and labels if repository supports them)
   - Otherwise: Use parsed flags only (no prompting, no automatic assignments)
7. Execute the {{scm.provider}} CLI commands to create the issue with configured settings

## Plan Mode Output Format

In plan mode, Claude must present the issue using this exact structure:

```markdown
# {type(scope): short description in lowercase}

{Expanded context from user, rewritten as a clear summary. Explain what is being built, changed, or fixed. Be concise but complete. Include motivations and stakeholder context if relevant. Use bullet points where useful.}

## Acceptance Criteria

- [ ] {Itemized criteria, actions, or functional changes that define "done"}
- [ ] {Use checkboxes for each}
- [ ] {Group logically if needed with subheadings}

## Notes (optional)

- {Additional clarifications, decisions, references, or design choices}
```

**Critical**: The title format must be `{type(scope): description}` in all lowercase for {{scm.provider}} CLI compatibility.

---

## Rules & Style Guide

### Title Format

- **Required pattern**: `{type(scope): description}` in all lowercase
- **Supported types**: `feat`, `fix`, `chore`, `docs`, `refactor`, `epic` (see .commitlintrc.yaml for complete list)
- **Scope format**: Parentheses with component name, e.g., `(web)`, `(api)`, `(ssp)`
- **Description**: Brief, actionable summary

### Content Guidelines

- Use standard markdown heading syntax (`#`, `##`)
- Default to "Acceptance Criteria" for features/fixes, "Requirements" for epics
- Use consistent voice, past tense only for bugs
- Always rewrite user input for clarity, grammar, and consistency
- Include context and motivation, not just what to build

### {{scm.provider}} CLI Requirements

- Title must be passed to `--title` parameter
- Body content (everything after title) goes to `--body` parameter
- Project assignment uses `{{issue.add_to_project}}`

---

## Issue Format Examples

### Example 1 (feature)

```markdown
# feat(web): add ssp section to product grid cards

Add a new **System Security Plan (SSP)** section to each product card in the product grid. This section should:

- Display whether an SSP exists for the product
- Include a button labeled "Analyze" to trigger SSP analysis

Also update the glossary to define:
- What "SSP" is
- What "Analyze" means in this context

## Acceptance Criteria

- [ ] Each product card displays SSP status
- [ ] "Analyze" button is present and functional
- [ ] Glossary includes definitions for "SSP" and "Analyze"
```

### Example 2 (chore)

```markdown
# chore(aws): configure ecr scan policy for sandbox vs prod environments

We hit our ECR scan limits in sandbox, which blocked scan visibility before a release. To avoid this:

- Disable **automatic scanning** in sandbox
- Keep it **enabled** in production to meet AWS Marketplace requirements
- Add a manual scan trigger via CLI or CD pipeline stage in sandbox

## Acceptance Criteria

- [ ] Automatic ECR scanning disabled in sandbox
- [ ] ECR scanning enabled in prod remains untouched
- [ ] CD pipeline optionally supports JIT scan step in sandbox
```

---

## Execution Workflow

### Phase 1: Plan Mode

1. **Parse Input**:
   - Extract flags from command (e.g., `-b`, `--board`, `-l`, `--label`)
   - Extract issue description from remaining input
   - Note: No defaults are applied - only use explicitly provided flags
2. **Format Issue**: Create {{scm.provider}} issue using the exact markdown format above
3. **Present Plan**: Use `ExitPlanMode` tool with the formatted issue as the plan, including parsed settings
4. **Wait for Approval**: Do not proceed until user explicitly approves

### Phase 2: Issue Creation

#### Mode A: Default Mode (No Prompting)

1. **Extract Components**:
   - Title: First line of the plan (the `# title` line)
   - Body: Everything after the title
2. **Create Issue**:
   - Basic: `{{issue.create}}`
   - With labels: `{{issue.create_with_labels}}`
3. **Add to Project Board** (if specified):
   - Only if `-b` or `--board` flag was provided
   - Execute: `{{issue.add_to_project}}`
4. **Confirm Success**: Report the created issue URL to user

#### Mode B: Interactive Mode (--prompt flag)

If `--prompt` flag is detected:

1. **Ask for Board**: "Would you like to add this issue to a project board? If so, which one?"
2. **Ask for Labels**: "Would you like to add any labels to this issue? (comma-separated)"
3. **Create Issue**: Same as Mode A but with user-provided values
4. **Add Board/Labels**: Apply user-specified board and labels

## {{scm.provider}} CLI Commands

### Basic Issue Creation

```bash
# Create a basic issue
{{issue.create}}

# Create issue with labels
{{issue.create_with_labels}}

# Add to project board after creation (if your team uses boards)
{{issue.add_to_project}}
```

### Available {{scm.provider}} CLI Options

- `--title`: Issue title (required)
- `--body`: Issue description (required)
- `--label`: Comma-separated list of labels (optional)
- `--assignee`: Assign to specific users (optional)
- `--milestone`: Add to milestone (optional)

**Note**: Project board integration varies by repository configuration. Some teams use {{scm.provider}} Projects, others use different systems, and some don't use boards at all.

```bash
# List available project boards
{{project.list}}

# Step 1: Create issue WITHOUT project assignment
{{issue.create}}

# Step 2: Add to project board using project NAME (not number)
{{issue.add_to_project}}

# Common project names:
# - "{{project.board}}" (default)
# - "Tech Debt"
# - "Compliance" 
# - "Webhooks"
# - "Reporting"
```

{{- if or .ore.status.enabled .ore.priority.enabled .ore.iteration.enabled}}

## Project Field Management

### {{.project.board}} Board Field IDs

- Project ID: `{{.project.id}}`
{{- if .ore.status.enabled}}
- Status Field ID: `{{.ore.status.field_id}}`
{{- end}}
{{- if .ore.priority.enabled}}
- Priority Field ID: `{{.ore.priority.field_id}}`
{{- end}}
{{- if .ore.iteration.enabled}}
- Iteration Field ID: `{{.ore.iteration.field_id}}`
{{- end}}
{{- if .ore.status.enabled}}

**Status Options:**

{{range $key, $opt := .ore.status.options -}}
- {{$opt.label}}{{if $opt.id}}: `{{$opt.id}}`{{end}}
{{end}}

```bash
# Set Status
{{api.base}} graphql --field query='
mutation {
  updateProjectV2ItemFieldValue(
    input: {
      projectId: "{{.project.id}}"
      itemId: "<ITEM_ID>"
      fieldId: "{{.ore.status.field_id}}"
      value: {
        singleSelectOptionId: "<OPTION_ID>"
      }
    }
  ) {
    projectV2Item { id }
  }
}'
```

{{- end}}
{{- if .ore.priority.enabled}}

**Priority Options:**

{{range $key, $opt := .ore.priority.options -}}
- {{$opt.label}}{{if $opt.id}}: `{{$opt.id}}`{{end}}
{{end}}

```bash
# Set Priority
{{api.base}} graphql --field query='
mutation {
  updateProjectV2ItemFieldValue(
    input: {
      projectId: "{{.project.id}}"
      itemId: "<ITEM_ID>"
      fieldId: "{{.ore.priority.field_id}}"
      value: {
        singleSelectOptionId: "<OPTION_ID>"
      }
    }
  ) {
    projectV2Item { id }
  }
}'
```

{{- end}}
{{- end}}

### Flag Parsing Rules

- Short flags can be combined: `-bl bug` equals `-b board -l bug`
- Long flags require equals or space: `--board="My Board"` or `--board "My Board"`
- Flags must come before the issue description
- Unknown flags are treated as part of the description

## Default Behaviors

When no flags are specified, Claude will:

- Create a basic {{scm.provider}} issue with title and description only
- Not add to any project board
- Not apply any labels
- Not prompt for any configuration

This ensures a streamlined experience where issues are created immediately after plan approval.

## Interactive Mode (--prompt)

When the `--prompt` flag is used, Claude will:

- Ask if you want to add the issue to a project board
- Ask if you want to add any labels to the issue
- List available options when relevant

This provides flexibility for users who prefer to configure settings interactively or are unsure about their repository's configuration.

## Error Handling

- If `{{scm.cli}}` command fails, report the error and suggest checking {{scm.provider}} CLI authentication
- If project board doesn't exist, ask user to verify the board name or skip board assignment
- If title format is invalid, reformat according to rules automatically
- If labels don't exist, ask user to verify label names or skip label assignment

## Recap

Use this command for creating {{scm.provider}} issues efficiently:

- **Default mode**: Issues are created immediately after plan approval with sensible defaults
- **Flag overrides**: Use flags when you need specific values (e.g., `-b "Tech Debt" -p P0`)
- **Interactive mode**: Use `--prompt` flag when you need full control or are unsure of settings

The command balances automation with flexibility, defaulting to streamlined execution while allowing interactive configuration when needed.
