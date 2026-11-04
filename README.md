# Herdr

My Herdr workspace: custom layouts, mods, themes, and the tools that support them.
It is split into independent **setups** you can install separately: a neutral **base**
(sidebar, tab bar, usage bar, right panel), an **obsidian** vault reader, and **poke**
(Pokémon partners layered on top of base).

## Structure

```text
layouts/<name>.json       Window layouts loaded by Herdr
mods/<name>/              UI mods loaded by Herdr
themes/<name>.json        Themes loaded by Herdr
helpers/<name>/           Optional helper programs
scripts/setups.json       Manifest: which resources each setup needs
scripts/setup.py          Selective installer (list / install / status)
scripts/<name>/           Setup-specific build, install and uninstall tools
tests/<name>/             Setup-specific tests
docs/modes/<name>.md      Setup-specific guide
keys.json                 Shared keyboard bindings (setup keys live in `layout:<name>` sections)
programs.json             Shared terminal launch commands
settings.json             Shared app settings
```

Layouts, mods and themes are separate Herdr resources. A setup can combine several
mods or reuse an existing theme; it does not need one of every resource.

## Setups

| Setup | Layout | Mod | Theme | Requires | Guide |
|---|---|---|---|---|---|
| **base** | `layouts/base.json` | `mods/base/` | `themes/base.json` | nothing | [Base](docs/modes/base.md) |
| **obsidian** | `layouts/obsidian.json` | `mods/obsidian/` | `themes/base.json` | nothing (its panel tab needs base) | [Obsidian](docs/modes/obsidian.md) |
| **poke** | `layouts/poke.json` | `mods/poke/` | `themes/poke.json` | base | [Development](docs/modes/poke.md) · [Installation](docs/modes/poke-setup.md) |

What each install touches:

- **base**: its mod, layout and theme files, plus the `layout:base` key bindings
  (`cmd-j` jump to a waiting agent, `cmd-k` show/hide the right panel, `cmd-shift-o` notes).
  Claude's limits come from the global Claude statusline; Codex limits and daily usage come
  from the `herdr-poke` helper if it is installed. See the Base guide.
- **obsidian**: its mod, layout and the shared base theme. Nothing else.
- **poke**: base, plus the persona mod and layout, the `herdr-poke` helper, a launchd usage
  job, the `codex` shell alias, a managed PATH line and Herdr's sounds setting. Read its
  installation guide first; it is the only setup that edits shell and app settings.

## Choose what to install

```sh
npm run setup:list                               # setups, what they require, which are installed
npm run setup:install -- base obsidian           # install only these (and what they require)
npm run setup:install -- base --select base      # install, then switch Herdr to that layout
npm run setup:status                             # installed setups and whether their files changed
```

`setup:install` copies only the selected resources into `~/.config/herdr-app` (nothing is
copied when this checkout already is that folder), adds only missing key bindings (a
different binding you already have is kept and reported), backs up any file it would
replace, and records what it did. Installing `poke` also installs `base`; its own installer
runs last. `bash scripts/poke/install.sh` is the same as installing `poke`.

## Development

Read [the repository guide](docs/guide.md) for conventions and how to add a setup.
The repo keeps Herdr's native directory structure so it can live at
`~/.config/herdr-app`. A checkout elsewhere is a source checkout, not an installed
Herdr configuration: install the setups you want with the commands above.

Tests:

```sh
npm test                        # JavaScript tests for every setup, including the docs check
npm run test:setup              # selective installer
npm run test:poke:helper        # Go helper
npm run test:poke:scripts       # Poke installer scripts
npm run preflight:poke          # read-only installation check for Poke
```

`herdr-app check mods/<name>` compiles a mod without reloading it.

The helper binary, its state directory and launchd job keep their `herdr-poke` names. The
usage data that the UI reads is in `~/.local/state/herdr-app/usage/`. The local checkout
and GitHub repository are named `herdr`. The `origin` remote is
`git@github.com:keoneokeoneo/herdr.git`.
