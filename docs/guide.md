# herdr-poke — guide for whoever changes it (human or LLM)

Read this first, then open only the files the change touches. The product intent and
the original decisions are in `docs/specs/2026-11-03-herdr-poke-design.md`. Its
"Current state" section lists what changed after v1. Herdr's own API reference is the
generated `AGENTS.md` at the repo root, with `herdr.d.ts` and `gpui-kit.d.ts`. That file
runs to about 1000 lines, so search it for the API you need. Don't read it top to
bottom, and never edit it: Herdr rewrites it on every start.

## What lives where

The repo root is Herdr's home, `~/.config/herdr-app`. `.gitignore` lists everything
Herdr writes, so `git status` shows only our files. Paths beginning with `views/`
in the table are relative to `mods/poke/`.

| To change… | Look at |
|---|---|
| Window structure (blocks, sizes, which view goes where, fonts, traffic lights) | `layouts/poke.json` |
| Colours | `themes/poke.json` (`ansi` and the main UI `colors`), `mods/poke/views/ui.js` `C` (type/chip accents) |
| Codex menu launch command | `programs.json` (`programs.codex.run`) |
| Keys | `keys.json` (`cmd-j` → `poke.jump`, `cmd-k` → right panel, `cmd-b` → sidebar) |
| Which view a block shows | `mods/poke/main.js` (`props.view` → module) |
| Sidebar top: logo, search, fold button | `views/brand.js` (`narrow()`, `toggleSidebar()` are shared with the other sidebar views) |
| Sidebar: spaces list | `views/spaces.js` (folded: `rail()`) |
| Sidebar: agents list | `views/agents.js` (folded: `railList()`) |
| Sidebar: pokeTokenBar companion card / egg | `views/companion.js` |
| Tab bar, right-panel toggle button | `views/tabs.js` |
| Pane header (avatar, status, model, ctx) | `views/pane.js` |
| Right panel header tabs (Usage / Dex / Changes / Files) | `views/paneltabs.js`. The stack `poke-panel-body` is in the layout. Changes and Files are Herdr's shipped mods. |
| Usage cards, 14-day chart, Dex grid | `views/panel.js` |
| Limit labels ("Session", "Weekly"), reading `limits.json` / `usage.json` | `views/store.js` |
| Status bar | `views/status.js` (also hosts behaviours and the `jump` command) |
| Avatars, pills, sprite lookup, GIF redraw | `views/ui.js` (`avatar`, `sprite`, `alive`, `pill`) |
| Agent rows (Herdr data → what views draw; provider; labels) | `views/data.js` |
| Things that happen: partner assignment, cries, Dex sync, Codex attach, evening dim | `mods/poke/behaviours.js` |
| Pure logic (urgency, groups, chips, partner pick, growth formula, formatting) | `mods/poke/lib/*.js`, with tests in `tests/mod-lib/` |
| Label caching, failure backoff, `hc label` timeout | `helper/internal/labels/labels.go` |
| CLI dispatch and helper build | `helper/main.go`, `scripts/build.sh` |
| Mod registration and bundled UI fonts | `mods/poke/mod.json`, `mods/poke/fonts/` |
| Node entry points for hc | `scripts/nvm-shim` |
| Claude statusline → labels and limits | `helper/internal/claude` |
| Codex → labels and limits (wrapper and `--attach`) | `helper/internal/codex` (`wrap.go`, `attach.go`, `find.go`, `rollout.go`) |
| Sprites, cries, names, GIF normalisation | `helper/internal/assets` (`assets.go`, `normalize.go`) |
| 14-day usage aggregation (launchd every 10 min) | `helper/internal/usage` |
| State files and locks | `helper/internal/state` (`~/.local/state/herdr-poke/`) |
| Install and undo | `scripts/install.sh`, `scripts/uninstall.sh`, `scripts/settings_edit.py` (configuration journal), `scripts/install_files.py` (installed-file ownership/backups), tests in `tests/scripts/` |

Seen is a deduplicated snapshot of partners on open agent sessions across spaces.
The existing 2-second behaviour tick replaces it only when membership changes.

## Data flow in one paragraph

Herdr gives the mod `agents()`, `spaces()`, `terminals()` (status, labels, focus). The Go
helper `~/.local/bin/herdr-poke` adds what Herdr lacks:
- For Claude, `statusline` runs as Claude's statusLine command.
- For Codex, the `codex -- …` wrapper (the zsh alias) does it. A Codex started any other
  way gets periodic `codex --attach <t>` probes from `behaviours.js`, even if old
  model labels remain. Local bare Codex foreground processes are supported; the
  helper cannot inspect a remote machine's PID or files.

Both write terminal labels `poke.model`, `poke.effort`, `poke.ctx` and
`~/.local/state/herdr-poke/limits.json`. `usage` writes `usage.json`. Sprites, cries and
names go to `~/.config/herdr-app/poke-assets/`, which the mod reads as
`poke-assets/<file>`. pokeTokenBar's folder
(`~/Library/Application Support/PokeTokenBar`) is read only.

## Gotchas (each one cost real time)

- **Hot reload:** Herdr reloads a mod whenever any non-hidden file inside
  `mods/<name>/` is written. Anything written at runtime goes in `poke-assets/` or
  `~/.local/state/herdr-poke/`. Writing into `mods/poke/` at runtime loops the reload
  until QuickJS runs out of memory.
- **GIFs advance only when the window redraws.** `ui.js` `alive()` wraps a sprite in a
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
  - get node from the `~/.local/bin/{node,npm,npx}` shims (`scripts/nvm-shim`);
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
- **Codex needs `--no-daemon`.** Without it, `lsof` can't tie the process to its
  rollout and `HC_TERMINAL_ID` is lost.

## How to verify a change

1. `herdr-app check mods/poke` must print `check passed`. `herdr-app status` must show
   `"errors": []`.
2. Logic: `npm test` (mod `lib/`) and `cd helper && go test ./...`. Installer:
   `PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tests/scripts`.
   Installer integration uses a temporary HOME and a stub builder; launchd and
   the live app are excluded. Fix with a failing test first.
3. Visual: `herdr-app shot out.png` saves the last frame without the macOS traffic
   lights. With the lights, find the window id (`CGWindowListCopyWindowInfo`, owner
   "Herdr") and run `screencapture -x -o -l <id> out.png`. Captures are 2×. Crop before
   looking. `sips -c` crops around the centre, so use a small Swift or CG script.
4. Sidebar rail: `herdr-app do layout.resize '{"id":"sidebar","px":92}'` to fold, `262`
   to restore. Put the width back when done.
5. Helper changes: `cd helper && go build -o ~/.local/bin/herdr-poke .`. A hidden test
   terminal (no `herdr.space` label, so the app never shows it):
   `hc run --bg --no-notify --name x-test -- codex --no-daemon "Reply only OK."`, then
   `hc kill`, then `hc rm`.
6. Never touch the user's other terminals.

## Rules

- Commits follow `~/.claude/commit-convention.md`: `type(herdr-poke): 한국어 명사형` title,
  body lines ending in ~함/~됨. No AI signature in commits or PRs. Pushing to
  `keoneokeoneo/herdr-poke` (private) is fine.
- Docs go in `docs/specs/` and `docs/plans/`, never `docs/superpowers/`. No commit
  hashes in docs.
- pokeTokenBar files are read only. Never open or decrypt its `usage-cache.json`.
- Back up before editing `~/.claude/settings.json`, `~/.zshrc`, `~/.codex/config.toml` or
  Herdr's `settings.json`.
- UI text is English. Only the Pokémon are pixel art. The theme stays dark and calm.

## Outside the repo (what a new machine needs)

`scripts/install.sh` sets up most of this; check it after a migration. Close Claude
sessions before running it, because Claude may rewrite its settings concurrently.
Missing configuration files are created. An absent Codex alias is added; an exact
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
| Claude statusLine → `herdr-poke statusline` | `~/.claude/settings.json` | install.sh (manifest) |
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
| Attach health probe | 60 seconds per Codex | One helper; held flock exits before querying hc |
| Attach foreground validation | 5 seconds per live attach | One bounded `hc get`; failure exits for later retry |
| Rollout discovery / tail | about 1 second | `lsof` until found; then incremental file reads |
| Companion + Dex copy, evolution/level toasts | PTB folder watch (`companion-state.json`, `sprites/`) + 300 seconds | Three helper executions per run, coalesced; no unchanged file rewrite; only new variant GIFs normalized |
| Limits/usage/Dex load | filtered folder events + 15 seconds | Suppresses unchanged notify; concurrent loads coalesced |
| Status/panel time labels | 60 seconds | Countdown/expiry refresh |
| Agent/pane elapsed labels | 20 seconds per initialized view | Timers continue in hidden views |
| Companion card / events | 30 seconds + card folder watch | Reads PTB state; initial defaults lookup once |
| Usage aggregation | launchd every 600 seconds | Incremental log reads, but directory discovery walks log roots |

Asset jobs start three helpers each time pokeTokenBar writes its state (every few
minutes) and on the 5-minute fallback, plus partner fetches, retries and attach
probes. Claude statusline also runs two git commands normally (three on detached
HEAD), each with a two-second timeout. CPU impact has not been remeasured.

The brand button folds to a rail; `cmd-b` hides the sidebar. Double-click rename and
reorder are implemented for expanded spaces and tabs, not agent rows or rail spaces.
Pane footer/dirty/ahead, review completion badges, partner-based terminal renaming,
and a selectable by-repository type mode are deferred. Current scope is also recorded
in the design's Current state section.
