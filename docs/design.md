# Design

The design reference is the frozen React prototype in [`../prototype/`](../prototype/). To view it, open `prototype/tuskline-prototype.html`; it works offline. Its 35 screenshots in `prototype/screenshots/` are the visual spec, and its Playwright smoke checks (`prototype/scripts/smoke.mjs`) are the behavioural spec. The principles are in [vision.md](vision.md#design-principles).

## Tokens

| Token | Value | Use |
|---|---|---|
| Ink | `#0E1116` | Dark background; dark logo tile |
| Paper | `#FAFAF9` | Light background |
| Accent | `#3E63DD` | Primary actions, focus, links (`--primary`) |
| Accent light | `#7B96FF` | Accent on dark surfaces, focus ring in dark |
| Border (dark) | `#2A303A` | Logo outline, dividers |
| Prod | `#E5484D` | Production rail and badges (`bg-env-prod`) |
| Staging | `#F5A524` | Staging rail (`bg-env-staging`) |
| Dev | `#30A46C` | Development rail (`bg-env-dev`) |
| Local | `#8B8D98` | Local or unclassified (`bg-env-local`) |

- Neutral greys carry most of the UI. Colour is reserved for environment identity, status and data.
- **Type:**
  - UI text uses Inter (variable), with tabular numerals for every metric.
  - Code uses JetBrains Mono, with ligatures off.
- **Themes:**
  - Dark and light are designed together from tokens, in 12-step scales in the style of Radix Colors.
  - A high-contrast mode is planned.
  - The shell uses a single comfortable density.
  - Scrollbars are thin everywhere: a `--tl-line2` thumb on a transparent track.
- Tokens are implemented in `web/src/routes/layout.css` (shadcn-svelte CSS variables plus `--env-*` and `--brand`). Components must not use raw hex values.

## Information architecture

- **Top bar:** organisation and environment switcher, global search, command palette.
- **Left rail:** cluster-scoped sections: Overview/Health, Queries, SQL, HA, Backups, Migrations, Advisor.
- **Breadcrumb:** Project › Environment › Cluster › Instance › Database.
- **Time range:** one global time-range control that every chart obeys.
- **Setup & deploy:** an area containing Setup overview, Add connection, Config advisor, Deploy and Deployments.

## Screens (port in this order)

| # | Screen | Reference screenshots | Milestone |
|---|---|---|---|
| 1 | App shell, command palette, splash, about | 08, 31, 32, 33 | M3 |
| 2 | Fleet (incl. empty state, new connection/cluster) | 01, 12, 19, 28, 30 | M3 |
| 3 | Cluster health | 02, 11 | M3 |
| 4 | Query insights | 03 | M3 |
| 5 | SQL editor (prod read-only, write blocked, elevated) | 04, 10, 13 | M3 |
| 6 | Setup overview, Add connection (details, test, capabilities, role) | 14–18, 35 | M4 |
| 7 | Config advisor (incl. prod guard) | 20, 21 | M4 |
| 8 | HA topology, switchover modal | 05, 09 | M6 |
| 9 | Backups and PITR | 06 | M6 |
| 10 | Deploy (target, topology, sizing, review, running, complete), Deployments | 22–27, 29, 34 | M5 |
| 11 | Migrations | 07 | M7 |

## Keyboard

| Keys | Action |
|---|---|
| `Cmd/Ctrl K` | Command palette |
| `g f` / `g c` / `g q` / `g s` / `g h` / `g b` / `g m` | Fleet, cluster, queries, SQL, HA, backups, migrations |
| `n` / `g d` / `g v` / `g u` / `g l` | Add connection, deploy, config advisor, setup, deployments |
| `e` | Switch environment (colour flash; confirmation for prod) |
| `t` | Toggle theme |
| `Cmd/Ctrl Enter` | Run statement; `Shift` adds EXPLAIN ANALYZE |
| `/` | Filter the current list |
| `[` / `]` | Move the time range back or forward |
| `?` | Contextual shortcut sheet |

## Porting the prototype

1. Use the frozen React prototype as the spec. Its screenshots and smoke checks carry over unchanged, because they check visible text and behaviour.
2. Move the data: `prototype/src/data.ts` now lives in `web/src/lib/fixtures` (M3.2). `setup-data.ts` moves with the setup screens. Demo mode is the empty-inventory path; `?demo=1` forces the fixtures. The React context store becomes `store.svelte.ts` with `$state` and `$derived`.
3. Port primitives first: buttons, badges, cards, dialogs, tabs and the command palette, onto shadcn-svelte and Bits UI. `lucide-react` becomes `@lucide/svelte`.
4. Port screens in the order above. Recharts becomes uPlot for time series, and plain SVG components replace the small charts.
5. AI agents do the mechanical JSX-to-Svelte conversion. Each screen is accepted only when its smoke checks pass and its screenshot matches.
6. Embed the result: `make web` puts the build into the server, and the same build runs in Wails.
7. Delete `prototype/` when every screen passes.

## AI features, done tastefully

- **Do:**
  - Explain plans while citing their nodes and numbers.
  - Validate index advice with hypopg.
  - Turn natural language into SQL that is always shown as editable SQL.
  - Draft incident summaries.
  - Let users bring their own model (including local Ollama). AI is off by default.
- **Don't:**
  - Auto-execute writes or DDL.
  - Send rows or parameters to a model without a policy allowing it.
  - Make chat the main interface.
  - Show confidence without evidence.
