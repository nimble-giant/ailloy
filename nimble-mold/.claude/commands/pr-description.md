# pr-description

Generate a pull request description for the current branch changes.

## Usage

```
claude pr-description
```

## Title

The title of the pr should adhere to our git conventional commit guidelines to make our titles consistent.

## Description

This command generates a comprehensive PR description by comparing the current branch to the head of `main` using `git diff main...`. The PR description is saved as a markdown file in the `/pr-descriptions` folder.

If the `/gh-issue` command was used and Claude knows what {{scm_provider}} issue the branch is for, the PR description should begin with `closes #<github-issue-number>`.

## Output

- **Location**: `/pr-descriptions` folder
- **File naming**: `pr-description-YYYY-MM-DD-HH-MM-SS.md` or descriptive name based on feature
- **Format**: Markdown with clear section headers

## Required Sections

### Summary

Brief description of the PR purpose (1-3 sentences)

### Changes

List of specific changes made (bullet points)

### UAT

Instructions for how to test the changes

## Example Structure

```markdown
# feat(web): add control families to compliance section

closes #1234

## Summary
This PR adds support for Control Families in the Judge web UI, allowing users to organize compliance controls into logical groupings. Control families create a hierarchy between frameworks and controls.

## Changes
- Added Control Families menu item in Compliance section
- Created new page and components for viewing/managing control families
- Added family column to Controls table
- Updated GraphQL queries to include control family information
- Added hooks for querying, creating, and updating control families

## UAT
1. Navigate to Compliance > Families
2. Create a new control family
3. Create a control and assign it to the family
4. Verify the control shows up in the family's detail page
```

## Notes

- Only generate when explicitly requested with `claude pr-description`, `/pr-description`, or when directly asked to "create a PR description"
- Always compare current branch to main branch for complete context
- Do not include Claude Code attribution or generated-by notices in PR descriptions
