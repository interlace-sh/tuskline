# 0001. Launch personas: both developers and DBAs/SREs

- Status: Accepted
- Date: 2026-10-28

## Context

PostgreSQL tools split into developer SQL clients (DataGrip, DBeaver, TablePlus) and fleet tools for DBAs (pganalyze, pgwatch, vendor consoles). Each group needs much of what the other has: query insights, plans and health.

## Decision

Launch for both: developers (IDE first) and DBAs/SREs (fleet first). One core and one inventory, with two front doors.

- The desktop app and `tl` lead for developers.
- The server and the hosted fleet view lead for DBAs.
- Shared screens (query insights, plans, health checks, config advisor, schema browser) are built once.

## Consequences

- The home screen adapts to what is connected: one database gives the database view, three or more clusters give Fleet. Features appear when their data exists.
- There is no persona switch and there are no editions. Anything outside both personas' top jobs becomes a YAML check or a plugin.
- Release 1 is the developer door, because it needs no server. Release 2 adds the fleet door.
