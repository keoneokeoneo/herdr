# Personal Herdr — repository guide

This repository owns a personal Herdr configuration with multiple layouts, mods and
themes, organised as independent **setups**: [Base](modes/base.md) (the neutral sidebar,
tab bar, usage bar and right panel), [Obsidian](modes/obsidian.md) (a vault reader) and
Poke ([development](modes/poke.md), [installation](modes/poke-setup.md); Pokémon partners on top of
Base). Read the setup's guide before changing its implementation, and [Customizing Herdr](herdr-dev.md)
for how Herdr loads these files, its traps and how to check a change.

## What lives where

| Responsibility | Location |
|---|---|
| Native Herdr resources | `layouts/<name>.json`, `mods/<name>/`, `themes/<name>.json` |
| Which resources a setup needs, and what it requires | `scripts/setups.json` |
| Selective installer | `scripts/setup.py` (`npm run setup:list`, `setup:install`, `setup:status`) |
| Optional setup helper | `helpers/<name>/` (Poke's helper is an independent Go module) |
| Setup installation tools | `scripts/<name>/` |
| JavaScript and installer tests | `tests/<name>/mod-lib/`, `tests/<name>/scripts/`, `tests/setup/` |
| Herdr how-to and traps | `docs/herdr-dev.md` |
| Setup guide | `docs/modes/<name>.md` |
| Design records and plans | `docs/specs/`, `docs/plans/` (dated history; the guides are the living docs) |
| Shared keyboard, launcher and app settings | `keys.json`, `programs.json`, `settings.json` |

The root matches Herdr's configuration home (`~/.config/herdr-app`). Keep native
resources in these directories; do not nest them beneath a separate `modes/` folder.
“Setup” or “mode” here is an organizational concept, not an additional Herdr API.
A layout arranges views, a mod implements views/commands, and a theme provides colors.
Mods are separate folders: they cannot import each other, so shared behaviour goes through
shared state, terminal labels, commands or a data file contract. Only extract shared
implementation when multiple setups actually need it.

## Layers

- **Data**: the usage data contract in `~/.local/state/herdr-app/usage/` (`claude.json`,
  `codex.json`, `daily.json`), one writer per file, described in `docs/modes/base.md`.
- **Base** (`mods/base`): reads the contract and draws the shell. It works with nothing else running.
- **Poke** (`mods/poke`): the persona. It reaches base only through the documented seam
  (shared keys and terminal labels listed in `docs/modes/base.md`); a test guards the list.
- **Obsidian** (`mods/obsidian`): independent; embedded in base's panel when base is installed.

## Adding a setup

1. Choose a unique name for its resources and command/state namespaces.
2. Add the needed layout, mods and theme in the native directories. Reuse shipped
   mods where useful; Herdr restores shipped files on startup, so do not edit them.
3. Add the setup to `scripts/setups.json` (its mods, layouts, themes, required setups,
   `layout:<name>` key sections, and an `installer` only if it edits shell or app settings).
4. Add a helper or installer only if the setup needs one, under its named directory.
   Runtime assets and state belong outside `mods/`: writes there trigger reloads.
5. Add tests under `tests/<name>/` and helper tests beside its source. `npm test`
   discovers `tests/*/mod-lib/*.test.mjs`; add explicit package scripts for any other
   test suite and for setup-specific installation.
6. Document dependencies, resource names, installation effects, runtime storage and
   verification in `docs/modes/<name>.md`, then add it to the README's setup table
   (`tests/base/mod-lib/docs-current.test.mjs` fails if the table, the guide or a path in the docs is stale).
7. Check changed mods with `herdr-app check mods/<name>`. When testing an installed
   setup, inspect `herdr-app status` and the UI, including any collapsible regions.

## Shared settings and key ownership

Root settings affect every layout, so setups keep their own bindings in `layout:<name>` sections of
`keys.json` (Herdr applies one while that layout shows). `global` holds only generic bindings. The selective
installer adds a setup's missing bindings and never overwrites a different one.

`programs.json` currently launches Codex through `herdr-poke`, and app sounds are disabled for Poke's cries;
those two files and `~/.zshrc` are edited only by installing the `poke` setup. Selecting another layout does
not remove them. Poke does not edit the global Claude `statusLine`: Claude's limits are written by the global
statusline (see `docs/modes/base.md`).

For each new setup, explicitly decide whether to reuse or change these defaults. Do not copy Poke's installer
as though it were a generic prerequisite. Back up shared settings before installation edits; retain the
install/uninstall ownership journals.

## Development rules

- `AGENTS.override.md` and `CLAUDE.md` route agents here. The existing
  `herdr-poke-dev` skill covers the Poke setup and its helper.
- Herdr generates `AGENTS.md`, `herdr.d.ts` and `gpui-kit.d.ts` in the installed home.
  Search those for API details; never edit generated references. They may be absent
  from a source-only checkout.
- `.gitignore` excludes generated files, shipped resources and runtime data. Extend
  it when a new setup introduces runtime assets; do not ignore custom source files.
- Keep docs in `docs/modes/`, `docs/specs/` and `docs/plans/`, whatever location another tool suggests.
  Do not add commit hashes to docs.
- Commit titles follow the repo history: `type(scope): 한국어 요약`, scoped to the relevant setup or the
  repository. (`~/Documents/work/commit-convention.md` applies only to work under `~/Documents/work`.)
  No AI signatures in commits or PRs.
- Never touch unrelated user terminals. Poke-specific visual and integration rules
  are in its guide; they are not requirements for every future setup.

## Path migration

The former `helper/`, `scripts/*` and `tests/{mod-lib,scripts}/` now live at
`helpers/poke/`, `scripts/poke/*` and `tests/poke/{mod-lib,scripts}/`. The shell and usage views moved from
`mods/poke` to `mods/base`, layout ids lost their `poke-` prefix and the usage files moved to
`~/.local/state/herdr-app/usage/`. Use the named npm commands or the new paths. Poke's binary name, Go
module, labels, asset paths, helper state path and launchd identifier are unchanged, so no reinstall of the
helper is needed. The local checkout and GitHub repository are named `herdr`; `origin` points to
`git@github.com:keoneokeoneo/herdr.git`.
