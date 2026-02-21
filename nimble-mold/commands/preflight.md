# preflight

Run pre-flight planning for a new task or mini-sprint.

## Usage

```
claude preflight
claude preflight <issue-number>
```

## Description

Executes the Pre-Flight Planning ceremony to ensure proper understanding and alignment before starting implementation.

**Important:** When this command is invoked, Claude should immediately switch to plan mode to properly analyze and plan the task before any implementation begins.

## Steps

**First:** Enter plan mode to properly analyze the task. This ensures thoughtful planning before implementation.

1. **Read & Clarify**
   - Read the issue aloud
   - Capture domain language and user intent
   - Identify key requirements

2. **INVEST Check**
   Evaluate the task against INVEST criteria:
   - **I**ndependent
   - **N**egotiable
   - **V**aluable
   - **E**stimable
   - **S**mall
   - **T**estable

3. **Acceptance Criteria**
   - Express as BDD specs (Given/When/Then format)
   - Ensure measurable outcomes
   - Cover edge cases

4. **Design Sketch**
   If helpful, propose a light design:
   - Mermaid diagram for architecture
   - Bullet points for data flow
   - Test strategy outline

5. **Go/No-Go Decision**
   - Present findings for approval
   - Request refinement if needed

## Example Output

```markdown
## Pre-Flight Analysis: Issue #123

### INVEST Evaluation
- ✅ Independent: Can be completed without other tickets
- ✅ Negotiable: Scope can be adjusted
- ✅ Valuable: Delivers user value
- ✅ Estimable: ~2 days effort
- ✅ Small: Fits in one sprint
- ✅ Testable: Clear pass/fail criteria

### Acceptance Criteria
```gherkin
Given a user has admin privileges
When they navigate to the settings page
Then they should see the "Export Data" button

Given a user clicks "Export Data"
When the export completes
Then they receive a downloadable CSV file
```

### Design Sketch

- New endpoint: `GET /api/export/data`
- Service layer: `DataExportService`
- Test approach: Unit tests for service, integration for endpoint

Ready for go/no-go decision.

```

## Plan Mode Note

After completing the pre-flight analysis and receiving user approval, use the `exit_plan_mode` tool to transition to implementation. This ensures a clear separation between planning and execution phases.
