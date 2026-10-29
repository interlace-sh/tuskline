# 0004. Surfaces: desktop app as well as web and CLI

- Status: Accepted
- Date: 2026-10-28

## Context

Developers expect a native app with local access: SSH agent, `~/.pgpass`, `pg_service.conf`, the OS keychain and file dialogs. DBAs expect a shared web server. Everyone wants a CLI.

## Decision

Ship four surfaces on one core:

- **Desktop app:** local-first, SQLite, optional sign-in to a server or Cloud.
- **Web UI:** served by the server.
- **`tl` CLI/TUI.**
- **`tl agent`.**

The desktop app is the same Svelte UI and Go core, packaged natively. It is not a fork. The framework choice is in [0009](0009-desktop-wails-v3.md).

## Consequences

- The UI talks only HTTP to the core (`/api/v1`), whether the core is in a browser tab or in-process in the desktop app. There are no private endpoints.
- Every release must work end to end from the desktop app and from `docker run`.
- Signed builds for macOS, Windows and Linux add CI work. Desktop is scheduled as M8 so the core and UI stabilise first.
