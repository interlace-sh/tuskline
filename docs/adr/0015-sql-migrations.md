# 0015. SQL migrations are Tuskline's own

- Status: Accepted
- Date: 2026-11-04

## Context

Schema changes need a review, a lock check, and a way to apply a directory of SQL exactly once. External migration runners already do the bookkeeping: a history table, a checksum, and a lock so two applies do not overlap. They also bring a runtime Tuskline would have to shell out to, a second history of what was applied, and scripts the lock linter cannot see.

The migration screen is a review of SQL. The runner's job is to apply that SQL and remember what it applied.

## Decision

`tl migrate` reads the `.sql` files in one directory and applies them in natural name order. It does not recognise or invoke another migration tool.

Applied scripts are recorded in `tuskline.schema_migration` on the target database, with a SHA-256 checksum. A second run skips a script whose checksum still matches and refuses one that was edited, removed, or inserted before a script that is already applied. A session advisory lock makes a concurrent apply fail immediately. Each script runs in one transaction. A script that PostgreSQL cannot run inside a transaction runs on its own, and a file that mixes the two is refused.

Production stays read-only by default. `migrate apply --run` is a write on that one session, after `--approve` and `--confirm`, and it is audited. `DefaultPolicy` is unchanged.

## Consequences

- A directory laid out for another tool is just a directory of SQL files, if the files are `.sql` and sit in the top directory. Ordering follows the file names.
- There is no undo command. Recovery from a bad migration is a restore, or a new script.
- Repeatable scripts are not re-run when they change. A change to an applied file is an error.
- [M7.2b](../roadmap.md) is this runner. Drift between environments remains M7.3.
