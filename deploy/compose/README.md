# deploy/compose

`compose.yaml` starts throwaway PostgreSQL 14, 15, 16, 17 and 18 servers for local development and manual testing. Automated integration tests do not use it: they start their own containers with testcontainers-go (`moon run core:integration`).

The production-style Compose HA topology (Patroni, etcd, HAProxy, PgBouncer, pgBackRest) is something Tuskline will *generate* in milestone M5. See `docs/onboarding-and-deploy.md`.
