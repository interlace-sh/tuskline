# Security policy

Tuskline will hold database credentials and run statements against production, so we take security reports seriously.

## Reporting a vulnerability

**Do not open a public issue.** Use GitHub's private vulnerability reporting ("Report a vulnerability" under the repository's Security tab). Include steps to reproduce, the affected version or commit, and the impact.

Expect an acknowledgement within 5 working days. We will agree a disclosure date with you, normally within 90 days, and credit you unless you prefer otherwise.

## Supported versions

Pre-alpha: only `main` is supported. A supported-versions table will be added at the first release.

## Scope

In scope:

- Credential handling.
- Bypassing environment policy, for example writing to a read-only production environment.
- Authentication and RBAC.
- The audit log.
- Agent ↔ server mTLS.
- SQL injection through Tuskline-generated SQL.
- Supply chain: releases, images, SBOMs.

The security design is in [docs/security.md](docs/security.md).
