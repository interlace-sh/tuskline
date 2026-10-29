<!--
Title: Conventional Commits, for example `feat(cli): add tl ls` (AGENTS.md section 9).
Every commit is signed off (`git commit -s`, DCO).
-->

## What and why

<!-- What changes, and the reason. Link the roadmap task, for example "Roadmap: M1.3". -->

Roadmap:

## How it was tested

<!-- Commands you ran and their output, for example `moon run :check` and `moon run core:integration`. -->

## Screenshots

<!-- UI changes only: dark and light themes. Delete otherwise. -->

## Risks and follow-ups

<!-- Anything a reviewer should watch, and new roadmap tasks you added. -->

## Definition of done (AGENTS.md section 8)

- [ ] Code meets AGENTS.md section 5 and the relevant `.cursor/rules`; SPDX headers are present; the licence boundary holds.
- [ ] Unit tests added or updated. Integration tests added for anything that touches PostgreSQL, passing on 14–18 (CI).
- [ ] `moon run :check` passes.
- [ ] User-visible behaviour is documented: `docs/cli.md` for commands, `docs/design.md` for screens, an ADR for decisions.
- [ ] The roadmap checkbox is ticked in this PR; follow-ups are added as new unchecked tasks.
- [ ] No secrets and no generated artefacts (`web/build`, `server/internal/webui/dist/*` except the placeholder, `bin/`) committed.
- [ ] UI work: matches the prototype screenshot, dark and light themes, keyboard reachable.
- [ ] Safety (AGENTS.md section 7): prod read-only defaults are untouched; any mutating action has a preview, pre-flight checks, typed confirmation in prod and an audit entry.
