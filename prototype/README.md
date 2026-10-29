# prototype/ — design reference (React, frozen)

This is the clickable front-end prototype of Tuskline (v0.2.0-prototype). It is the **design specification** for the real web UI in `../web/` (SvelteKit + Svelte 5 + shadcn-svelte). It is not shipped, and nothing here talks to a database: every number is fake demo data and every connection, check and deploy is simulated.

- Open `tuskline-prototype.html` in a browser: one self-contained file that works offline.
- `screenshots/` holds 35 reference screenshots (dark, plus a few light). Port screens until they match these.
- `src/data.ts` and `src/setup-data.ts` hold the fake data and types. They move to `web/` largely unchanged, as typed fixtures for the demo mode and for tests.
- `scripts/smoke.mjs` contains 21 Playwright checks that look at visible text and behaviour, so they can be pointed at the Svelte build as acceptance tests. They need Chrome (`/usr/bin/google-chrome`; adjust `executablePath` locally).

Rebuild (optional): `npm install && npm run build` (Vite 5, React 18, Tailwind 3).

## Porting rules

Follow `../docs/design.md` § "Porting the prototype" and `.cursor/rules/frontend-svelte.mdc`:

1. Port primitives first (buttons, badges, cards, dialogs, tabs, command palette) onto shadcn-svelte / Bits UI. `lucide-react` becomes `@lucide/svelte`.
2. Then port the screens in this order: Fleet, Cluster health, Query insights, SQL editor, HA, Backups, Migrations, Setup & deploy (Add connection, Config advisor, Deploy, Deployments).
3. The React context store (`src/store.tsx`) becomes `store.svelte.ts` using `$state` / `$derived` runes. Recharts becomes uPlot for time series, with plain SVG for small charts.
4. A screen is done when its smoke checks pass against the Svelte build and it matches its screenshot. Delete this directory once every screen is ported (milestone M3/M4).

`README.prototype.md` is the prototype's original README.
