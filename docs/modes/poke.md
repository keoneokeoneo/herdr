# Poke — guide for whoever changes it (human or LLM)

Read `docs/guide.md` for repository conventions first, then open only the files the change touches. Poke is the
persona layer on top of **base** (`docs/modes/base.md`): it adds the pokeTokenBar companion, the
Dex tab, partner sprites and cries. Base owns the sidebar, tab bar, usage bar and right panel, and
works without Poke; Poke requires base. The product intent and the original decisions are in
`docs/specs/2026-11-03-herdr-poke-design.md` (its "Current state" section lists what changed after
v1); the split into layers is in `docs/specs/2026-11-04-layer-separation-design.md`. Herdr's own API
reference is the generated `AGENTS.md` at the repo root, with `herdr.d.ts` and `gpui-kit.d.ts`.
That file runs to about 1000 lines, so search it for the API you need. Don't read it top to
bottom, and never edit it: Herdr rewrites it on every start.

## What lives where

When installed, the repo root is Herdr's home, `~/.config/herdr-app`. `.gitignore` lists everything
Herdr writes, so `git status` shows only our files.

| To change… | Look at |
|---|---|
| Window structure (blocks, sizes, which view goes where, traffic lights) | `layouts/poke.json` (keep its shared block ids equal to `layouts/base.json`; a test checks it) |
| Colours | `themes/poke.json`, `mods/poke/views/ui.js` `C` (type/chip accents) |
| Codex menu launch command | `programs.json` (`programs.codex.run`) |
| Keys | `keys.json`, section `layout:poke` (`cmd-j`, `cmd-k`, `cmd-shift-o` are base commands) |
| Which view a block shows | `mods/poke/main.js` (`props.view`: `companion`, `dex`, `host`) |
| Sidebar: pokeTokenBar companion card / egg | `mods/poke/views/companion.js` |
| Right panel Poke tab (Dex grid, tab-type controls) | `mods/poke/views/dex.js` |
| Things that happen: partner assignment, cries, Dex sync, level/evolution toasts, evening dim | `mods/poke/behaviours.js`, started by the invisible `poke-host` block (`mods/poke/views/host.js`) |
| Agent rows the persona reads (status and partner label) | `mods/poke/views/rows.js` |
| Sprites, pills, GIF redraw | `mods/poke/views/ui.js` (`sprite`, `alive`, `pill`) |
| Pure logic (partner pick, growth formula, cries, defaults) | `mods/poke/lib/`, with tests in `tests/poke/mod-lib/` |
| Sidebar, tabs, pane header, bottom bar, Usage tab, panel tabs, avatars | **base**: `docs/modes/base.md` |
| Session ownership and complete telemetry snapshots | `helpers/poke/internal/labels/session.go`, `mods/base/lib/telemetry.js` |
| Label caching, failure backoff, `hc label` timeout | `helpers/poke/internal/labels/labels.go` |
| CLI dispatch and helper build | `helpers/poke/main.go`, `scripts/poke/build.sh` |
| Mod registration | `mods/poke/mod.json` (bundled UI fonts are in `mods/base/fonts/`) |
| Node entry points for hc | `scripts/poke/nvm-shim` |
| Codex → labels and limits (wrapper and `--attach`) | `helpers/poke/internal/codex` (`wrap.go`, `attach.go`, `find.go`, `rollout.go`) |
| Sprites, cries, names, GIF normalisation | `helpers/poke/internal/assets` (`assets.go`, `normalize.go`) |
| 14-day usage aggregation (launchd every 10 min) | `helpers/poke/internal/usage` |
| State files and locks | `helpers/poke/internal/state` (`~/.local/state/herdr-poke/`; the usage contract is `~/.local/state/herdr-app/usage/`) |
| Installation prerequisites | `scripts/poke/preflight.py` (read-only installation check), `tests/poke/scripts/test_preflight.py` |
| New Mac migration | `docs/plans/2026-11-04-machine-migration.md` |
| Install and undo | `scripts/poke/install.sh`, `scripts/poke/uninstall.sh`, `scripts/poke/settings_edit.py` (configuration journal), `scripts/poke/install_files.py` (installed-file ownership/backups), tests in `tests/poke/scripts/` |
| Selective install of the setups | `scripts/setups.json`, `scripts/setup.py`, `tests/setup/test_setup.py` |

Seen is a deduplicated snapshot of partners on open agent sessions across spaces.
The existing 2-second behaviour tick replaces it only when membership changes.

## Data flow in one paragraph

Herdr gives the mods `agents()`, `spaces()`, `terminals()` (status, labels, focus). The Go
helper `~/.local/bin/herdr-poke` adds what Herdr lacks, for Codex:
- the `codex -- …` wrapper (the zsh alias) mirrors a Codex session. A Codex started any other
  way gets periodic `codex --attach <t>` probes from `mods/base/behaviours.js`, even if old
  model labels remain. Local bare Codex foreground processes are supported; the
  helper cannot inspect a remote machine's PID or files.

It publishes complete terminal snapshots (`poke.model`, `poke.effort`, `poke.ctx`) with
`poke.owner` (foreground PID), `poke.provider` and `poke.session`, and the limits file
`~/.local/state/herdr-app/usage/codex.json`. `usage` writes `daily.json` in the same folder.
Claude's limits come from the global Claude statusline, which Poke no longer touches; Claude rows
therefore have no model, effort or context. Sprites, cries and names go to
`~/.config/herdr-app/poke-assets/`, which the mods read as `poke-assets/<file>`, and Poke
publishes the file list as shared `poke.sprites` so base can draw avatars. pokeTokenBar's folder
(`~/Library/Application Support/PokeTokenBar`) is read only.

## Gotchas (each one cost real time)

- **Hot reload:** Herdr reloads a mod whenever any non-hidden file inside
  `mods/<name>/` is written. Anything written at runtime goes in `poke-assets/` or
  `~/.local/state/herdr-poke/` or `~/.local/state/herdr-app/usage/`. Writing into `mods/poke/` at runtime loops the reload
  until QuickJS runs out of memory.
- **GIFs advance only when the window redraws.** `mods/poke/views/ui.js` `alive()` (and the copy in `mods/base/views/ui.js`) wraps a sprite in a
  sub-pixel native `.loop()` so it keeps animating. Static PNGs bypass this loop.
  An earlier local observation was about 3% CPU; this is not a measured budget
  across machines or sprite counts.
- **Sprite GIFs come with uneven frames** (each frame a smaller rectangle, non-square
  canvas). Herdr stretches every frame to the box. The helper squares them
  (`assets --normalize` for repair; download, companion copy and `assets --dex`
  normalize new sprites). Never show
  a raw GIF.
- **macOS traffic lights:** `traffic_lights: [12, 10]` in the layout centres them on the
  46px brand row. They span up to x≈80, so the brand row pads 96px and the folded
  sidebar is 92px wide. Below 120px every sidebar view switches to its rail form
  (`narrow()`).
- **Colours:** `"transparent"` is not a colour; use `"#00000000"`.
- **Start Herdr from the Dock,** never from a Claude shell. A herdr-core server started
  there inherits `CLAUDE_CODE_CHILD_SESSION=1`, and every Claude in it stops saving
  transcripts. Don't run `hc` from a Claude shell when no server is up.
- **The hc server has a minimal PATH** (no nvm, no zsh aliases). Programs started by
  `hc run` therefore:
  - get node from the `~/.local/bin/{node,npm,npx}` shims (`scripts/poke/nvm-shim`);
  - get no permission prompts from the global defaults (see "Outside the repo");
  - get Codex labels from `--attach`.
- **Shipped mods are restored on every start.** An edited one is moved to
  `<name>-edited`. Clone a shipped mod instead of editing it. Reuse shipped behaviours
  through commands: `openOverlay({mod:"menu"})`, `mod:"new-menu"`, `layout.toggle`,
  `layout.resize`, `terminal.pin`, `app.palette`.
- **`setShared` re-renders every view.** Only call it when the value really changed.
  `cx.notify()` rebuilds its owning view and redraws the window. Hidden views keep
  their timers after first display; native animation stops when hidden.
- **Codex mirror ownership:** both wrapper and attach must acquire the terminal's
  flock before monitoring. Attach checks the terminal foreground every 5 seconds
  and exits if it changes or cannot be queried. A minute probe recovers a stopped
  mirror; retained model labels are not evidence of a live mirror. Lock files stay
  on disk intentionally: flock releases on process exit, so do not unlink live locks.
  Label backoff returns an error until publication can retry; latest Codex limit
  comparison and replacement happen under the same lock.
- **Telemetry ownership:** the UI hides snapshots without a matching foreground PID
  and provider. The Codex wrapper checks `hc get` and process ancestry under a bounded
  timeout before publishing, so a delayed old process cannot claim a replacement
  session. Codex rechecks its open rollout every 5 seconds for in-process session
  switches, clearing missing fields when it switches. PID reuse is not a process
  start-time identity check; unsupported/legacy snapshots are shown as unavailable.
  Rebuilding the helper does not upgrade an already-running Codex wrapper/attach
  process. Existing wrappers need a normal restart after the user finishes their
  session; do not interrupt their work to upgrade.
- **Codex needs `--no-daemon`.** Without it, `lsof` can't tie the process to its
  rollout and `HC_TERMINAL_ID` is lost.

## How to verify a change

1. `herdr-app check mods/poke` and `herdr-app check mods/base` must print `check passed`. `herdr-app status` must show
   `"errors": []`.
2. Logic: `npm test` (mod `lib/`, layouts, docs) and `cd helpers/poke && go test ./...`. Installer:
   `npm run test:poke:scripts`; selective installer: `npm run test:setup`.
   Installer integration uses a temporary HOME and a stub builder; launchd and
   the live app are excluded. Fix with a failing test first.
3. Visual: `herdr-app shot out.png` saves the last frame without the macOS traffic
   lights. With the lights, find the window id (`CGWindowListCopyWindowInfo`, owner
   "Herdr") and run `screencapture -x -o -l <id> out.png`. Captures are 2×. Crop before
   looking. `sips -c` crops around the centre, so use a small Swift or CG script.
4. Sidebar rail: `herdr-app do layout.resize '{"id":"sidebar","px":92}'` to fold, `262`
   to restore. Put the width back when done.
5. Helper changes: `cd helpers/poke && go build -o ~/.local/bin/herdr-poke .`. A hidden test
   terminal (no `herdr.space` label, so the app never shows it):
   `hc run --bg --no-notify --name x-test -- codex --no-daemon "Reply only OK."`, then
   `hc kill`, then `hc rm`.
6. Never touch the user's other terminals.

## Rules

- Commits follow the repo history: `type(scope): 한국어 명사형` title (`~/Documents/work/commit-convention.md` applies only under `~/Documents/work`),
  body lines ending in ~함/~됨. No AI signature in commits or PRs. Pushing to
  `keoneokeoneo/herdr` (private) is fine.
- Docs go in `docs/specs/` and `docs/plans/` (whatever location a tool suggests). No commit
  hashes in docs.
- pokeTokenBar files are read only. Never open or decrypt its `usage-cache.json`.
- Back up before editing `~/.claude/settings.json`, `~/.zshrc`, `~/.codex/config.toml` or
  Herdr's `settings.json`.
- UI text is English. Only the Pokémon are pixel art. The theme stays dark and calm.

## Outside the repo (what a new machine needs)

Use `docs/plans/2026-11-04-machine-migration.md` for backup, safe checkout placement,
external settings and post-migration verification. `python3 scripts/poke/preflight.py`
checks required CLIs, the local Go version and Codex `--no-daemon` support before
installation. Node/nvm, Jetendard and pokeTokenBar state produce warnings. It does
not verify login or live UI. `install.sh` runs it before any installation writes;
`HERDR_POKE_DRY=1` still writes files and is for isolated tests, not a read-only check.

`scripts/poke/install.sh` sets up most of this; check it after a migration. It first installs
`base` and `poke` through `scripts/setup.py`. It never edits the global Claude `statusLine`. Missing
configuration files are created. An absent Codex alias is added; an exact
old alias is replaced; custom or duplicate aliases abort rather than being overwritten.
The installer adds a managed `~/.local/bin` PATH line to `.zshrc`.

Configuration originals are journaled before the first write. Repeated installation
preserves those originals. Installed binaries, shims, links and launchd plist have
an independent ownership journal; uninstall restores originals only if the installed
file still matches. User replacements and unjournaled legacy files are kept. An
interruption between writing a file and recording its fingerprint is left for manual
recovery from its saved original. Do not run installers concurrently.

Uninstall leaves the selected layout, repo, runtime state, assets and backups in place.
It does not undo the deliberately manual permission settings or external skills.

| Piece | Where | Installed by |
|---|---|---|
| Terminal font Jetendard | macOS fonts | manual; not bundled |
| Node versions and default selection | `${NVM_DIR:-~/.nvm}` | install nvm/Node first; shims require it |
| Helper binary | `~/.local/bin/herdr-poke` | install.sh |
| Claude limits file (the global statusline writes `claude.json`) | `~/.claude/statusline/` | manual; not in this repo, see `docs/modes/base.md` |
| `alias codex='herdr-poke codex -- codex --yolo --no-daemon'` | `~/.zshrc` | install.sh (manifest) |
| Herdr sounds off | Herdr `settings.json` | install.sh (manifest) |
| launchd usage job `dev.keno.herdr-poke.usage` | `~/Library/LaunchAgents/` | install.sh |
| node/npm/npx shims for hc | `~/.local/libexec/nvm-shim`, `~/.local/bin/{node,npm,npx}` | install.sh |
| No permission prompts anywhere: Codex `approval_policy="never"`, `sandbox_mode="danger-full-access"`; Claude `permissions.defaultMode="bypassPermissions"` | `~/.codex/config.toml`, `~/.claude/settings.json` | manual, deliberately (security setting) |
| Review skills launch with the alias flags and the Codex wrapper | `~/.claude/skills/dual-review-hc`, `dual-review` | not in this repo |

## Background work and UI scope

| Work | Interval / trigger | Cost and scope |
|---|---|---|
| Agent transitions, partner assignment | 2 seconds | Reads Herdr rows; no unconditional notify |
| Attach health probe (base status bar copy) | 60 seconds per Codex | One helper; held flock exits before querying hc |
| Attach foreground validation | 5 seconds per live attach | One bounded `hc get`; failure exits for later retry |
| Rollout discovery / tail | about 1 second | `lsof` until found; then incremental reads + open-rollout recheck every 5 seconds |
| Companion + Dex copy, evolution/level toasts | PTB folder watch (`companion-state.json`, `sprites/`) + 300 seconds | Three helper executions per run, coalesced; no unchanged file rewrite; only new variant GIFs normalized |
| Limits/usage load (base) and Dex load (Poke) | filtered folder events + 15 seconds | Suppresses unchanged notify; concurrent loads coalesced |
| Status/panel time labels | 60 seconds | Countdown/expiry refresh |
| Agent/pane elapsed labels | 20 seconds per initialized view | Timers continue in hidden views |
| Companion card / events | 30 seconds + card folder watch | Filtered/coalesced reads; defaults lookup every 30 seconds; notify only on visible changes |
| Usage aggregation | launchd every 600 seconds | Incremental log reads, but directory discovery walks log roots |

Asset jobs start three helpers each time pokeTokenBar writes its state (every few
minutes) and on the 5-minute fallback, plus partner fetches, retries and attach
probes. CPU impact has not been remeasured.

The brand button folds to a rail; `cmd-b` hides the sidebar. Double-click rename and
reorder are implemented for expanded spaces and tabs, not agent rows or rail spaces.
Pane footer/dirty/ahead, review completion badges, partner-based terminal renaming,
and a selectable by-repository type mode are deferred. Current scope is also recorded
in the design's Current state section.

## Unified right panel

`panel-body` (in `layouts/poke.json`) shows one of Usage / Changes / Files / Poke / Obsidian; the tabs
are declared on the `panel-tabs` block. Usage, Changes, Files and Obsidian are base's; the Poke tab
(`poke-dex`) is the Dex. The panel button or Cmd-K hides/shows the whole panel, and Cmd-Shift-O
(`base.notes`, handled by the bottom bar) opens Obsidian. There is no separate Notes panel. The Obsidian
reader has a collapsible folder tree and shows vault/note names without full paths; see
[the reader guide](obsidian.md).
