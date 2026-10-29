# Security

Tuskline will hold production credentials and run statements against production. Safety is designed in. The rules agents must follow are in `.cursor/rules/security-prod-safety.mdc`. Report vulnerabilities as described in [../SECURITY.md](../SECURITY.md).

| Area | Design |
|---|---|
| Transport | TLS `verify-full` by default, even where libpq would default to `prefer`, and `PGSSLMODE` cannot override it. A weaker `sslmode` must be set explicitly on the profile and always comes with a warning (`conn.TLSMode.Warning`). SSH tunnels with jump hosts and agent forwarding; client certificates |
| Credentials in DSNs | `conn.Redact` masks passwords in URLs, keyword/value strings and `password`/`sslpassword` parameters. `conn.Profile` keeps the password in an unexported field, so it never reaches JSON, YAML, `%v` or errors. `~/.pgpass` is ignored, as in libpq, when the group or others can read it |
| Cloud auth | Short-lived tokens: AWS RDS IAM, GCP Cloud SQL IAM, Azure Entra ID; no static passwords where avoidable |
| Secrets | Envelope-encrypted at rest. Pluggable providers: HashiCorp Vault / OpenBao, AWS Secrets Manager, GCP Secret Manager, Azure Key Vault, 1Password, Kubernetes secrets; OS keychain on desktop |
| Identity | Local users and OIDC (Google, Entra, Okta, Keycloak) in the open-source server; SAML and SCIM in Enterprise |
| RBAC | Roles scoped to project and environment: **viewer, analyst, operator, admin**. Just-in-time elevation with expiry and a reason |
| Prod safety | Read-only sessions, statement and lock timeouts, row limits on grids, typed confirmation for destructive actions, SQL preview of every action |
| Privacy | Query parameters and result data are never stored unless a policy allows it; PII redacted in captured plans; **no telemetry by default** |
| Audit | Append-only log of logins, prod queries, actions and elevations; export to SIEM |
| Agent | Outbound only, mTLS, one-time enrolment token; credentials never leave the customer network |
| Supply chain | Reproducible builds, checksums, signed SBOMs, distroless images, Dependabot |

## Environment policy defaults (`core/inventory.DefaultPolicy`)

| Environment | Read-only | Statement timeout | Lock timeout | Row limit | Typed confirmation | Max elevation |
|---|---|---|---|---|---|---|
| prod | yes | 30 s | 2 s | 1,000 | yes | 15 min |
| staging | no | 60 s | 5 s | 5,000 | yes | 1 h |
| dev, local | no | none | none | 10,000 | no | n/a |

Weakening any prod default needs an ADR. `TestProdIsReadOnlyByDefault` and `TestProdSessionIsReadOnlyWithTimeouts` guard the defaults.

`conn.Connect` applies the policy (`Policy.Session()`) before it returns a connection: `set_config` for `default_transaction_read_only` and the two timeouts, then it reads them back from `pg_settings`. If a setting did not take, for example behind a pooler that resets session state, it closes the connection and returns `ErrSessionNotApplied`, so it fails closed. A policy never loosens a setting, so a dev session for a role that is read-only by default stays read-only. `default_transaction_read_only` is only a default: a client that sends `BEGIN READ WRITE` can still write. Tuskline's own surfaces must not do that without elevation; server-side enforcement of elevation is M3.6. Integration tests on 14–18 check that a write fails in a prod session and succeeds in dev, and that both timeouts fire.

## Monitoring role

Collectors use `tuskline_monitor`. It has `pg_monitor`, is limited to 5 connections, defaults to read-only transactions, and has timeouts of 5 s (statement), 1 s (lock) and 10 s (idle in transaction). `core/collector` adds a query-time budget on top of that: a pass stops once the budget is spent, and a timeout or an overloaded target doubles the wait before the next pass (up to five minutes). Actions such as cancelling or terminating sessions need a separate, optional `tuskline_operator` role with `pg_signal_backend`. See [onboarding-and-deploy.md](onboarding-and-deploy.md).

## Threats we design against

- Credential leakage through logs, errors, the inventory file or crash reports.
- Accidental writes to prod.
- Privilege escalation through RBAC gaps.
- SQL injection in generated SQL.
- Collector overload on busy servers.
- Provisioning that destroys data.
- A compromised agent: it is scoped to its own clusters and its tokens can be revoked.
- AI features that leak rows or run writes.
