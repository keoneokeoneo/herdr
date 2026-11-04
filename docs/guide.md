# Personal Herdr — repository guide

This repository owns a personal Herdr configuration with multiple layouts, mods and
themes. Poke is one setup. Read the setup's guide before changing its implementation:
[Poke](modes/poke.md), [Poke installation](modes/poke-setup.md).

## What lives where

| Responsibility | Location |
|---|---|
| Native Herdr resources | `layouts/<name>.json`, `mods/<name>/`, `themes/<name>.json` |
| Optional setup helper | `helpers/<name>/` (Poke is an independent Go module) |
| Setup installation tools | `scripts/<name>/` |
| JavaScript and installer tests | `tests/<name>/mod-lib/`, `tests/<name>/scripts/` |
| Setup guide | `docs/modes/<name>.md` |
| Design records and plans | `docs/specs/`, `docs/plans/` |
| Shared keyboard, launcher and app settings | `keys.json`, `programs.json`, `settings.json` |

The root matches Herdr's configuration home (`~/.config/herdr-app`). Keep native
resources in these directories; do not nest them beneath a separate `modes/` folder.
“Setup” or “mode” here is an organizational concept, not an additional Herdr API.
A layout arranges views, a mod implements views/commands, and a theme provides colors.
Only extract shared implementation when multiple setups actually need it.

## Adding a setup

1. Choose a unique name for its resources and command/state namespaces.
2. Add the needed layout, mods and theme in the native directories. Reuse shipped
   mods where useful; Herdr restores shipped files on startup, so do not edit them.
3. Add a helper or installer only if the setup needs one, under its named directory.
   Runtime assets and state belong outside `mods/`: writes there trigger reloads.
4. Add tests under `tests/<name>/` and helper tests beside its source. `npm test`
   discovers `tests/*/mod-lib/*.test.mjs`; add explicit package scripts for any other
   test suite and for setup-specific installation.
5. Document dependencies, resource names, installation effects, runtime storage and
   verification in `docs/modes/<name>.md`, then add it to the README's setup table.
6. Check changed mods with `herdr-app check mods/<name>`. When testing an installed
   setup, inspect `herdr-app status` and the UI, including any collapsible regions.

## Shared settings and current coupling

Root settings affect every layout. Currently `keys.json` includes Poke bindings
(`cmd-j` → `poke.jump`, `cmd-k` → `poke-panel`), `programs.json` launches Codex through
`herdr-poke`, and app sounds are disabled for Poke's cries. These remain the current
personal defaults. Selecting another layout does not replace these files or remove
Poke's installed statusline, shell alias or launchd job.

For each new setup, explicitly decide whether to reuse or change these defaults.
Do not copy Poke's installer as though it were a generic prerequisite. Back up shared
settings before installation edits; retain the install/uninstall ownership journals.

## Development rules

- `AGENTS.override.md` and `CLAUDE.md` route agents here. The existing
  `herdr-poke-dev` skill applies to Poke work only.
- Herdr generates `AGENTS.md`, `herdr.d.ts` and `gpui-kit.d.ts` in the installed home.
  Search those for API details; never edit generated references. They may be absent
  from a source-only checkout.
- `.gitignore` excludes generated files, shipped resources and runtime data. Extend
  it when a new setup introduces runtime assets; do not ignore custom source files.
- Keep docs in `docs/modes/`, `docs/specs/` and `docs/plans/`, not `docs/superpowers/`.
  Do not add commit hashes to docs.
- Follow `~/.claude/commit-convention.md` for commits. Scope changes to the relevant
  setup or the repository. No AI signatures in commits or PRs.
- Never touch unrelated user terminals. Poke-specific visual and integration rules
  are in its guide; they are not requirements for every future setup.

## Path migration

The former `helper/`, `scripts/*` and `tests/{mod-lib,scripts}/` now live at
`helpers/poke/`, `scripts/poke/*` and `tests/poke/{mod-lib,scripts}/`. Use the named npm
commands or the new paths. No reinstall is needed solely for these source moves:
Poke's binary name, Go module, labels, asset paths, state paths and launchd identifier
are unchanged. The local checkout and GitHub repository are named `herdr`; `origin` points to
`git@github.com:keoneokeoneo/herdr.git`.
