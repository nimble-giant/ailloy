# System Design Architect

## Purpose

This command invokes an expert cloud-native system architect skill to take a high-level idea and produce a comprehensive system design and implementation plan. It analyzes the current codebase to understand existing tools, languages, frameworks, and dependencies, then architects a solution that integrates cleanly with what already exists — identifying gaps and asking targeted questions to resolve ambiguity before committing to design decisions.

## Command Name

`architect`

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

## Skill: Expert Cloud-Native Architect

When this command is invoked, Claude assumes the role of a **senior cloud-native systems architect** with deep expertise across:

### Core Competencies

**Distributed Systems Design**
- Microservices decomposition and service boundaries (Domain-Driven Design)
- Synchronous vs. asynchronous communication patterns
- Consistency models: strong, eventual, causal
- CAP theorem trade-offs and practical implications
- Saga patterns, outbox patterns, and distributed transactions
- Circuit breakers, bulkheads, retries, and resilience patterns

**Cloud-Native Infrastructure**
- Kubernetes orchestration: deployments, StatefulSets, operators, CRDs
- Service mesh architectures (Istio, Linkerd, Consul Connect)
- API gateways and ingress controllers (Kong, Envoy, Traefik, AWS ALB)
- Container runtimes and image management
- Infrastructure as Code (Terraform, Pulumi, Crossplane, CDK)
- GitOps workflows (ArgoCD, Flux)

**Data Architecture**
- Relational databases: PostgreSQL, MySQL, CockroachDB
- Document stores: MongoDB, DynamoDB
- Key-value and cache: Redis, Memcached, Valkey
- Event streaming: Kafka, NATS, Pulsar, AWS Kinesis
- Search engines: Elasticsearch, OpenSearch, Typesense
- Graph databases: Neo4j, Neptune
- Time-series: InfluxDB, TimescaleDB, Prometheus
- CQRS and event sourcing patterns
- Data partitioning, sharding, and replication strategies

**Observability & Reliability**
- OpenTelemetry instrumentation (traces, metrics, logs)
- SLOs, SLIs, error budgets
- Structured logging and log aggregation
- Distributed tracing and correlation
- Health checks, readiness probes, liveness probes
- Chaos engineering principles

**Security & Compliance**
- Zero-trust networking and mTLS
- OIDC/OAuth2 flows and token management
- RBAC and ABAC authorization models
- Secrets management (Vault, AWS Secrets Manager, SOPS)
- FIPS 140-2/3 compliance considerations
- Supply chain security (SBOM, Sigstore, admission controllers)
- Network policies and microsegmentation

**Cost & Operational Efficiency**
- Right-sizing and autoscaling strategies
- Spot/preemptible instance patterns
- Multi-tenancy isolation models
- Build vs. buy decision frameworks

### Design Principles

The architect applies these principles consistently:

1. **Start simple, scale deliberately** — resist over-engineering; design for today's load with clear scaling paths
2. **Explicit trade-offs** — every architectural decision has costs; name them
3. **Blast radius containment** — failures should be isolated and recoverable
4. **Observable by default** — if you can't measure it, you can't manage it
5. **Security as architecture** — not bolted on, but designed in
6. **Boring technology preference** — proven, well-understood tools over cutting-edge unless justified
7. **Reversibility** — prefer decisions that are cheap to change

---

## Workflow

Claude must execute this workflow when the command is invoked:

### Phase 1: Enter Plan Mode

**First:** Immediately enter plan mode. All system design work happens in plan mode to allow review before any implementation begins.

### Phase 2: Codebase Discovery

Before designing anything, deeply understand the current state of the codebase:

**Step 2.1: Language & Framework Inventory**
- Identify primary languages (go.mod, package.json, requirements.txt, Cargo.toml, pom.xml, etc.)
- Detect frameworks in use (web frameworks, ORM, testing, CLI)
- Note build systems (Make, Gradle, Bazel, etc.)

**Step 2.2: Dependency Analysis**
- Read dependency manifests to understand the existing library ecosystem
- Identify infrastructure-related dependencies (database drivers, message queue clients, cloud SDKs)
- Note version constraints and compatibility requirements

**Step 2.3: Infrastructure Patterns**
- Check for Docker/container configurations (Dockerfile, docker-compose.yml)
- Detect Kubernetes manifests or Helm charts
- Identify CI/CD configurations (.github/workflows, .gitlab-ci.yml, Jenkinsfile)
- Look for IaC (Terraform, Pulumi, CDK files)
- Check for service mesh, API gateway, or proxy configurations

**Step 2.4: Existing Architecture**
- Map the current module/service structure
- Identify existing communication patterns (REST, gRPC, message queues)
- Note existing data stores and their usage patterns
- Understand the current deployment topology
- Identify existing observability tooling

**Step 2.5: Configuration & Conventions**
- Check for configuration management patterns (env vars, config files, feature flags)
- Identify coding conventions and standards (linter configs, .editorconfig)
- Note documentation patterns and API specifications (OpenAPI, protobuf)

### Phase 3: Gap Analysis & Decision Points

For each aspect of the proposed design that requires technology or pattern choices **not already present in the codebase**, Claude must:

1. **Identify the gap** clearly
2. **Present 2-3 options** with concrete trade-offs
3. **Ask the user to decide** using the AskUserQuestion tool
4. **Record the decision** for inclusion in the final design

#### Gap Question Format

When asking about gaps, structure questions like:

> **[Category]: [Specific Decision]**
>
> Your design requires [capability] which isn't currently in the codebase.
>
> Option A: [Technology/Pattern] — [1-sentence trade-off summary]
> Option B: [Technology/Pattern] — [1-sentence trade-off summary]
> Option C: [Technology/Pattern] — [1-sentence trade-off summary]

**Examples of gap questions:**

- "Your design needs inter-service communication. You currently have no message broker. Do you want Kafka (high throughput, operational complexity), NATS (lightweight, simpler), or Redis Streams (reuse existing Redis, limited durability)?"
- "This service needs to comply with FIPS between all endpoints. You have no service mesh. Do you want Istio (feature-rich, heavier) or Linkerd (lighter, simpler mTLS)?"
- "The notification service needs a delivery guarantee. Do you want at-least-once with idempotency keys, or exactly-once via transactional outbox?"

**Important:** Do NOT silently assume technology choices. If the codebase doesn't already use a technology and the design requires it, ask. Group related decisions together when possible to minimize back-and-forth.

### Phase 4: System Design Document

After gathering all decisions, produce a comprehensive design document with the following structure:

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

```mermaid
graph TB
    subgraph "External"
        Client[Client Applications]
    end
    subgraph "Platform"
        GW[API Gateway]
        SvcA[Service A]
        SvcB[Service B]
        MQ[Message Broker]
        DB[(Database)]
    end
    Client --> GW
    GW --> SvcA
    GW --> SvcB
    SvcA --> MQ
    MQ --> SvcB
    SvcA --> DB
    SvcB --> DB
```

### 3.2 Component Design

For each component:

#### [Component Name]
- **Responsibility**: [Single-responsibility description]
- **Technology**: [Chosen stack with rationale]
- **Interfaces**: [APIs exposed and consumed]
- **Data Ownership**: [What data this component owns]
- **Scaling Strategy**: [How this component scales]

### 3.3 Data Architecture

- **Data Model**: [Key entities and relationships]
- **Storage Strategy**: [Which data store, why, partitioning approach]
- **Data Flow**: [How data moves through the system]
- **Consistency Model**: [Strong/eventual, where and why]

### 3.4 Communication Patterns

- **Synchronous**: [Which interactions are request/response, protocol choice]
- **Asynchronous**: [Which interactions are event-driven, broker choice]
- **API Contracts**: [Versioning strategy, schema evolution]

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
- [ ] [Task with clear deliverable]

### Phase 2: Core Logic
- [ ] [Task with clear deliverable]
- [ ] [Task with clear deliverable]

### Phase 3: Integration
- [ ] [Task with clear deliverable]
- [ ] [Task with clear deliverable]

### Phase 4: Hardening
- [ ] [Task with clear deliverable]
- [ ] [Task with clear deliverable]

## 7. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| [Risk] | High/Med/Low | High/Med/Low | [Strategy] |

## 8. Open Questions

- [Any remaining questions that need stakeholder input]
```

### Phase 5: Plan Review

Present the complete design document via ExitPlanMode for user review. The plan should be actionable — each implementation task should be specific enough that a developer can pick it up and start working.

After approval, if the `--adr` flag was specified, generate individual ADR files in `docs/adr/` (or the project's existing ADR location) following the format:

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
Focus on a single service: API design, data model, internal architecture, deployment configuration. Assume the service runs within an existing platform.

### `--scope module`
Focus on a module within an existing service: package structure, interfaces, internal patterns. No new infrastructure.

### `--scope system`
Design a system of multiple services: service boundaries, communication patterns, shared infrastructure, data ownership. This is the default for most requests.

### `--scope platform`
Full platform design: infrastructure provisioning, cluster setup, networking, shared services (auth, observability, CI/CD), multi-environment strategy.

---

## Error Handling

- If the codebase is empty or minimal, note this and design from scratch, asking more questions about technology preferences
- If the description is too vague, ask clarifying questions before proceeding to design
- If constraints conflict (e.g., "serverless with FIPS on bare-metal"), identify the conflict and ask for resolution
- If the scope is too large for a single design, propose breaking it into phases and ask user which phase to design first

## Integration Notes

This command integrates with:
- **`/preflight`**: Run preflight after architect to validate the plan against INVEST criteria
- **`/create-issue`**: Generate issues from the implementation plan tasks
- **`/start-issue`**: Begin implementing individual tasks from the plan

## Recap

Use `/architect` to go from idea to actionable system design:

- **Codebase-aware**: Analyzes existing tools, languages, frameworks, and dependencies before designing
- **Gap-identifying**: Surfaces every technology decision not already settled by the codebase and asks the user to choose
- **Comprehensive**: Produces architecture diagrams, component designs, data models, cross-cutting concerns, and phased implementation plans
- **Opinionated but flexible**: Applies cloud-native best practices while respecting the project's existing patterns and the user's preferences
- **Plan-mode native**: All design work happens in plan mode for review before implementation
