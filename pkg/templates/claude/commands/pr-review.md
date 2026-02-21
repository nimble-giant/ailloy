# PR Review

## Purpose

This command helps reviewers conduct comprehensive code reviews efficiently. It analyzes PR changes, generates review feedback, and can either output a markdown review file (silent mode) or post comments directly to the PR (interactive mode with plan approval).

## Command Name

`pr-review`

## Invocation Syntax

```bash
/pr-review <pr-number> [--silent-mode|-s] [--focus=<area>]
/pr-review <pr-url> [--silent-mode|-s] [--focus=<area>]
/pr-review [--silent-mode|-s] [--focus=<area>]
```

**Examples:**

```bash
# Silent mode (default) - generates markdown review file
/pr-review 1234
/pr-review 1234 --silent-mode
/pr-review 1234 -s

# Interactive mode - posts comments to PR after plan approval
/pr-review 1234 --silent-mode=false

# Focus on specific areas
/pr-review 1234 --focus=security
/pr-review 1234 --focus=performance
/pr-review 1234 --focus=testing

# Auto-detect current PR
/pr-review --silent-mode=false
```

**Flags:**

- `--silent-mode` / `-s`: Generate markdown review file only (default: true)
- `--silent-mode=false`: Interactive mode with plan approval and PR commenting
- `--focus=<area>`: Focus review on specific areas (security, performance, testing, architecture, etc.)

## Workflow Overview

Claude must execute this workflow when the command is invoked:

1. **Parse Flags**: Determine silent mode and focus areas
2. **Fetch PR Information**: Get PR details, diff, and context
3. **Analyze Code Changes**: Comprehensive code review analysis
4. **Generate Review**: Create structured review feedback
5. **Output Mode**: Either save markdown file (silent) or enter plan mode (interactive)

## Phase 1: Flag Parsing and Setup

### Step 1.1: Parse Command Flags

Default behavior: `--silent-mode=true`

```javascript
// Parse user input for flags
const flags = {
  silentMode: true,  // Default to silent mode
  focus: null,       // No specific focus by default
  prNumber: null,
  prUrl: null
};

// Check for explicit flags in user input
if (input.includes('--silent-mode=false')) flags.silentMode = false;
if (input.includes('--focus=')) flags.focus = extractFocusArea(input);
```

### Step 1.2: Determine PR Number

If no PR number provided:

```bash
# Get current branch
CURRENT_BRANCH=$(git branch --show-current)

# Find PR for current branch
{{pr_find_for_branch_cmd}}
```

## Phase 2: PR Information Gathering

### Step 2.1: Fetch PR Details

```bash
# Get comprehensive PR information
{{pr_view_full_cmd}}

# Get PR diff with context
{{pr_diff_cmd}}

# Get file list and stats
{{pr_view_files_cmd}}

# Get existing reviews and comments
{{api_get_pr_reviews_cmd}}
{{api_get_pr_review_comments_cmd}}
```

### Step 2.2: Analyze Changed Files

For each changed file:

```bash
# Get file content before and after (replace <ref> with base or head branch)
{{api_get_file_content_cmd}}

# Get file history and blame info if needed
git log --oneline -n 10 <file-path>
```

## Phase 3: Code Review Analysis

### Step 3.1: General Code Quality

Analyze each changed file for:

**Code Structure:**
- Function/method complexity
- Class design and responsibilities
- Module organization
- Code duplication

**Best Practices:**
- Naming conventions
- Error handling
- Resource management
- Documentation/comments

**Language-Specific Issues:**
- Type safety (TypeScript/Go/Rust)
- Memory management (C++/Rust)
- Async patterns (JavaScript/Python)
- Exception handling

### Step 3.2: Security Analysis

**Security Concerns:**
- Input validation and sanitization
- SQL injection vulnerabilities
- XSS prevention
- Authentication/authorization
- Secrets management
- Dependency vulnerabilities

**Data Handling:**
- Personal data protection
- Encryption requirements
- Data validation
- Access control

### Step 3.3: Performance Analysis

**Performance Issues:**
- Algorithm efficiency
- Database query optimization
- Memory usage patterns
- Network request optimization
- Caching strategies

**Scalability Concerns:**
- Resource utilization
- Concurrency handling
- Load distribution
- Bottleneck identification

### Step 3.4: Testing Analysis

**Test Coverage:**
- Unit test presence and quality
- Integration test coverage
- Edge case handling
- Error condition testing

**Test Quality:**
- Test readability and maintainability
- Mock usage appropriateness
- Test data management
- Assertion clarity

### Step 3.5: Architecture Analysis

**Design Patterns:**
- Pattern appropriateness
- SOLID principles adherence
- Dependency injection
- Separation of concerns

**API Design:**
- RESTful principles
- GraphQL best practices
- Versioning strategy
- Documentation completeness

## Phase 4: Review Generation

### Step 4.1: Categorize Findings

Group findings by severity:

- **Critical**: Security vulnerabilities, data corruption risks
- **Major**: Performance issues, architectural problems
- **Minor**: Code style, documentation improvements
- **Nit**: Formatting, naming suggestions

### Step 4.2: Generate Review Comments

For each finding, create:

```markdown
**Category**: Security/Performance/Testing/Style
**Severity**: Critical/Major/Minor/Nit
**File**: `path/to/file.js:42`

**Issue**: Brief description of the problem

**Details**: Detailed explanation of why this is an issue

**Suggestion**: Specific recommendation for improvement

**Example**: Code example if applicable
```

## Phase 5: Output Generation

### Mode A: Silent Mode (Default)

Generate comprehensive markdown review file:

```markdown
# Code Review: PR #<number> - <title>

**Reviewer**: Claude AI Assistant  
**Date**: <current-date>  
**PR Author**: @<author>  
**Branch**: <head-branch> → <base-branch>

## Summary

- **Files Changed**: X files
- **Lines Added**: +Y
- **Lines Deleted**: -Z
- **Overall Assessment**: ✅ Approve / ⚠️ Needs Changes / ❌ Request Changes

## Key Findings

### Critical Issues (🚨)
[List critical issues requiring immediate attention]

### Major Issues (⚠️)
[List major issues that should be addressed]

### Minor Issues (ℹ️)
[List minor improvements and suggestions]

### Positive Highlights (✅)
[Call out good practices and improvements]

## Detailed Review

### File: `src/components/UserForm.tsx`

#### Security Issues

**🚨 Critical: Input Validation Missing**
- **Line 42**: User input is not validated before database insertion
- **Risk**: SQL injection vulnerability
- **Recommendation**: Add input sanitization using a validation library
```typescript
// Current
const user = await db.query(`INSERT INTO users VALUES ('${userInput}')`);

// Recommended
const user = await db.query('INSERT INTO users VALUES (?)', [sanitize(userInput)]);
```

#### Performance Issues

**⚠️ Major: Inefficient Re-rendering**
- **Lines 15-25**: Component re-renders on every prop change
- **Impact**: Poor user experience with large datasets
- **Recommendation**: Implement React.memo or useMemo for optimization

### File: `src/api/users.js`

#### Testing Issues

**ℹ️ Minor: Missing Edge Case Tests**
- **Function**: `validateEmail`
- **Missing**: Tests for internationalized domain names
- **Recommendation**: Add test cases for unicode domains

## Recommendations

### Immediate Actions Required
1. Fix SQL injection vulnerability in UserForm.tsx
2. Add input validation middleware
3. Implement proper error handling

### Suggested Improvements
1. Add TypeScript types for better type safety
2. Implement caching strategy for user data
3. Add API documentation

### Future Considerations
1. Consider migrating to a more robust validation library
2. Evaluate performance monitoring tools
3. Plan for internationalization support

## Approval Status

**Recommendation**: ⚠️ Request Changes

**Rationale**: While the overall implementation is solid, the security vulnerability in user input handling must be addressed before merging.

---

*Generated by Claude AI Assistant via Ailloy PR Review Template*
*For questions about this review, please reach out to the reviewer*
```

Save this to: `pr-<number>-review-<timestamp>.md`

### Mode B: Interactive Mode

If `--silent-mode=false`, enter plan mode:

### Step 5.1: Enter Plan Mode

Present review findings in plan mode for user approval:

```markdown
# PR Review Plan for #<number>: <title>

## Review Summary
- **Total Issues Found**: X
- **Critical**: Y (require immediate action)
- **Major**: Z (should be addressed)
- **Minor**: A (suggestions for improvement)

## Proposed Comments

### Critical Issue 1
**File**: `src/api/users.js:42`
**Type**: Security Vulnerability
**Comment**: "🚨 **Security Issue**: This code is vulnerable to SQL injection. User input should be sanitized before database queries."

**Post this comment?** [User approval needed]

### Major Issue 2
**File**: `src/components/Form.tsx:15`
**Type**: Performance
**Comment**: "⚠️ **Performance**: This component re-renders unnecessarily. Consider using React.memo to optimize rendering."

**Post this comment?** [User approval needed]

[Continue for all findings...]

## Overall PR Assessment

**Proposed Review Decision**: Request Changes / Approve / Comment

**Summary Comment**: 
"Thanks for this PR! Overall the implementation looks good, but there are a few security and performance issues that should be addressed before merging. I've left detailed comments on specific lines."

**Post summary comment and submit review?** [User approval needed]
```

### Step 5.2: Get User Approval

For each proposed comment, ask:
- "Should I post this comment? (yes/no/modify)"
- If "modify": "How would you like to change this comment?"

### Step 5.3: Execute Approved Comments

If user approves, post comments to PR:

```bash
# Post line-specific review comments
{{api_post_review_cmd}}

# Post general PR comment
{{pr_comment_cmd}}
```

## Focus Area Specializations

### --focus=security

Emphasize:
- Input validation and sanitization
- Authentication and authorization
- Secrets management
- Dependency vulnerabilities
- Data encryption and protection

### --focus=performance

Emphasize:
- Algorithm complexity
- Database query optimization
- Memory usage
- Network efficiency
- Caching strategies

### --focus=testing

Emphasize:
- Test coverage analysis
- Test quality assessment
- Missing test scenarios
- Test maintainability
- CI/CD integration

### --focus=architecture

Emphasize:
- Design patterns
- SOLID principles
- Code organization
- API design
- Scalability considerations

## Review Quality Guidelines

### Comment Tone

**Constructive and Helpful:**
- "Consider using X approach here because..."
- "This could be improved by..."
- "Great implementation! One suggestion..."

**Specific and Actionable:**
- Provide exact line numbers
- Include code examples
- Suggest specific alternatives

**Educational:**
- Explain the "why" behind suggestions
- Link to relevant documentation
- Share best practices

### Comment Templates

**Security Issues:**
```
🚨 **Security**: [Brief description]

This could lead to [specific risk]. Consider [specific solution].

References: [link to security documentation]
```

**Performance Issues:**
```
⚠️ **Performance**: [Brief description]

This approach has O(n²) complexity. For better performance, consider [alternative].

Example: [code snippet]
```

**Best Practice Suggestions:**
```
💡 **Suggestion**: [Brief description]

This follows the pattern well! For even better maintainability, you might consider [enhancement].
```

## Error Handling

### {{scm_provider}} API Issues
- Rate limiting: Implement exponential backoff
- Authentication: Guide user through `{{auth_login_cmd}}`
- Permissions: Verify repository access

### File Analysis Issues
- Large diffs: Focus on most critical files first
- Binary files: Skip or note as non-reviewable
- Deleted files: Focus on impact analysis

### Review Quality Issues
- Complex changes: Ask user for guidance on focus areas
- Unclear context: Request additional information
- Conflicting best practices: Present options with rationale

## Integration Notes

This template integrates with:
- {{scm_provider}} CLI for PR management
- Git for code analysis
- Language-specific linters and analyzers
- Security scanning tools
- Performance profiling tools

## Usage Tips

1. **Preparation**: Ensure you have repository access and context
2. **Focus**: Use focus flags for targeted reviews
3. **Batch Reviews**: Review multiple files systematically
4. **Follow-up**: Monitor PR for author responses and updates
5. **Learning**: Use reviews as opportunities to share knowledge

## Security Considerations

- Respect repository access permissions
- Don't expose sensitive information in comments
- Validate all file paths and content
- Use secure methods for code analysis

## Recap

Use this command to conduct thorough, consistent code reviews. In silent mode (default), it generates comprehensive markdown review files for your records. In interactive mode, it helps you post structured, helpful comments directly to the PR after plan approval, ensuring high-quality feedback while maintaining reviewer control.