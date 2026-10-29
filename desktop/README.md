# desktop/ — Tuskline desktop app (Wails v3), placeholder

**Status:** not started. Milestone **M8** in `../docs/roadmap.md`. Licence: AGPL-3.0, because it bundles the web UI. See `../LICENSING.md`.

## Plan (from docs/adr/0009-desktop-wails-v3.md)

- **Wails v3** with a Go host and the system webview. The Tuskline core runs **in-process**, so there is no sidecar.
- The Wails asset server serves the **same** static build as `../web/`, and mounts the server's `http.Handler`. The UI therefore talks HTTP to the core exactly as it does in a browser. Generated Wails bindings are only for native extras: file dialogs, OS keychain, tray, deep links and the updater.
- **Local-first:** inventory and history live in SQLite in the user's config directory. Optional sign-in to a team server or Tuskline Cloud syncs the inventory.
- **Native access a browser lacks:** SSH agent, `~/.pgpass`, `pg_service.conf`, the keychain.
- **Build matrix:**
  - macOS on Intel and Apple Silicon, signed and notarised.
  - Windows x64 and arm64.
  - Linux x64 and arm64.
  - Test WebKitGTK early.
- **Distribution:** not through the Mac App Store (GPL-family licence conflict).
- **DuckDB** (needs cgo) ships here first. The server stays pure Go.
- **Exit route:** because the UI only speaks HTTP, moving to Tauri v2 with the core as a sidecar would be a packaging change.

## Scaffold when M8 starts

```sh
go install github.com/wailsapp/wails/v3/cmd/wails3@latest
wails3 init -n tuskline-desktop -t svelte-ts   # then replace its frontend with ../web
```

Add `./desktop` to `../go.work` once it has a `go.mod`.
