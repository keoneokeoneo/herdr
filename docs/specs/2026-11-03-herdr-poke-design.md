# herdr-poke — design

Date: 2026-11-03 · Owner: Keno · Status: approved
Readable version: https://claude.ai/artifact/6FnMqh5cpHSRHKiWziVrmb · Visual spec: prototype https://claude.ai/artifact/6mQsSo45wkmNmEQGS2WjEZ (v6, source in `docs/prototype/`)

Where the prototype and this document disagree, this document wins.

## Current state (updated 2026-11-03, after v1 shipped)

The sections below are the v1 design. Where they differ from this list, this list wins.
`docs/guide.md` maps each item to files.

- **Assets** live in `~/.config/herdr-app/poke-assets/`, not inside the mod. A write
  inside `mods/poke/` reloads the mod. The helper squares every sprite GIF so all frames
  fill one square canvas, because Herdr stretches each frame to its box. Avatars fit the
  sprite into the circle; no crop, no zoom.
- **Animation:** every sprite animates (companion, all agents, Dex), not only working
  agents. GIF sprites are wrapped in a sub-pixel native loop; static PNGs do not request animation redraws.
- **Sidebar:**
  - The brand row is a pokeball and "Herdr", with search and a fold button.
  - Folded, the sidebar becomes a 92px rail:
    - Spaces show as letter chips with a waiting/new dot.
    - Agents show as faces only, grouped by short rules.
    - The companion shows as sprite, level and a thin bar.
    - A › button unfolds it.
  - Traffic lights are centred on the brand row (`traffic_lights: [12, 10]`).
- **Spaces** have a + (new space). Spaces, tabs and agents have right-click menus (the
  shipped `menu` mod). Expanded spaces and tabs have double-click rename and drag
  reorder; agents and folded spaces do not. Agents can be pinned.
- **Dex** Caught is a grid of sprite tiles, using pokeTokenBar's own art and ✦ for shiny.
  Active lists unique assigned partners from currently open agent sessions across spaces,
  including idle/waiting sessions. It refreshes every 2 seconds and removes partners
  when their last session disappears; it is not a cumulative encounter history.
- **Right panel** tabs are Usage, Dex, Changes and Files. Changes and Files are Herdr's
  shipped mods. The panel toggle is at the right end of the tab bar, with `⌘K`.
- **Usage labels** are "Session" (5h) and "Weekly". A window a provider never reports is
  hidden, not shown as "–". Keno's Codex plan reports weekly only.
  Both provider cards omit plan names; internal identifiers such as `prolite` are not displayed.
- **Codex menu launch:** `programs.json` starts `herdr-poke codex -- codex --yolo --no-daemon` directly. The menu launcher does not expand zsh aliases. Existing sessions keep their original launch mode.
- **Codex labels without the wrapper:** a Codex started by `hc run` or a script gets
  `herdr-poke codex --attach <t>`, probed by the mod every minute even when a Codex row retains model labels.
  It finds the rollout via `lsof`; both attach and wrapper require a per-terminal
  flock before monitoring. Attach rechecks foreground every 5 seconds. This supports
  local bare Codex, not remote terminal PIDs.
- **Terminals started by the hc server:**
  - There are no permission prompts. The global defaults are Codex
    `approval_policy="never"` with `sandbox_mode="danger-full-access"`, and Claude
    `permissions.defaultMode="bypassPermissions"`. Keno approved this on 2026-11-03.
  - Node comes from the `~/.local/bin` nvm shims.
- **Fonts:** terminal `Jetendard` at 10, UI `IBM Plex Sans KR`. The theme uses the One
  Double Dark ANSI palette.
- The egg state (no active companion) has its own static card.
- **Post-review corrections:** missing cries/names retry independently of sprite
  presence. Dex caches shiny separately and normalizes only newly copied GIFs.
  Status bar uses Session/Weekly and omits windows the provider does not report.
- **v1 features deferred:** partner-based terminal renaming, pane footer with
  dirty/ahead/type details, review completion labels (`N to fix` / `clean`),
  and a selectable by-repository type mode. Existing review links still work.
- **Data differences from v1:** Today in Usage/status comes from the 10-minute usage
  aggregation; the companion's Fed today comes from PTB. Codex resume uses open-file
  lookup, not SQLite. Legacy fallback spans -0.5 to +10 seconds; `--no-daemon` and
  attach disable fallback. Assets are outside the mod. Repository publishing is
  permitted by the guide; the old local-only restriction below is historical.
- **Install/undo:** creates missing config, adds the wrapper alias and PATH, refuses
  custom aliases, journals before configuration edits, and records installed-file
  ownership. Uninstall restores owned files, preserves user replacements and leaves
  the selected layout, assets, state and backups intact. Security defaults and review
  skills remain manual. Jetendard is a separate font prerequisite.

## Intent

Keno works in the Herdr 2 desktop app (`herdr-app`, on top of herdr-core `hc`) with 2–3 Claude Code sessions, plus extra Claude or Codex sessions opened mid-task for review. What Keno mostly does in the window is **check how far each session has got and which one needs them**. Keno also raises a Pokémon in **pokeTokenBar** (menu-bar app that turns token usage into a companion).

This project turns Herdr into a calm, cute workspace that makes "who needs me / who finished / who's working" readable at a glance, keeps the Claude and Codex **5h and weekly limits** always visible, and puts the companion and Pokémon avatars into the UI.

## Goals

- A Herdr layout + mods reproducing prototype v6: tab bar, left sidebar (spaces, agents, companion), panes with a rich header, collapsible right panel (Changes / Usage / Dex), status bar.
- A Go binary `herdr-poke` that collects what Herdr doesn't have (model, effort, ctx %, limits, 14-day usage) for Claude Code and Codex.
- Installable on a fresh Mac by following the README.

## Non-goals (v1)

- Garden; extra gamification (quests, badges, combos, eggs, wild encounters, trainer card); review battle (HP/VS); messenger, evolution forecast, token weather.
- Partner evolution — partners are random and never evolve.
- An "error / tests failed" state. Herdr's app `Status` is `working | blocked | done | idle`; v1 uses exactly those. ⌘J targets `blocked`.
- Writing to pokeTokenBar files. Read-only, always. Never read/decrypt its `usage-cache.json`.

## Anti-goals

- Pixel-art chrome (only Pokémon sprites are pixel art). Bright/loud theme.
- Keybindings that swallow terminal typing. Timer-driven animation.
- Anything that can corrupt pokeTokenBar data or Keno's Claude/Codex setup.

## Constraints (verified against source, 2026-11-03)

Reference clones under the session scratchpad: `herdr-app`, `herdr-core`, `herdr-docs`, `ref/codex`, `ref/herdr`, `ref-ptb/PokeTokenBar`. Authoritative API: `~/.config/herdr-app/AGENTS.md`, `herdr.d.ts`, `gpui-kit.d.ts` (rewritten each start; app 0.1.12).

- Mods: QuickJS + gpui-kit, no DOM/CSS/Node/`setTimeout`. Draw only in own block or overlays/floats. `herdr-app check <mod>` must pass; `herdr-app shot` for visual checks. Shipped mods are restored at start (edited ones get moved to `<name>-edited`) — clone, never edit.
- `image()` plays animated GIFs natively (frame delays, transparency; pauses when the window is inactive). Paths resolve **only inside `~/.config/herdr-app`** — copy sprites in.
- `terminals()` exposes `turn` and `labels` live; `agents()` does not. `hc label $HC_TERMINAL_ID k=v` works from inside a terminal; labels persist in core; write at human speed (≈ once per turn).
- `readTextFile` reads any absolute path, ≤ 1 MiB, no `~` expansion (use `home()`); `watchFolder` needs an existing folder (≤ 64 watched).
- Built-in notifications are Rust-side with hard-coded wording `"{name} needs you"` / `"{name} finished"` (`{name}` = terminal name). No hook. A mod `notify` adds a second toast. Sounds: `settings.json` `sounds.enabled` / `done` / `blocked` only. `playSound` (MP3) from a mod ignores `sounds.enabled`.
- Keys: `keys.json` `global` → `"cmd-j": "<mod>.<command>"` (dot form) works while a terminal has focus; reaches shown mod copies only; `cmd-j`/`cmd-k` are free. Focus with `run("agent.focus", {terminal})`.
- Layout root keys `font`, `font_size`, `ui_font`, `theme`, `colors`; fonts from `fonts/` or `mods/<m>/fonts/`. IBM Plex isn't installed → ship the OFL font files.
- Codex 0.166.0 emits OSC 7501 itself (`app=codex`, states incl. `blocked:kind=…`) — pass it through untouched. Must run with `--no-daemon` (daemon mode loses `HC_TERMINAL_ID`). Rollout path `~/.codex/sessions/YYYY/MM/DD/rollout-<local ts>-<uuid>.jsonl`, file created on first turn; no flag/env exposes it.
- Claude Code 2.1.317 statusline stdin: `model.display_name`, `effort.level`, `context_window.*`, `rate_limits.five_hour|seven_day {used_percentage, resets_at}` (optional). Emits OSC 7501 natively.
- Installed: Herdr.app (`~/Applications`, home `~/.config/herdr-app` exists), `hc` 0.1.17, Claude Code, Codex (`alias codex='codex --yolo'`, `~/.zshrc:46`), pokeTokenBar, Go 1.27.3, `gh`. No ffmpeg.
- Homebrew sources for README: cask `herdr-app`; `chattymin/tap/poke-token-bar`; `go`.

## Decided

### Window (prototype v6)

- **Tab bar**: tab = most urgent agent's avatar, name, ⎇ if worktree, state pill with symbol (`! needs you`, `● N working`, `✓ new`, `✓ done`, `idle`), underline in the tab's random type colour at ~40%.
- **Sidebar — Spaces**: chip letter, name, branch, `⌃N`, `N waiting`, `N new`, stacked avatars of that space's agents (max 4 + `+n`, by urgency, click = jump). Start from a clone of the shipped `spaces` mod.
- **Sidebar — Agents**: groups Needs you / Working / Done / Idle; row = avatar, space chip, tab name, ⎇, chips, elapsed, sub-line (ask text / model·effort·ctx / "New · not checked yet" / "↳ reviewing X · N found").
- **Sidebar — Companion card**: pokeTokenBar active Pokémon: sprite, name, ✦ shiny, level, nature, ability, types, next form + progress bar, fed today, Dex/Seen counts.
- **Panes**: header = avatar, program, status pill (`●` working, `!` blocked kind, `✓` done, `‖` idle) + elapsed, chips, model, effort pill, ctx meter. Footer = space chip, worktree, branch, dirty, ahead, muted type dot + name.
- **Review link**: reviewer pane top `↳ Reviewing <tab> · N found`; reviewed pane top `↳ Codex is reviewing this · N found`; after: `N to fix` / `✓ clean`; both click to jump. Link = label `poke.review_of=<terminal>` on the reviewer (set by a "review this" action or `hc label`).
- **Right panel** (collapses to a rail with both providers' 5h/wk): *Changes*, *Usage* (Claude card + Codex card: 5h and weekly with reset countdown, window decided by `window_minutes`, `–` when absent — Keno's Codex Pro Lite has weekly only; Today split; 14-day bars + average), *Dex* (Caught from pokeTokenBar, Seen = partners met, empty state).
- **Status bar**: Claude `5h x% · wk y%`, Codex same, Today, `✓ N new`, "Jump to waiting · N ⌘J".
- **Evening**: 18:00–06:00 slightly dimmer; idle/done avatars drowsy.
- **Keys**: `⌘J` next blocked session (oldest first, cycles), `⌘K` toggle right panel, `⌃1–9` spaces (Herdr default). Never `⌘P` or plain letters.

### Pokémon

- **Session partner**: random from all species (1–1025), stable for the terminal's life (store as a label), **excluding species in Keno's pokeTokenBar dex and the active companion's line**. No evolution. Sprites like pokeTokenBar: PokeAPI BW animated GIF for ≤ 649, static PNG above.
- **Motions**: working bob, blocked shake + `!`, idle grey, done-unread yellow dot.
- **Status chips — plain words (style A)**: `ctx 83%` (ctx ≥ 80%, tooltip: try /compact), `waiting 6m` (blocked ≥ 5 min). Unread-done is the dot / `new`, no chip.
- **Notifications** (one per event): set each agent terminal's name to its partner so Herdr's own toast reads `Cyndaquil needs you` / `Cyndaquil finished`; `sounds.enabled: false`; the mod plays the partner's cry (MP3) when an agent becomes blocked. Nature-voiced toasts (Jolly; table `V` in the prototype) only for companion level-up / evolution.
- **Seen**: partner species met are stored locally by this project.
- **Types**: random per tab (stable hash of branch + seed), reroll command; modes random / by repo / off.
- **Unread done**: Herdr's own `done` (driven by `herdr.seen`). No parallel system.
- **Companion progress**: pokeTokenBar formula — next-stage cost = `graduationTotal(rarity) × (stage+1) / (k(k+1)/2)` × `growthDifficulty` (UserDefaults `io.github.chattymin.poketokenbar`, default 1.0); progress = `active.usedAtStage / cost`. Totals: common 0.75B, uncommon 1.875B, rare 3B, legendary 6B. (Today: Charmeleon rare, k=3, stage 1 → 1.0B, 70.9%.)

### Data

| What | Source |
|---|---|
| Sessions, status, space/tab, focus, unread, turn, labels | `agents()`, `spaces()`, `terminals()` |
| Worktree, branch, dirty, ahead/behind | shipped `spaces`/`worktree` approach (`exec git`) |
| Claude model/effort/ctx | statusline stdin → `herdr-poke statusline` → labels `poke.*` |
| Claude 5h/weekly | statusline stdin → `limits.json` |
| Codex model/effort/ctx | rollout `turn_context.payload.{model,effort}`, `token_count.payload.info.last_token_usage.total_tokens` + `model_context_window` (Codex's 12k-baseline formula) → labels |
| Codex limits | rollout `payload.rate_limits.{primary,secondary}` → `limits.json` |
| 14-day usage | Claude `~/.claude/projects/**/*.jsonl` assistant `message.usage` (dedupe by `message.id`, include `subagents/`); Codex `token_usage_record.payload.usage` (dedupe `response_id`) → `usage.json` |
| Today | pokeTokenBar `last-snapshot.json` |
| Companion, dex | pokeTokenBar `companion-state.json`, sprites `sprites/` (copied into the mod), `pokemon-details-v1` |

### `herdr-poke` (Go, single binary)

- `statusline` — Claude `statusLine.command`. Parse stdin, set labels on `$HC_TERMINAL_ID` (only when values change), update `limits.json`, print **exactly** what `~/.claude/statusline.mjs` prints today (account · folder · git · model:effort · ctx % · 5h/7d bars + countdown · output style · elapsed; same colours and formulas).
- `codex -- <args>` — alias `codex='herdr-poke codex -- codex --yolo --no-daemon'`. Start Codex as a child with inherited stdio (no PTY, exit code passed through). Find its rollout: open-file lookup on the child pid (`lsof`) after the first turn; fallback = new file in today's folder with matching `cwd`, no parent/fork id, start time within ~2 s (mark uncertain). `codex resume` → `~/.codex/state_5.sqlite` `threads`. Tail it, set labels, update `limits.json`.
- `usage` — incremental 14-day aggregation (remember offsets) → `usage.json`.
- State dir `~/.local/state/herdr-poke/` (`limits.json`, `usage.json`, offsets). Mods read with `readTextFile` + `watchFolder`.
- Never break Claude or Codex: on any helper error, still print a statusline / still run Codex.

### Repo and install

- Repo = `~/.config/herdr-app/`, `git init` in place. **Local only for now**; later private on GitHub `keoneokeoneo` (fresh install: launch Herdr once → `git init` → `remote add` → `fetch` → `checkout`).
- `.gitignore` everything Herdr writes/rewrites (`AGENTS.md`, `*.d.ts`, `icons/`, `frame/`, shipped themes/layouts/mods, `view.json`, `shared.json`, `data/`, `seen.json`, `installed.json`, `window*.json`, `telemetry/`, `update-notes*`, `*.sock`, `*.error`, `*.reset-*`, `jsconfig.json`). Track our layout, mods, `keys.json`, `settings.json`, helper source, assets, fonts, docs, scripts.
- Docs live in `docs/specs/` (this file) and `docs/plans/`.
- **Apply to Keno's real setup**: Claude `statusLine.command`, `~/.zshrc:46` alias, Herdr `settings.json` sounds, show the new layout. Back up each file first (timestamped, beside it); `scripts/uninstall.sh` restores all. Verify in a real Claude and a real Codex session.
- README: prerequisites, install, undo.
- Commits: `~/.claude/commit-convention.md`; no AI signatures.

## Left to the builder

Mod split; layout JSON; exact colours within the prototype palette (`--bg #1b1d26`, working `#7dd3b0`, blocked `#f6a58a`, done `#b9a6f5`, accent `#f3d17a`, sky `#8ec5f5`); label names; Go package layout and tests; how sprites/cries are fetched (PokeAPI sprites + cries; cries need MP3 — no ffmpeg, pick a converter or source); how the "review this" action opens a reviewer.

## Verify early in the plan

1. Codex rollout ↔ child pid via `lsof` with `--no-daemon` (two runs in one folder).
2. Renaming an agent terminal to its partner shows in Herdr's toast and doesn't break anything else that displays the name.
3. `hc label` from inside the statusline process (statusline runs often — confirm no lag).
