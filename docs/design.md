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
| Chart series | `--c1` … `--c6` | Wait and metric colours, light and dark, in `layout.css` |

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
- **Setup & deploy:** an area containing Setup overview, Config advisor, Deploy and Deployments. Add connection is the button on Fleet (`n`), not a sidebar item.

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
| 11 | Migrations: sample review. Approve stays disabled until the rewrite; nothing is applied | 07 | M7 |

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

Explain draws the plan with a D3 tree (`d3-hierarchy`), a plain-English summary and warnings for sequential scans, row misestimates and disk spills. On production, `EXPLAIN ANALYZE` of a write asks for the cluster name before it is sent. The session stays read-only, so PostgreSQL still rejects the write.

Result rows arrive as a stream and the grid virtualises them (TanStack Virtual), so only the visible window is in the DOM. The server has no sign-in; that is a later feature.

Config advisor (`g v`) lists the changes from `GET /api/v1/clusters/{env}/{name}/advise`. Demo mode shows a fixed 16GB web sample. Apply opens a confirmation and does not run the SQL. Production stays disabled until a change ticket and the cluster name are typed. Setup (`g u`) links to Add connection and Config advisor, and for a saved cluster lists missing extensions with the install preview. The preview is not applied. Deploy (`g d`) walks Target, Topology, Sizing and Review, then records a demo history row. It does not start Docker. HA (`g h`) shows members and a switchover that, on production, stays disabled until the cluster name is typed. The demo switchover does not call Patroni. Backups (`g b`) previews recovery and does not restore.

Add connection (`n`, or the Fleet button) is one form. Test connection probes the server and saves nothing. A local server on the default TLS mode, verify-full, does not answer when PostgreSQL has no certificate: choose disable for that disposable database, and set the database to `postgres` when the URL does not name one. Saved connections live in the inventory file. Create probes the server and saves it the way `tl connect` does: the button spins, an error stays on the form, and a success turns the action green before the SQL editor opens on that database. A typed password is saved in the OS keychain (`tuskline/<env>/<name>`). The inventory stores that reference, or an environment-variable reference, or relies on `~/.pgpass`. The password is not kept in the browser, in the inventory file or in SQLite. `?demo=1` checks the form locally and does not dial out. Deployments is for clusters Tuskline provisions later, not this list.

## Managing saved connections

A saved connection is one inventory cluster, `environment/name`. The databases on that server stay in the SQL editor tree. Fleet is the list of connections. Demo fixtures have no edit or delete.

**View.** A live Fleet card shows the name, host and port, database, user, TLS mode, PostgreSQL version from the last probe, and where the password comes from: keychain, environment variable, or `~/.pgpass`. It does not show the password. Clicking the card opens cluster health. `tl ls` is the same list.

**Edit.** Each card has an Edit action that does not fire when the card itself is clicked. Edit opens the Add connection form, titled Edit connection, filled from `GET /api/v1/clusters/{env}/{name}`. Environment and name are the identity. They are read-only, because the keychain item is `tuskline/<env>/<name>`. Renaming is a new connection. Host, port, database, user, TLS and the password reference can change. Test connection is available and still saves nothing.

Save probes again, then updates that entry. An empty password keeps the current keychain item, environment variable or `~/.pgpass`. A new password replaces the keychain item. Setting a password environment variable stores that reference and deletes the old keychain item only after the inventory save succeeds. On production, Save stays disabled until the cluster name is typed. A failed probe leaves the saved entry unchanged.

**Delete.** Delete removes the inventory entry and, when the password reference is a Tuskline keychain item, that item. It does not drop a database or role on the server. The confirm dialog asks for the cluster name on every environment. Production uses the danger style. The button stays disabled until the typed name matches. There is no delete in demo mode.

`DELETE /api/v1/clusters/{env}/{name}` carries `{ "confirm": "<name>" }`. A mismatch is rejected and nothing is removed. `tl rm <env/name>` asks for the same name on a terminal. `--yes` is accepted only when the environment is not prod.

The CLI and the form stay in step: `tl connect` with the same `env/name` is the edit, and it follows the same password rules.

A saved cluster's health screen can cancel a query or terminate a session after the environment's confirmation, and it draws session counts while the page is open. Stored metric samples appear when the collector has written them. Query insights can test one CREATE INDEX with hypopg; no real index is created. Migrations can upload .sql files and apply them with the same confirmation as `tl migrate apply`. Backups can parse a pgBackRest info file the operator chooses. A saved cluster's HA screen reads the Patroni URL registered on the inventory and can send a switchover or a replica restart after the environment's confirmation. Backup contents stay on `tl backup status`, because the server does not run pgBackRest. The migrations screen lints a script and diffs two clusters; applying a directory stays on `tl migrate apply`. Deploy on a real inventory does not start Docker. Demo mode (`?demo=1`) keeps the prototype samples so the smoke checks still pass. Load-history charts, session cancel, hypopg index advice and alert delivery are not on these screens.

The SQL editor fills the page width. Narrower screens, such as Add connection, stay in a centred column. The database column lists extensions installed in the current database and those available to install, plus `shared_preload_libraries`. A saved cluster reads them from the server (`GET .../probe`). Demo mode shows a labelled sample instead. The database name is a menu for the other databases on that server. Tables and views share the schema tree, with a different icon for views. Selecting one highlights it in the brand colour and opens the inspector; the arrow expands its columns. The active sidebar item uses the same colour. Overview and Indexes open to a set height, up to a maximum, and the divider resizes that panel. Overview loads counts, distinct values and nulls on its own: a saved cluster reads `pg_stats` (planner estimates, not a table scan), and the orders demo profiles its sample rows. A histogram is in the column detail, not on every row. Indexes lists that relation's indexes. Notebook snippets can be renamed. The results bar switches a page between one SQL script and a notebook. A snippet, and the SQL editor, can expand to fill the page. Escape leaves that full screen. The files control on the tab bar opens a panel that can attach a local folder and open its `.sql` files.

## Porting the prototype

1. Use the frozen React prototype as the spec. Its screenshots and smoke checks carry over unchanged, because they check visible text and behaviour.
2. Move the data: `prototype/src/data.ts` now lives in `web/src/lib/fixtures` (M3.2). `setup-data.ts` moves with the setup screens. Demo mode is the empty-inventory path; `?demo=1` forces the fixtures. The React context store becomes `store.svelte.ts` with `$state` and `$derived`.
3. Port primitives first: buttons, badges, cards, dialogs, tabs and the command palette, onto shadcn-svelte and Bits UI. `lucide-react` becomes `@lucide/svelte`.
4. Port screens in the order above. Recharts becomes uPlot for time series, and plain SVG components replace the small charts.
5. AI agents do the mechanical JSX-to-Svelte conversion. Each screen is accepted only when its smoke checks pass and its screenshot matches. `moon run server:smoke` runs the checks for the shipped shell against `tuskline serve` with `?demo=1`. Connect, advisor, deploy, HA, backups and migrations stay in `prototype/scripts/smoke.mjs` until those screens land.
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
