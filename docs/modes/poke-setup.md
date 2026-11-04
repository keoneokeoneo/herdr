# Poke — installation

A calm, dark work setup for Herdr: a layout and a persona mod on top of [Base](base.md) that show every Claude Code and
Codex session at a glance — who needs you, who is working, how much of the 5h and weekly
limits are left — with a random Pokémon partner on each session and your pokeTokenBar
companion in the sidebar. Only the Pokémon are pixel art. Poke requires Base; you can install
Base (and Obsidian) without it, see the README.

## Changing it

Start with `docs/guide.md` and `docs/modes/poke.md`: a map from each part of the window to its files, the gotchas,
how to verify, and what lives outside the repo. Claude Code loads it through `CLAUDE.md`.
Codex loads it through `AGENTS.override.md`; Herdr's generated `AGENTS.md` stays as the
API reference. Both tools get the `herdr-poke-dev` skill from `.agents/skills/`.

## Prerequisites

- Herdr: `brew install --cask herdr-app`
- pokeTokenBar: `brew install --cask chattymin/tap/poke-token-bar` (read only; never edited)
- Go matching `helpers/poke/go.mod` or newer: `brew install go`
- Python 3 (installer), Jetendard terminal font (install separately)
- nvm with an installed Node version for the optional hc node/npm/npx shims
- Claude Code and a Codex CLI that supports `--no-daemon`

## Install

1. Follow the [new Mac migration checklist](../plans/2026-11-04-machine-migration.md)
   to back up existing Herdr settings and place the repo at `~/.config/herdr-app`.
   Clone into a staging folder first; do not force-checkout over existing settings.
   Keep `programs.json`: it makes menu-launched Codex use the wrapper and `--no-daemon`.
2. Run `python3 scripts/poke/preflight.py`. Required tools, the Go version from
   `helpers/poke/go.mod`, and Codex `--no-daemon` support must pass. Node/nvm, Jetendard
   and pokeTokenBar state produce optional-feature warnings. Login and live UI
   verification remain manual.
3. Run `bash scripts/poke/install.sh` (the same as `python3 scripts/setup.py install poke`). It repeats
   preflight before any installation writes, then
   - installs the `base` and `poke` mods, layouts, themes and the `layout:` key bindings
     (`scripts/setup.py`; nothing is copied when the repo already is `~/.config/herdr-app`),
   - builds `~/.local/bin/herdr-poke`,
   - backs up and edits `~/.zshrc` (`codex` runs through `herdr-poke codex -- codex --yolo --no-daemon`)
     and Herdr's `settings.json` (Herdr's own attention sounds off; partners' cries replace them).
     It never edits `~/.claude/settings.json`: Claude's limits come from your global statusline,
     see [Base](base.md),
   - copies the companion sprite, collects 14 days of usage, adds a launchd job
     (`dev.keno.herdr-poke.usage`, every 10 min) and switches Herdr to the `poke` layout.
4. Open a new zsh. Start Herdr from the Dock, not from inside a Claude session
   (a Herdr launched from Claude passes `CLAUDE_CODE_CHILD_SESSION` to its terminals and
   their transcripts stop saving).

Keys (layout `poke`, same as `base`): `⌘J` next session waiting for you (oldest first), `⌘K`
show/hide the right panel, `⌘⇧O` Obsidian tab, `⌃1–9` spaces.

## Undo

`scripts/poke/uninstall.sh` uses the configuration and installed-file journals, not the
newest whole-file backup. It restores original settings and owned files, preserving
later user replacements and unjournaled legacy files. The current layout, repo,
assets, runtime state and backups stay. Manual security defaults and external review
skills are not removed. See `docs/guide.md` for recovery and migration details.

## Troubleshooting

- `herdr-app status` — errors from layouts and mods.
- `*.error` files next to a layout or mod name what failed to load.
- `ctrl-alt-shift-escape` resets Herdr to its default layout if the window is unusable.
- `herdr-app check mods/poke` and `herdr-app check mods/base` — compile the mods without reloading.
- Helper state lives in `~/.local/state/herdr-poke/`; the usage data the UI reads is in
  `~/.local/state/herdr-app/usage/` (`claude.json`, `codex.json`, `daily.json`).
