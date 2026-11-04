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

## Quick start (for other people)

You need Herdr installed and started once, so that `~/.config/herdr-app` exists, plus
Python 3 and Node.js for the installer and tests. Nothing here replaces your own files silently:
the installer backs up whatever it would overwrite and never changes a key binding you already have.

```sh
git clone git@github.com:keoneokeoneo/herdr.git && cd herdr
npm run setup:list                        # what exists, what it requires
npm run setup:install -- base --select base
```

That gives you the neutral shell (sidebar, tab bar, usage bar, right panel). Add more only if you want it:

| I want | Run | Needs |
|---|---|---|
| A vault reader | `npm run setup:install -- obsidian` | the Obsidian app and a vault |
| Pokémon partners | `npm run setup:install -- poke` | macOS, Go; it also edits `~/.zshrc`, `programs.json` and `settings.json` (read the [installation guide](docs/modes/poke-setup.md) first) |

The usage bar shows nothing until something writes the usage files. The UI only reads them, so you can
feed it from any tool. Each file is one JSON document in `~/.local/state/herdr-app/usage/`:

```json
{ "plan": "pro", "windows": [
    { "minutes": 300,   "used": 13, "resets_at": 1793791800 },
    { "minutes": 10080, "used": 27, "resets_at": 1794121200 } ],
  "updated": 1793775748 }
```

Name it `claude.json` or `codex.json` (`used` is a percentage, `resets_at` and `updated` are epoch seconds;
`300` is the 5-hour window, `10080` the weekly one). A window whose reset time has passed is shown as "–".
`daily.json` holds per-day token totals; the exact rules are in the [Base guide](docs/modes/base.md).
Without Poke's helper, Codex limits and daily totals stay empty unless you write those files yourself.

To remove a setup, delete its mod, layout and theme files from `~/.config/herdr-app` and switch layout
(`Cmd-Shift-L`); `setup:status` shows what the installer recorded.

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
