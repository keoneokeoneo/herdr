# herdr-poke

A calm, dark work setup for Herdr: a layout and one mod that show every Claude Code and
Codex session at a glance — who needs you, who is working, how much of the 5h and weekly
limits are left — with a random Pokémon partner on each session and your pokeTokenBar
companion in the sidebar. Only the Pokémon are pixel art.

## Changing it

Start with `docs/guide.md`: a map from each part of the window to its files, the gotchas,
how to verify, and what lives outside the repo. Claude Code loads it through `CLAUDE.md`.
Codex loads it through `AGENTS.override.md`; Herdr's generated `AGENTS.md` stays as the
API reference. Both tools get the `herdr-poke-dev` skill from `.agents/skills/`.

## Prerequisites

- Herdr: `brew install --cask herdr-app`
- pokeTokenBar: `brew install --cask chattymin/tap/poke-token-bar` (read only; never edited)
- Go: `brew install go`
- Claude Code and the Codex CLI

## Install

1. Launch Herdr once so it creates `~/.config/herdr-app`.
2. Put this repo there:
   ```sh
   cd ~/.config/herdr-app
   git init && git remote add origin <repo url> && git fetch && git checkout -f main
   ```
3. Run `scripts/install.sh`. It
   - builds `~/.local/bin/herdr-poke`,
   - backs up and edits `~/.claude/settings.json` (statusLine → `herdr-poke statusline`),
     `~/.zshrc` (`codex` runs through `herdr-poke codex -- codex --yolo --no-daemon`) and
     Herdr's `settings.json` (Herdr's own attention sounds off; partners' cries replace them),
   - copies the companion sprite, collects 14 days of usage, adds a launchd job
     (`dev.keno.herdr-poke.usage`, every 10 min) and switches Herdr to the `poke` layout.
4. Open a new zsh. Start Herdr from the Dock, not from inside a Claude session
   (a Herdr launched from Claude passes `CLAUDE_CODE_CHILD_SESSION` to its terminals and
   their transcripts stop saving).

Keys: `⌘J` next session waiting for you (oldest first), `⌘K` show/hide the right panel,
`⌃1–9` spaces.

## Undo

`scripts/uninstall.sh` restores the newest `*.herdr-poke-backup-*` of each edited file,
removes the launchd job, the binary and the node shims, and switches Herdr back to `main`. The repo stays.

## Troubleshooting

- `herdr-app status` — errors from layouts and mods.
- `*.error` files next to a layout or mod name what failed to load.
- `ctrl-alt-shift-escape` resets Herdr to its default layout if the window is unusable.
- `herdr-app check mods/poke` — compile the mod without reloading.
- Helper state lives in `~/.local/state/herdr-poke/` (`limits.json`, `usage.json`).
