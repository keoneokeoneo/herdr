# Base — guide for whoever changes it (human or LLM)

Read `docs/guide.md` for repository conventions first. Base is the neutral layer: the
sidebar, tab bar, pane header, bottom usage bar and right panel. It needs no Poke code. The
design record is `docs/specs/2026-11-04-layer-separation-design.md`.

## Install

```sh
python3 scripts/setup.py install base            # or: npm run setup:install -- base
python3 scripts/setup.py install base --select base
```

It copies `mods/base/`, `layouts/base.json` and `themes/base.json` (nothing when the repo is
already `~/.config/herdr-app`) and adds the `layout:base` key bindings. It does not edit
`~/.zshrc`, `programs.json` or `settings.json`.

## What lives where

| To change… | Look at |
|---|---|
| Window structure, sizes, which view goes where | `layouts/base.json` |
| Colours | `themes/base.json` |
| Keys for this layout | `keys.json`, section `layout:base` |
| Which view a block shows | `mods/base/main.js` (`props.view` → module) |
| Sidebar top (optional logo via `props.logo`, search, fold button) | `mods/base/views/brand.js` |
| Sidebar spaces and agents | `mods/base/views/spaces.js`, `mods/base/views/agents.js` |
| Tab bar and panel toggle | `mods/base/views/tabs.js` |
| Pane header | `mods/base/views/pane.js` |
| Right panel header tabs (declared per layout in `props.tabs`) | `mods/base/views/paneltabs.js`, `mods/base/lib/panel.js` |
| Usage tab (limit cards, 14-day chart) | `mods/base/views/panel.js` |
| Bottom bar (limits, today's tokens, jump pill) and key commands | `mods/base/views/status.js`, `mods/base/behaviours.js` |
| Reading the usage files | `mods/base/views/usagefiles.js`, `mods/base/lib/contract.js` |
| Agent rows (Herdr data → what views draw) | `mods/base/views/data.js` |
| Avatars and pills | `mods/base/views/ui.js`, `mods/base/lib/avatar.js` |
| Pure logic (urgency, groups, chips, sublines, formatting) | `mods/base/lib/`, tests in `tests/base/mod-lib/` |
| Bundled UI fonts | `mods/base/fonts/` |

Layout ids: `sidebar`, `brand`, `spaces`, `agents`, `footer`, `work`, `tabs`, `work-body`,
`main`, `panel` (`panel-tabs`, `panel-body` with `usage`, `changes`, `files`, `obsidian`), `status`.
Commands (reach shown copies; the bottom bar copy acts on them): `base.jump` (focus the next
agent waiting for you, oldest first) and `base.notes` (show the panel on its Obsidian tab).

## Usage data contract

The UI only reads these files. Each has exactly one writer.

| File in `~/.local/state/herdr-app/usage/` | Content | Writer |
|---|---|---|
| `claude.json` | `{plan, windows:[{minutes, used, resets_at}], updated}` for the 5h (300) and weekly (10080) windows | the global Claude statusline (`~/.claude/statusline`, outside this repo) |
| `codex.json` | same shape | the `herdr-poke codex` wrapper (`helpers/poke/internal/codex`) |
| `daily.json` | `{days:[{date, claude, codex}], updated}`, 14 local days, oldest first | `herdr-poke usage` (launchd, every 10 minutes) |

Rules every reader and writer keeps: `updated` and `resets_at` are epoch seconds; a window
whose `resets_at` has passed is never shown as a percentage; an older snapshot never replaces a
newer one; a missing, empty or corrupt file means "no data", never an error. Override the folder
with `HERDR_USAGE_STATE` (tests). `HERDR_POKE_STATE` only relocates the helper's private state
and never moves the contract folder (the launchd job exports it permanently).

### The Claude statusline change

Claude's rate limits only arrive on the statusline's stdin, so `~/.claude/statusline/main.go`
records them after printing its line: at most 300 ms, errors ignored, the printed text unchanged.
It skips the write when the file holds a newer snapshot or nothing changed, and keeps the larger
`used` for the same reset time. The previous source and binary were kept as
`main.go.bak-layer-separation` and `statusline.bak-layer-separation`. Rebuild with the command in
that file's header and test with `go test main.go main_test.go` there.

### Dependencies without Poke

- Claude figures need only the statusline change above.
- Codex figures, daily usage and the Codex model/effort/context labels need the `herdr-poke`
  helper (`~/.local/bin/herdr-poke`): without it those parts stay empty, and everything else works.
- Claude agent rows no longer show model, effort or context %; Codex rows still do.

## Persona seam

Base draws agent avatars and tab-type pills, and it reads these keys when something provides them.
When nothing does (no Poke), it shows a plain letter avatar and no type pill.

- Shared state: `poke.sprites` (file names), `poke.evening`, `poke.typeMode` (default off),
  `poke.typeSeed`.
- Terminal labels: `poke.partner`, `poke.review_of`, `poke.findings`, and the telemetry labels
  `poke.owner|provider|model|effort|ctx` (Codex only).

`tests/base/mod-lib/layouts.test.mjs` fails if `mods/base`, `layouts/base.json` or `themes/base.json`
mention anything Poke-named beyond this list.

## How to verify a change

1. `herdr-app check mods/base` prints `check passed`; `herdr-app status` shows `"errors": []`.
2. `npm test` (includes layout parity, key checks, the seam guard and the docs check).
3. Live: `herdr-app do layout.switch '{"name":"base"}'`; the bottom bar and Usage tab show both
   providers; after a Claude prompt `claude.json` `updated` advances; stop everything Poke and
   confirm the sidebar, tabs, usage and notes still work.
4. Never touch the user's other terminals.
