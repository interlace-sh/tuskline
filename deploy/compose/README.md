# deploy/compose

`moon run :up` builds and starts throwaway PostgreSQL 14, 15, 16, 17 and 18 servers for local development and manual testing. `moon run :down` stops them and deletes their volumes, so the next `:up` runs the setup script again. Each image is the official PostgreSQL image plus the PGDG packages `hypopg`, `pg_stat_kcache` and `pg_wait_sampling`. Automated integration tests do not use it: they start their own stock `postgres` containers with testcontainers-go (`moon run core:integration`).

The production-style Compose HA topology (Patroni, etcd, HAProxy, PgBouncer, pgBackRest) is something Tuskline will *generate* in milestone M5. See `docs/onboarding-and-deploy.md`.
