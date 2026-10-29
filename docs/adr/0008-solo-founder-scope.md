# 0008. Solo founder plus AI tooling

- Status: Accepted
- Date: 2026-10-28

## Context

One person builds Tuskline, with AI coding agents. Scope creep across eight areas is the largest risk.

## Decision

Three releases with an explicit cut list:

- **R1, developer door (0–5 months).**
- **R2, fleet door and Cloud (5–10 months).**
- **R3, guarded actions (10–18 months).**

Durations are planning estimates, not commitments.

Cut: gRPC API (REST only), Terraform provider, WASM plugin host, VS Code extension. Also deferred:

- PGO, StackGres, Barman and WAL-G integrations.
- PagerDuty and Opsgenie (webhooks cover them).
- Workload replay, thin clones, forecasting, cost insight, data compare and masking.
- SAML and SCIM.

Not building: our own HA manager, backup engine, Kubernetes operator, or mobile apps.

## Consequences

- AI tooling does the mechanical work: the Svelte port, SQL packs across versions, tests, docs and cloud snippets. The founder owns design, review and the safety-critical paths (credentials, actions, provisioning). That is why `AGENTS.md` and `docs/roadmap.md` are detailed.
- Sequence work by dependency: core and onboarding first, history before fleet, read-only views before actions.
- Ship every two weeks. A CI matrix (PostgreSQL 14–18 in containers) plus a nightly run against low-cost managed instances stands in for a QA team.
- Every new integration must remove a support question, not add one. Recruit a co-maintainer once R2 ships.
