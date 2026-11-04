# Layer separation: usage, shell, Obsidian and Poke

Status: draft for Keno's approval. Nothing here is implemented.

## Intent

Keno wants features in this Herdr configuration to be independent of each other, so
that a feature (Poke most of all) can be changed or dropped later without breaking the
rest. Today the whole UI is wired to Poke although Poke is only a small part of it.
Keno may tire of the Pokémon theme; this project makes that a cheap decision.
Actually removing Poke is **not** part of this project.

A second, concrete reason: Keno replaced the global Claude Code `statusLine` with a
standalone binary (`~/.claude/statusline/statusline`, source `main.go` beside it). It
only renders text. The Poke UI got Claude's rate limits from `herdr-poke statusline`
writing `~/.local/state/herdr-poke/limits.json`, so Claude limits have been stale
since the switch (last write 14:03 KST on 2026-11-04). This is the first bug the
separation fixes.

## Current coupling (facts found in the repo)

- `mods/poke` mixes three kinds of things:
  - Poke persona: `views/companion.js`, `lib/companion.js`, `lib/partner.js`,
    `lib/voice.js`, Dex (`panel.js`), `views/brand.js`, cries in `behaviours.js`.
  - LLM usage: `views/status.js` (bottom bar), the Usage tab in `views/panel.js`,
    `lib/usage.js`, `views/store.js` (`loadUsage`, `keepFresh`).
  - Generic shell: `views/spaces.js`, `views/agents.js`, `views/tabs.js`,
    `views/paneltabs.js`, `views/pane.js`, `views/ui.js`, `views/data.js`.
- `layouts/poke.json` ids are all `poke-*` (`poke-panel`, `poke-footer`, `poke-status`,
  `poke-usage`, `poke-obsidian`) and `keys.json` uses `layout.toggle:poke-panel`.
- `layouts/obsidian.json` uses `"theme": "poke"`. `mods/obsidian/main.js` hides
  `poke-panel` on close and has a "Back to Poke" button.
- Data: `helpers/poke` (Go, binary `herdr-poke`) writes `limits.json`, `usage.json`
  (launchd job, about every 10 minutes) and terminal labels `poke.owner|provider|model|
  effort|ctx`. Its `statusline` subcommand is a byte-compatible port of the old Node
  statusline plus those side effects. Its `codex` subcommand wraps Codex, reads the
  rollout log and writes Codex limits and labels. `programs.json` launches Codex through
  it.
- `scripts/poke/settings_edit.py` rewrites the global `statusLine` to
  `herdr-poke statusline` and restores it on uninstall. This conflicts with Keno's
  standalone statusline.
- `keys.json` has Poke bindings (`cmd-j` → `poke.jump`, `cmd-shift-o` → `poke.notes`).
- `docs/guide.md` already says shared files keep Poke defaults and that setups are an
  organizational concept, not a Herdr API.

## Goals

1. LLM usage (bottom status bar and the right-panel Usage tab) works with no Poke code
   present or running.
2. Claude limits come from the global statusline, through a documented neutral data
   contract. Codex limits and daily usage reach the same contract.
3. The generic shell (sidebar spaces and agents, tab bar, panel tabs) no longer lives
   under `mods/poke`.
4. Poke contains only persona: companion, Dex, brand, cries, Poke theme assets, and
   anything that only makes sense with them.
5. The Obsidian mod and its layout depend on nothing Poke-named.
6. Poke's installer no longer touches the global Claude `statusLine`.
7. Poke layout = the neutral layout plus Poke's persona pieces.

## Non-goals

- Removing Poke, or writing an "uninstall Poke cleanly" flow (Keno: not now).
- UI/UX changes. That is the next, separate project, planned only after this one.
- A shared abstraction for hypothetical future setups. Extract only what two setups
  already use.
- Changing Herdr's shipped mods or generated files.

## Anti-goals (failure even if it "works")

- Usage or Obsidian visibly broken or empty when Poke's helper is not running.
- Renames that force a reinstall, break the launchd job, or lose existing usage history
  without a plan for it.
- Two writers for the same data file, or Poke still silently rewriting global Claude
  settings.
- Making Keno's global statusline depend on Herdr to render. Its Herdr side effect
  must be best-effort and must never delay or break the status line output.

## Decisions Keno made

- Order: layer separation first, UI/UX after.
- Claude rate limits are delivered by the **global statusline writing a neutral file**
  (not read by usage directly from Claude internals, not via `herdr-poke statusline`).
- Removal tooling is out of scope.

## Calls I made (Keno should check these)

- **Mod split:** three mods: `usage` (status bar view + Usage panel view), `shell`
  (spaces, agents, tab bar, panel tabs, pane helper, `ui.js`), `poke` (persona only).
  Alternative: keep shell inside one neutral mod together with usage. I prefer a
  separate `usage` because it is the only part with its own data contract.
- **Contract location:** `~/.local/state/herdr-app/usage/`, files `claude.json`,
  `codex.json`, `daily.json`. Same JSON shapes as today's `limits.json` entries and
  `usage.json` (provider → `{plan, windows[{minutes, used, resets_at}], updated}`),
  so views change little. Each file has exactly one writer.
- **Fallback:** usage views read the neutral files first and fall back to the old
  `~/.local/state/herdr-poke/` files while they are fresher or the neutral file is
  missing. Remove the fallback once writers are moved.
- **Writers:** Claude: global statusline (best effort, atomic write, ignored on
  failure). Codex and daily usage: stay in the Go helper for now but are retargeted to
  write the neutral paths; the helper keeps its name and launchd identifier (no
  reinstall). Renaming or splitting the helper is a later step, not this project.
- **Labels (Keno decided: context percentage may be dropped):** the Claude statusline
  no longer sets terminal labels. Sidebar agent rows stop showing the Claude context
  percentage. My call, to check: Claude model and effort come from the same statusline
  input, so they also disappear from Claude rows rather than adding a new hook; Codex
  rows keep model/effort/context because the Codex wrapper still sets them. Label
  names (`poke.*`) are unchanged in this project; the sidebar must render a row
  cleanly when labels are missing.
- **IDs:** layout ids lose the `poke-` prefix (`panel`, `footer`, `status`, `usage`,
  `obsidian`); persona-only blocks keep `poke-`. Keys that point to them are updated:
  `cmd-k` → `layout.toggle:panel`. `poke.jump` is generic (jump to waiting agent) and
  moves with `shell`, with a new command name.
- **Theme:** the Poke theme is the visual baseline for now. The Obsidian layout stops
  naming it by a Poke id; it uses a neutral theme file that Poke's theme extends or
  copies. Visual output must not change in this project.
- **Obsidian:** closing the reader hides the neutral `panel` id; the "Back to Poke"
  button becomes "Back" to the previous or default layout. `cmd-shift-o` opens the
  Obsidian tab through an `obsidian` command, not `poke.notes`.

## Further calls (from the builder check; Keno should check these too)

- **Cutover data:** `daily.json` is seeded once by copying the existing `usage.json`
  (same shape), so history is kept. After that the helper's usage collector writes only
  the neutral path. Starting fresh is not acceptable.
- **Fallback lifetime:** the old-path fallback exists only during this project. The last
  step removes it and deletes the now-dead `herdr-poke statusline` subcommand and its
  golden tests (Keno's global statusline already replaced it). One writer per file at
  the end.
- **Statusline change is in scope:** this project includes writing the small change to
  `~/.claude/statusline/main.go` and rebuilding it. The write is best-effort: it runs
  after the output is printed, uses a short bounded timeout, and ignores every error.
  Without it Goal 2 cannot be verified.
- **Renames without reinstall:** the installed binary path, launchd identifier, shell
  alias and install journals do not reference mod or layout ids, so id renames need no
  reinstall. The builder must confirm this by searching `scripts/` and tests, and fix any
  `poke-*` id the installers write (for example `scripts/obsidian/install.sh --poke`).
  `poke.jump` becomes `shell.jump` and `poke.notes` becomes `obsidian.notes`.
- **Neutral layout:** a new `layouts/base.json` (shell + usage + Changes/Files/Obsidian
  tabs, no persona). The Obsidian "Back" button switches to `base`. `layouts/poke.json`
  stays a second hand-maintained file; no generator. A test asserts that both layouts
  contain the same non-persona block ids so they cannot drift.
- **Themes:** copy, not inheritance. `themes/base.json` is the neutral theme (current
  Poke colors, no persona names); `themes/poke.json` stays as is. Copying is accepted.

## Refinements found while planning (supersede the calls above where they differ)

- **One neutral mod, `base`, instead of `usage` + `shell`.** Herdr mods are separate
  folders and mod code is not shared across them; `<mod>.<command>` only reaches
  shown copies. The bottom bar mixes usage figures with the agent summary and the
  "jump to waiting" action, so splitting it would force duplicate code. `base` holds
  sidebar, tab bar, pane header, panel tabs, bottom bar and the Usage tab. The
  data-contract separation (neutral files, one writer each) is unchanged.
- **Commands:** `cmd-j` → `base.jump` and `cmd-shift-o` → `base.notes`, both handled by
  `base`'s bottom-bar copy (always shown). `obsidian.notes` is not used: the Obsidian
  copy inside the panel is often not shown, so it would never receive it.
- **Persona seam.** The sidebar and tab bar draw agent avatars and tab-type pills. They
  keep reading Poke's existing shared-state keys and labels (`poke.sprites`,
  `poke.evening`, `poke.typeMode`, `poke.typeSeed`, label `poke.partner`) and fall back
  to a plain letter avatar and no type pill when those are absent. No import crosses
  mods. A test lists the allowed `poke.*` keys so the seam cannot grow silently.
- **Panel tabs** are declared per layout (`props.tabs` on the panel-tabs block), so
  Poke's layout adds its Dex tab and `base` does not.
- **Brand logo** is a layout prop (`props.logo`): `base` shows text only, Poke passes
  its Pokéball.

## Added by Keno after approval

- **Selective install:** Keno can install `base`, `obsidian` and `poke` independently
  (`poke` requires `base`). A manifest (`scripts/setups.json`) and installer
  (`scripts/setup.py`) do this. Key bindings move into per-layout `layout:<name>` sections
  of `keys.json`, so an unselected setup adds nothing. Only `poke` touches `~/.zshrc`,
  `programs.json` and `settings.json` sounds.
- **Docs current:** README, `docs/guide.md` and `docs/modes/*` are rewritten to the final
  state, and a test compares them with the manifest.

## Constraints

- Follow `docs/guide.md`: native resources stay in `layouts/`, `mods/`, `themes/`;
  runtime state stays out of `mods/` (writes there trigger reloads); do not edit
  Herdr-shipped files; never touch unrelated user terminals.
- Back up shared settings (`keys.json`, `programs.json`, `settings.json`, global
  `~/.claude/settings.json`) before edits. Keep install journals consistent.
- Check each changed mod with `herdr-app check mods/<name>`; `npm test` must pass and
  tests move with their code (`tests/<name>/`). Inspect `herdr-app status` and the real
  UI, including collapsible regions, after switching layouts.
- Keno's statusline binary lives outside this repo (`~/.claude/statusline`). Changes
  there are small, separately reviewed, and rebuilt with the command in its header.
- Docs go in `docs/modes/`, `docs/specs/`, `docs/plans/`; add `docs/modes/usage.md`
  (contract and verification); update `docs/guide.md` and the README setup table. No
  commit hashes in docs. No AI signatures in commits or PRs.

## Left to the builder

- Exact JSON field names where the existing shapes need small changes.
- Order of the migration and whether to do it in one branch or stages.
- Test strategy, including a fake-data harness for the usage views.
- Whether `shell` helper code should be shared with the Obsidian mod's styling.

## Success looks like

With the Poke persona mod disabled (layout without its blocks), the sidebar, tab bar,
bottom usage bar, Usage tab, Changes/Files/Obsidian tabs and all shared shortcuts work;
Claude limits update live from the global statusline; Codex limits and daily usage
still update; no `poke-` id is required by any non-Poke file; Keno's global
`statusLine` is untouched by any Poke script.
