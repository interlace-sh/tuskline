# 0009. Desktop framework: Wails v3

- Status: Accepted
- Date: 2026-10-28

## Context

[0004](0004-surfaces.md) requires a desktop app built from the same Go core and Svelte UI. The candidates were Wails v3 (Go host, system webview), Tauri v2 (Rust host, core as a sidecar) and Electron (heavy, with a Node host). Wails v3 reached v3.0.0 on 26 Oct 2026.

## Decision

Use **Wails v3**.

- The core runs in-process: one language, one binary, no sidecar to sign and supervise.
- The Wails asset server serves the embedded Svelte build and mounts the server's `http.Handler`, so the desktop app uses exactly the same REST API as the browser.
- Generated bindings are used only for native features: file dialogs, keychain, tray, deep links and the updater.
- Local-first mode stores inventory and history in SQLite in the user's config directory.

## Consequences

- Risk: Wails v3 is newly stable and its ecosystem is smaller than Tauri's. WebKit (macOS) and WebKitGTK (Linux) need early testing.
- **Exit route:** the UI only talks HTTP to the core, so moving to Tauri v2 with the core as a sidecar is a packaging change, not a rewrite. Revisit this decision if Wails v3 shows stability problems in its first months.
- Build matrix from the first public build:
  - macOS on Intel and Apple Silicon, signed and notarised.
  - Windows x64 and arm64.
  - Linux x64 and arm64.
- DuckDB needs cgo, so it ships in the desktop build first, where cgo is already required. The server binary stays pure Go.
