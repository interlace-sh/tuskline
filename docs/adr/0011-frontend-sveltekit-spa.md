# 0011. Web stack: SvelteKit SPA, Svelte 5, Tailwind, shadcn-svelte

- Status: Accepted
- Date: 2026-10-29

## Context

The UI must run in a browser (served by the Go server) and inside a system webview (Wails). It is a dense operations UI. The design prototype was built in React for speed.

## Decision

- **SvelteKit with `adapter-static` in SPA mode:** `ssr = false`, `fallback: 'index.html'`, no Node server. `go:embed` puts the build into the server, and the same build runs inside Wails.
- **Svelte 5 runes**, **Tailwind CSS 4** and **shadcn-svelte** on Bits UI. Also TanStack Table and Virtual, CodeMirror 6, uPlot (and D3 for plan trees), and `@lucide/svelte`.
- The React prototype in `prototype/` is frozen as the **spec**. Its screenshots and Playwright smoke checks are the acceptance tests for the port. Delete it once every screen passes.

## Consequences

- Smaller bundles and less runtime work suit a webview. The smaller ecosystem matters little, because the heavy components (CodeMirror, uPlot, D3) are framework-agnostic.
- SvelteKit 3 needs Node.js 22.17 or newer for development and CI.
