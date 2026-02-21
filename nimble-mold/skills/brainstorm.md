# Skill: Brainstorm

## What This Skill Does

This skill applies structured brainstorming techniques to systematically analyze an idea for feasibility, scope, and value. It draws from the [UNC Writing Center's brainstorming methodology](https://writingcenter.unc.edu/tips-and-tools/brainstorming/), adapted for evaluating software and product ideas.

The skill produces a saved analysis report — not implementation. The user decides independently whether and when to act on the findings.

## When to Use This Skill

Use this skill whenever the user wants to brainstorm, evaluate, or pressure-test an idea. This includes requests like:
- "Brainstorm X"
- "Is this idea worth building?"
- "Analyze whether we should do X"
- "Help me think through this idea"
- "What do you think about building X?"

## Workflow

Follow these steps in order:

### Step 1: Enter Plan Mode

Enter plan mode immediately. The brainstorm output IS the plan.

### Step 2: Research

Before producing the brainstorm, gather relevant context:

- **Codebase context**: If the idea relates to the current project, explore the relevant code, architecture, and existing patterns
- **Web research**: Search for prior art, existing tools, competing solutions, and relevant technical concepts
- **Feasibility signals**: Look for libraries, APIs, or frameworks that would support or complicate the idea

Do not fabricate comparisons or claim prior art exists without verifying. If you cannot find relevant prior art, say so.

### Step 3: Execute All Seven Brainstorming Phases

Work through every phase in order. Each phase builds on the previous one. Do not skip phases.

#### Phase 1: Freewriting — Unpack the Idea

Let the idea breathe. Without filtering or judging, expand on what the user said. Restate it in different ways. Surface implicit assumptions, unstated goals, and adjacent possibilities.

Capture:
- What the idea is really about beneath the surface
- What problems it might solve (stated and unstated)
- Who might care about this and why
- What emotions or frustrations drive it
- Adjacent ideas that come to mind

Output as a short, candid narrative paragraph. No bullet points — flowing text.

#### Phase 2: Cubing — Examine from Six Angles

Analyze the idea from six distinct perspectives:

| Angle | Prompt |
|---|---|
| **Describe** | What is this idea, concretely? What does it do, who uses it, and what does the output look like? |
| **Compare** | What existing tools, products, or approaches are similar? How does this differ? |
| **Associate** | What does this idea connect to? What trends, technologies, or patterns does it relate to? |
| **Analyze** | What are the moving parts? What are the dependencies, risks, and unknowns? |
| **Argue For** | What is the strongest case for pursuing this? What value does it create? |
| **Argue Against** | What is the strongest case against? What could go wrong, or why might this fail? |

Present each angle as a short paragraph (2-4 sentences).

#### Phase 3: Journalistic Questions — Establish the Facts

Ground the idea using the six journalistic questions. Answer each with specifics:

- **Who** — Users, stakeholders, affected parties. Who builds it? Who maintains it?
- **What** — What exactly is being built or changed? Inputs and outputs?
- **When** — When would this be needed? Urgency? Phases?
- **Where** — Where does this live? What systems, platforms, or environments?
- **Why** — Why does this matter? Underlying need or pain point?
- **How** — How would this work at a high level? Basic mechanism or architecture?

#### Phase 4: Feasibility, Scope & Value Assessment

Synthesize Phases 1-3 into a structured evaluation:

- **Feasibility**: Rate as High/Medium/Low with 1-2 sentence justification. Consider technical complexity, existing tools, skill requirements, infrastructure demands.
- **Scope**: Classify as Small (days)/Medium (weeks)/Large (months+) with 1-2 sentence justification. Consider components, integrations, testing, documentation.
- **Predicted Value**: Rate as High/Medium/Low with 1-2 sentence justification. Consider problem severity, audience size, pain frequency, strategic alignment.

#### Phase 5: Hypothesis Extraction

Distill into a single, testable hypothesis:

> **If** [we build/change X], **then** [Y will happen], **because** [Z is true about users/systems/the problem].

The hypothesis must be specific enough to be proven wrong, grounded in prior analysis, and focused on core value.

#### Phase 6: Proposed Solutions

Propose 2-4 high-level solution approaches (directional sketches, not implementation plans). For each:

- **Approach name**: Short descriptive label
- **Summary**: 1-3 sentences at a conceptual level
- **Key tradeoff**: Main advantage vs. main risk or cost
- **First question to answer**: Most important unknown to resolve

Order from most conservative to most ambitious.

#### Phase 7: Verdict

Deliver a direct, honest assessment — one of:

- **Worth pursuing** — Sound, feasible, and valuable. Recommend a specific next step.
- **Worth exploring further** — Has promise but key questions remain. Identify what needs answering.
- **Not recommended** — Fundamental problems exist. Explain clearly without hedging. Suggest alternatives if applicable.

Do not soften a negative verdict.

### Step 4: Present the Plan

Present the complete brainstorm analysis via `ExitPlanMode` using this structure:

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

### Step 5: Save the Report and Stop

After user approval, save the brainstorm report to:
- `.claude/plans/` if a `.claude/` directory exists in the project root
- `~/.claude/plans/` otherwise

Create the `plans/` subdirectory if it does not exist.

Name the file `brainstorm-{slug}.md` where `{slug}` is a kebab-case summary of the idea (3-5 words max).

**Stop after saving.** Do not begin implementation or prompt toward it.

## Rules

- Always enter plan mode. The brainstorm output IS the plan.
- Do not skip phases. Every phase must appear in the output.
- Be direct. Avoid hedge words like "potentially," "might," "could possibly."
- Be honest. If the idea is bad, say so in the verdict.
- Stay grounded. Every claim should trace back to something concrete from the analysis.
- Keep it concise. Thorough but not verbose. The entire brainstorm should be scannable.
- **No implementation.** This skill produces documentation and analysis only. It must never write application code, create features, or modify project source files.
- **Stop after the report.** Do not ask "should I start building this?" or otherwise prompt toward implementation.
