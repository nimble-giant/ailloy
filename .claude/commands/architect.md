# System Design Architect

## Purpose

This command takes a high-level idea and produces a comprehensive, actionable system design. It activates the **system-design** skill (`.claude/skills/system-design.md`), enters plan mode, analyzes the current codebase, identifies gaps requiring user decisions, and outputs a structured architecture document with implementation plan.

## Command Name

`architect`

## Skill Dependency

This command requires the **system-design** skill. Before executing the workflow below, load and apply the full persona, competencies, design principles, codebase discovery protocol, and gap analysis protocol defined in `.claude/skills/system-design.md`. All architectural reasoning during this command must follow that skill's principles and behavioral patterns.

## Invocation Syntax

```bash
/architect <description of what you want to build>
/architect [flags] <description>
```

### Flags

- `--scope <scope>`: Constrain design scope (`service`, `module`, `system`, `platform`) (default: inferred from description)
- `--style <style>`: Architectural style preference (`event-driven`, `request-driven`, `hybrid`) (default: inferred)
- `--env <environment>`: Target deployment environment (`kubernetes`, `serverless`, `bare-metal`, `hybrid`) (default: inferred from codebase)
- `--constraints <list>`: Comma-separated constraints (`fips`, `hipaa`, `sox`, `pci-dss`, `fedramp`, `air-gapped`)
- `--adr`: Generate Architecture Decision Records for each major decision
- `--prompt`: Enable extended interactive mode (ask more questions, present more options)

### Examples

```bash
# Design a new microservice
/architect a user notification service that supports email, SMS, and push notifications with delivery guarantees

# Design with compliance constraints
/architect --constraints fips,fedramp a secrets management pipeline for multi-tenant SaaS

# Design a module within the existing system
/architect --scope module an audit logging subsystem that captures all API mutations with tamper-proof storage

# Event-driven architecture
/architect --style event-driven a real-time data pipeline that ingests IoT sensor data, applies anomaly detection, and triggers alerts

# Full interactive mode with ADRs
/architect --prompt --adr a multi-region active-active deployment strategy for our API layer
```

---

## Workflow

### Phase 1: Enter Plan Mode

**First:** Immediately enter plan mode. All system design work happens in plan mode to allow review before any implementation begins.

### Phase 2: Codebase Discovery

Execute the **Codebase Discovery Protocol** from the system-design skill:

1. **Language & Framework Inventory** — Scan for dependency manifests, identify languages, frameworks, and build systems
2. **Dependency Analysis** — Read manifests to map the library ecosystem, flag infrastructure-related dependencies
3. **Infrastructure Patterns** — Check for containers, orchestration, CI/CD, IaC, service mesh configs
4. **Existing Architecture** — Map service/module boundaries, communication patterns, data stores, deployment topology, observability tooling
5. **Configuration & Conventions** — Identify config patterns, code standards, API specs, documentation patterns

Produce a concise summary of what already exists. This summary becomes Section 2.1 of the design document.

### Phase 3: Gap Analysis & Decision Points

Execute the **Gap Analysis Protocol** from the system-design skill:

For each aspect of the proposed design that requires technology or pattern choices **not already present in the codebase**:

1. **Identify the gap** clearly
2. **Present 2-3 options** with concrete trade-offs
3. **Ask the user to decide** using the AskUserQuestion tool
4. **Record the decision** for inclusion in the Decisions Log

Group related decisions together to minimize back-and-forth. Never silently assume technology choices.

### Phase 4: System Design Document

After all decisions are gathered, produce the design document using this structure:

```markdown
# System Design: [Title]

## 1. Overview

[2-3 paragraph executive summary: what is being built, why, and the key architectural approach]

## 2. Context & Constraints

### 2.1 Current State
[Summary of relevant existing architecture from Phase 2]

### 2.2 Requirements
- **Functional**: [Derived from the user's description]
- **Non-Functional**: [Performance, availability, scalability targets]
- **Constraints**: [Compliance, budget, timeline, team expertise]

### 2.3 Assumptions
[Explicit list of assumptions made during design]

## 3. Architecture

### 3.1 High-Level Architecture

[Mermaid diagram showing system components and their interactions]

### 3.2 Component Design

For each component:
- **Responsibility**: Single-responsibility description
- **Technology**: Chosen stack with rationale
- **Interfaces**: APIs exposed and consumed
- **Data Ownership**: What data this component owns
- **Scaling Strategy**: How this component scales

### 3.3 Data Architecture

- **Data Model**: Key entities and relationships
- **Storage Strategy**: Which data store, why, partitioning approach
- **Data Flow**: How data moves through the system
- **Consistency Model**: Strong/eventual, where and why

### 3.4 Communication Patterns

- **Synchronous**: Which interactions are request/response, protocol choice
- **Asynchronous**: Which interactions are event-driven, broker choice
- **API Contracts**: Versioning strategy, schema evolution

## 4. Cross-Cutting Concerns

### 4.1 Security
- Authentication & authorization approach
- Network security (mTLS, network policies)
- Secrets management
- Data encryption (at rest, in transit)

### 4.2 Observability
- Metrics, traces, and logs strategy
- SLOs and alerting approach
- Dashboarding and debugging workflows

### 4.3 Reliability
- Failure modes and mitigation
- Retry and circuit breaker policies
- Backup and disaster recovery
- Rollback strategy

### 4.4 Deployment
- CI/CD pipeline changes
- Rollout strategy (canary, blue-green, progressive)
- Feature flagging approach
- Environment promotion path

## 5. Decisions Log

| # | Decision | Options Considered | Choice | Rationale |
|---|----------|--------------------|--------|-----------|
| 1 | [Decision] | [A, B, C] | [Choice] | [Why] |

## 6. Implementation Plan

### Phase 1: Foundation
- [ ] [Task with clear deliverable]

### Phase 2: Core Logic
- [ ] [Task with clear deliverable]

### Phase 3: Integration
- [ ] [Task with clear deliverable]

### Phase 4: Hardening
- [ ] [Task with clear deliverable]

## 7. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| [Risk] | High/Med/Low | High/Med/Low | [Strategy] |

## 8. Open Questions

- [Any remaining questions that need stakeholder input]
```

### Phase 5: Plan Review

Present the complete design document via ExitPlanMode for user review.

After approval, if the `--adr` flag was specified, generate individual ADR files in `docs/adr/` (or the project's existing ADR location) following:

```markdown
# ADR-NNN: [Title]

## Status
Proposed

## Context
[Why this decision was needed]

## Decision
[What was decided]

## Consequences
[Trade-offs accepted]
```

---

## Design Scope Guidelines

### `--scope service`
Single service: API design, data model, internal architecture, deployment config. Assumes an existing platform.

### `--scope module`
Module within an existing service: package structure, interfaces, internal patterns. No new infrastructure.

### `--scope system`
System of multiple services: service boundaries, communication, shared infrastructure, data ownership. Default for most requests.

### `--scope platform`
Full platform: infrastructure provisioning, cluster setup, networking, shared services (auth, observability, CI/CD), multi-environment strategy.

---

## Error Handling

- **Empty/minimal codebase**: Design from scratch, ask more questions about technology preferences
- **Vague description**: Ask clarifying questions before proceeding
- **Conflicting constraints**: Identify the conflict and ask for resolution
- **Scope too large**: Propose breaking into phases, ask which to design first

## Integration Notes

This command integrates with:
- **`/preflight`**: Validate the implementation plan against INVEST criteria
- **`/create-issue`**: Generate issues from implementation plan tasks
- **`/start-issue`**: Begin implementing individual tasks from the plan

## Recap

`/architect` is the command interface for the **system-design** skill. It provides a structured workflow — plan mode, codebase discovery, gap analysis with user decisions, and a comprehensive design document — while the skill provides the architectural expertise, design principles, and decision-making framework that drive the output.
