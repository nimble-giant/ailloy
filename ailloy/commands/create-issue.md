# create-issue
description: This command is used to instruct Claude to generate GitHub issues in the specific house style and...

```
bash
/create-issue [flags] <description>

```

## Examples
### Example 1 (feature)

```markdown
# feat(web): add ssp section to product grid cards

Add a new **System Security Plan (SSP)** section to each product card in the product grid. This section should:

- Display whether an SSP exists for the product
- Include a button labeled "Analyze" to trigger SSP analysis

Also update the glossary to define:
- What "SSP" is
- What "Analyze" means in this context
## Instructions for Claude

When this command is invoked, you must:

1. Immediately enter plan mode when this command is invoked
2. Parse the user input to extract flags and issue description
3. Format the GitHub issue using the exact markdown structure below
4. Use the ExitPlanMode tool to present the formatted issue as the plan
5. Wait for user approval before proceeding
6. After approval:
7. Execute the GitHub CLI commands to create the issue with configured settings


## Workflow

### Phase 1: Plan Mode

1. **Parse Input**:
   - Extract flags from command (e.g., `-b`, `--board`, `-l`, `--label`)
   - Extract issue description from remaining input
   - Note: No defaults are applied - only use explicitly provided flags
2. **Format Issue**: Create GitHub issue using the exact markdown format above
3. **Present Plan**: Use `ExitPlanMode` tool with the formatted issue as the plan, including parsed settings
4. **Wait for Approval**: Do not proceed until user explicitly approves

### Phase 2: Issue Creation

#### Mode A: Default Mode (No Prompting)

1. **Extract Components**:
   - Title: First line of the plan (the `# title` line)
   - Body: Everything after the title
2. **Create Issue**:
   - Basic: `gh issue create --title "<title>" --body "<body>"`
   - With labels: `gh issue create --title "<title>" --body "<body>" --label "label1,label2"`
3. **Add to Project Board** (if specified):
   - Only if `-b` or `--board` flag was provided
   - Execute: `gh issue edit <issue-number> --add-project "<project-name>"`
4. **Confirm Success**: Report the created issue URL to user

#### Mode B: Interactive Mode (--prompt flag)

If `--prompt` flag is detected:

1. **Ask for Board**: "Would you like to add this issue to a project board? If so, which one?"
2. **Ask for Labels**: "Would you like to add any labels to this issue? (comma-separated)"
3. **Create Issue**: Same as Mode A but with user-provided values
4. **Add Board/Labels**: Apply user-specified board and labels


## GitHub CLI Commands

```bash
# Create a basic issue
gh issue create --title "feat(web): add new feature" --body "Description..."
# Create issue with labels
gh issue create --title "fix(api): resolve bug" --body "Description..." --label "bug,priority:high"
# Add to project board after creation (if your team uses boards)
gh issue edit <issue-number> --add-project "<project-name>"

# List available project boards
gh project list --owner testifysec
# Step 1: Create issue WITHOUT project assignment
gh issue create --title "feat(web): add new feature" --body "Description..."
# Step 2: Add to project board using project NAME (not number)
gh issue edit <issue-number> --add-project "Engineering"
# Common project names:
# - "{{default_board}}" (default)
# - "Tech Debt"
# - "Compliance" 
# - "Webhooks"
# - "Reporting"

```


## Configuration

This command reads from `.ailloy/ailloy.yaml` for default values.
