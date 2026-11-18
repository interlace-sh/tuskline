# 0016. Base product first, cluster lifecycle when it exists

- Status: Accepted
- Date: 2026-11-18

## Context

Tuskline is one inventory and one core ([0001](0001-launch-personas.md)). The screens that work against a database you already have are connections, fleet, health, query insights, the SQL editor, migrations, setup and config. Deploy, HA topology and backups drive Patroni, CloudNativePG and pgBackRest ([0007](0007-build-vs-reuse.md)). They only mean something after a connection exists, and only when that connection has an HA API or a backup repository.

Showing those screens for every database makes the workbench look like a cluster manager. Splitting them into a second product would duplicate the inventory, the prod read-only policy and the audit trail.

## Decision

The base product is connections, fleet by environment, health, query insights, the SQL editor, migrations, setup and config. That is the CLI and web app we finish and test.

Cluster lifecycle is deploy, deployments, HA topology and backups. The rail, the command palette and the keyboard shortcuts for that phase appear only when the open connection has an HA engine (Patroni, CloudNativePG, RDS Multi-AZ, Crunchy PGO) or a registered backup repository. A saved cluster with a Patroni URL or a backup repository counts. Demo clusters that already have an engine still show the phase, so the prototype smoke checks keep passing.

`tl provision`, `tl ha` and `tl backup` stay available. The server does not launch Docker or pgBackRest from the browser.

Sign-in is unchanged: the server stays unauthenticated.

## Consequences

- One product, one inventory. There is no second edition.
- A standalone database shows the base rail only.
- M5 and M6 remain the cluster-lifecycle milestone. They are not required to call the base product done.
- Desktop, the agent, alerting and cloud discovery stay later phases of the same product.
