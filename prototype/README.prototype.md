# Tuskline — clickable front-end prototype

High-fidelity, front-end-only prototype of **Tuskline** (CLI `tl`), an open-source PostgreSQL workbench.
Brand: Tusk Arc mark, accent `#3E63DD` (`src/components/Brand.tsx`).
All data is fake and hard-coded in `src/data.ts` and `src/setup-data.ts` (hosts use `.example`). No backend, no network calls; connection checks and deploys are simulated.

## Open it
* Double-click `tuskline-prototype.html` (single self-contained file, works offline).
* Or `npm install && npm run dev` for hot reload; `npm run build` writes `dist/` and the single-file HTML.
* Deep links: `#/fleet`, `#/cluster/<id>`, `#/queries/<id>`, `#/sql/<id>`, `#/ha/<id>`, `#/backups/<id>`, `#/migrations/<id>`, `#/advisor/<id>`, `#/setup`, `#/connect`, `#/deploy`, `#/deployments`; `?theme=light`, `?empty=1` (fresh install), `?nosplash`.

## Shortcuts
⌘/Ctrl K palette · g f/c/q/s/h/b/m navigate · n add connection · g d deploy · g v config advisor · g u setup · g l deployments · e environment · t theme · / filter · [ ] time range · ? help · ⌘/Ctrl ↵ run · ⌘/Ctrl ⇧ ↵ explain

## Stack
Vite 5, React 18, TypeScript, Tailwind 3, Recharts 3, lucide-react, Inter + JetBrains Mono (bundled via Fontsource), vite-plugin-singlefile.
`scripts/screenshots.mjs` and `scripts/smoke.mjs` use playwright-core with a local Chrome.
