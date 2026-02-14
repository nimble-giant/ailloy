# Brainstorm

## Purpose

This command takes a raw idea or prompt and systematically dissects it using structured brainstorming techniques inspired by the [UNC Writing Center's brainstorming methodology](https://writingcenter.unc.edu/tips-and-tools/brainstorming/). It analyzes the idea for feasibility, scope, and predicted value, extracts a hypothesis, and proposes high-level solutions. If the idea is not viable, it says so directly.

**This command produces supporting artifacts only — not implementation.** The output is documentation, analysis reports, diagrams, and structured findings that help the user decide whether and how to proceed. The user takes implementation as a separate, deliberate act.

## Command Name

`brainstorm`

## Invocation Syntax

```bash
/brainstorm <idea or prompt>
```

### Examples

```bash
/brainstorm a CLI tool that auto-generates changelogs from conventional commits
/brainstorm adding real-time collaboration to our document editor
/brainstorm replacing our REST API with GraphQL
/brainstorm a service that monitors GitHub Actions costs and suggests optimizations
```

## Workflow

When this command is invoked, Claude must:

1. **Enter Plan Mode** immediately
2. Execute all brainstorming phases below against the user's idea
3. Present the complete brainstorm analysis as the plan using `ExitPlanMode`
4. Wait for user approval
5. After approval, save the brainstorm as a markdown report and stop. Do **not** begin implementation. The user decides if and when to act on the findings separately

---

## Brainstorming Methodology

Claude must work through the following phases in order. Each phase builds on the previous one. The methodology draws from UNC's brainstorming framework, adapted for evaluating software and product ideas.

### Phase 1: Freewriting — Unpack the Idea

Let the idea breathe. Without filtering or judging, expand on what the user said. Restate it in different ways. Surface implicit assumptions, unstated goals, and adjacent possibilities. Write freely and let associations flow.

The goal is volume and breadth, not precision. Capture:
- What the idea is really about beneath the surface
- What problems it might solve (stated and unstated)
- Who might care about this and why
- What emotions or frustrations drive it
- Adjacent ideas that come to mind

Output this as a short, candid narrative paragraph. Do not use bullet points for this phase — write it as flowing text.

### Phase 2: Cubing — Examine from Six Angles

Analyze the idea from six distinct perspectives, just as each face of a cube offers a different view:

| Angle | Prompt |
|---|---|
| **Describe** | What is this idea, concretely? What does it do, who uses it, and what does the output look like? |
| **Compare** | What existing tools, products, or approaches are similar? How does this differ? |
| **Associate** | What does this idea connect to? What trends, technologies, or patterns does it relate to? |
| **Analyze** | What are the moving parts? What are the dependencies, risks, and unknowns? |
| **Argue For** | What is the strongest case for pursuing this? What value does it create? |
| **Argue Against** | What is the strongest case against? What could go wrong, or why might this fail? |

Present each angle as a short paragraph (2-4 sentences).

### Phase 3: Journalistic Questions — Establish the Facts

Ground the idea using the six journalistic questions. Answer each with specifics, not generalities:

- **Who** — Who are the users, stakeholders, and affected parties? Who builds it? Who maintains it?
- **What** — What exactly is being built or changed? What are the inputs and outputs?
- **When** — When would this be needed? Is there urgency or a time window? What are the phases?
- **Where** — Where does this live? What systems, platforms, or environments does it touch?
- **Why** — Why does this matter? What is the underlying need or pain point?
- **How** — How would this work at a high level? What is the basic mechanism or architecture?

### Phase 4: Feasibility, Scope & Value Assessment

Synthesize everything from Phases 1-3 into a structured evaluation:

#### Feasibility

Rate as **High**, **Medium**, or **Low** with a 1-2 sentence justification.

Consider:
- Technical complexity and known unknowns
- Availability of existing tools, libraries, or prior art
- Team skill requirements
- Infrastructure and integration demands

#### Scope

Classify as **Small** (days), **Medium** (weeks), or **Large** (months+) with a 1-2 sentence justification.

Consider:
- Number of components and integrations
- Breadth of change across systems
- Testing and validation requirements
- Documentation and rollout needs

#### Predicted Value

Rate as **High**, **Medium**, or **Low** with a 1-2 sentence justification.

Consider:
- Severity of the problem being solved
- Size of the audience affected
- Frequency of the pain point
- Strategic alignment and compounding effects

### Phase 5: Hypothesis Extraction

Distill the brainstorm into a single, testable hypothesis using this format:

> **If** [we build/change X], **then** [Y will happen], **because** [Z is true about users/systems/the problem].

The hypothesis must be:
- Specific enough to be proven wrong
- Grounded in the analysis from prior phases
- Focused on the core value proposition, not implementation details

### Phase 6: Proposed Solutions

Propose 2-4 high-level solution approaches. These are directional sketches, not implementation plans. For each, provide:

- **Approach name**: A short descriptive label
- **Summary**: 1-3 sentences describing the approach at a conceptual level
- **Key tradeoff**: The main advantage vs. the main risk or cost
- **First question to answer**: The most important unknown that must be resolved before this approach could move forward

Order the approaches from most conservative to most ambitious.

### Phase 7: Verdict

Deliver a direct, honest assessment. One of:

- **Worth pursuing** — The idea is sound, feasible, and valuable. Recommend a specific next step.
- **Worth exploring further** — The idea has promise but key questions remain. Identify what needs to be answered before committing.
- **Not recommended** — The idea has fundamental problems. Explain what they are clearly and without hedging. If a related idea would work better, suggest it.

Do not soften a negative verdict. A clear "this won't work because X" is more valuable than a vague "this might have challenges."

---

## Plan Mode Output Format

The plan presented via `ExitPlanMode` must follow this exact structure:

```markdown
# Brainstorm: {short title derived from the idea}

## The Idea

{One-sentence restatement of the user's idea in clear terms.}

## Phase 1: Freewriting

{Flowing narrative paragraph — unfiltered exploration of the idea.}

## Phase 2: Cubing

**Describe**: {paragraph}

**Compare**: {paragraph}

**Associate**: {paragraph}

**Analyze**: {paragraph}

**Argue For**: {paragraph}

**Argue Against**: {paragraph}

## Phase 3: Journalistic Questions

- **Who**: {answer}
- **What**: {answer}
- **When**: {answer}
- **Where**: {answer}
- **Why**: {answer}
- **How**: {answer}

## Phase 4: Assessment

| Dimension | Rating | Justification |
|-----------|--------|---------------|
| Feasibility | {High/Medium/Low} | {1-2 sentences} |
| Scope | {Small/Medium/Large} | {1-2 sentences} |
| Predicted Value | {High/Medium/Low} | {1-2 sentences} |

## Phase 5: Hypothesis

> **If** {X}, **then** {Y}, **because** {Z}.

## Phase 6: Proposed Solutions

### 1. {Approach Name}

{Summary}

- **Key tradeoff**: {advantage} vs. {risk/cost}
- **First question to answer**: {key unknown to resolve}

### 2. {Approach Name}

{Summary}

- **Key tradeoff**: {advantage} vs. {risk/cost}
- **First question to answer**: {key unknown to resolve}

{Repeat for each approach, 2-4 total}

## Phase 7: Verdict

**{Worth pursuing / Worth exploring further / Not recommended}**

{Direct, honest assessment with reasoning and recommended next step.}
```

---

## Research Requirements

Before producing the brainstorm, Claude should gather relevant context:

- **Codebase context**: If the idea relates to the current project, explore the relevant code, architecture, and existing patterns
- **Web research**: Search for prior art, existing tools, competing solutions, and relevant technical concepts
- **Feasibility signals**: Look for libraries, APIs, or frameworks that would support or complicate the idea

Do not fabricate comparisons or claim prior art exists without verifying. If you cannot find relevant prior art, say so.

---

## Rules

- Always enter plan mode. The brainstorm output IS the plan.
- Do not skip phases. Every phase must appear in the output, even if brief.
- Be direct. Avoid hedge words like "potentially," "might," "could possibly." State what you think.
- Be honest. If the idea is bad, say so in the verdict. Do not pad a weak idea with false encouragement.
- Stay grounded. Every claim about feasibility, scope, or value should trace back to something concrete from the analysis.
- Keep it concise. Each phase should be thorough but not verbose. The entire brainstorm should be scannable.
- **No implementation.** This command produces documentation and analysis only — reports, diagrams, structured findings, and supporting artifacts. It must never write application code, create features, modify project source files, or begin building the idea. Implementation is a separate act taken by the user on their own terms.
- **Stop after the report.** After the user approves the plan, save the brainstorm as a markdown file and stop. Do not ask "should I start building this?" or otherwise prompt toward implementation.

## Recap

Use this command to pressure-test an idea before committing to it. The brainstorm applies structured techniques (freewriting, cubing, journalistic questions) to thoroughly explore the idea from multiple angles, then synthesizes everything into a feasibility assessment, hypothesis, and concrete solution proposals. The output is a saved report — documentation and analysis that supports decision-making. It does not produce implementation. The user decides independently whether and when to act on the findings.
