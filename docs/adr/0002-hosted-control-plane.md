# 0002. Hosted control plane from v1

- Status: Accepted
- Date: 2026-10-28

## Context

A solo founder needs revenue to fund development. Open source drives developer adoption, but companies pay for operations they do not want to run themselves.

## Decision

Offer **Tuskline Cloud** from v1 (beta in release 2). It runs the same AGPL server, with published source. Full self-hosting remains first-class and complete.

- Data path: `tl agent` runs in the customer's network and dials out over mTLS. Database credentials never leave it; only metrics, redacted plans and metadata are sent.
- Pricing shape: a free tier for a few clusters, paid tiers by cluster count and retention, and Enterprise for SAML, SCIM, audit streaming and SLAs. Prices are set after the beta.

## Consequences

- The server must be multi-tenant-capable from the start: organisation scoping in the inventory and in RBAC.
- The agent protocol (outbound, mTLS, resumable) is a v1 requirement, not an afterthought.
- Self-hosted OIDC SSO, RBAC and audit log stay free. Paid features are hosting, SAML/SCIM, audit streaming, long retention and SLAs.
