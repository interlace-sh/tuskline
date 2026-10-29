# web

The Tuskline web UI: SvelteKit (static SPA via `adapter-static`), Svelte 5 runes, Tailwind CSS 4 and shadcn-svelte (Bits UI). Licensed AGPL-3.0 with the server.

```sh
npm install
npm run dev      # http://localhost:5173 (proxies /api to tuskline serve on :8080)
npm run check    # svelte-check
npm run lint     # prettier + eslint
npm test         # vitest
npm run build    # writes build/; `moon run server:embed-web` copies it into server/internal/webui/dist
```

Needs Node.js 22.17 or newer; `proto install` at the repository root installs the pinned Node.js and npm, and `moon run web:check` runs everything CI runs. Add components with `npx shadcn-svelte@latest add <name>`.
The React prototype in `../prototype/` is the design reference; see `../docs/design.md`.
