# Herdr

My Herdr workspace: custom layouts, mods, themes, and the tools that support them.
`poke` is the first setup; additional setups live alongside it.

## Structure

```text
layouts/<name>.json       Window layouts loaded by Herdr
mods/<name>/             UI mods loaded by Herdr
themes/<name>.json      Themes loaded by Herdr
helpers/<name>/           Optional helper programs
scripts/<name>/           Setup-specific build, install and uninstall tools
tests/<name>/             Setup-specific tests
docs/modes/<name>.md      Setup-specific development guide
keys.json                Shared keyboard bindings
programs.json            Shared terminal launch commands
settings.json            Shared app settings
```

Layouts, mods and themes are separate Herdr resources. A setup can combine several
mods or reuse an existing theme; it does not need one of every resource.

## Setups

| Setup | Layout | Mod | Theme | Documentation |
|---|---|---|---|---|
| Obsidian | `layouts/obsidian.json` | `mods/obsidian/` | Reuses Poke | [Usage and installation](docs/modes/obsidian.md) |
| Poke | `layouts/poke.json` | `mods/poke/` | `themes/poke.json` | [Development](docs/modes/poke.md) · [Installation](docs/modes/poke-setup.md) |

## Development

Read [the repository guide](docs/guide.md) for conventions and how to add a setup.
The repo keeps Herdr's native directory structure so it can live at
`~/.config/herdr-app`. A checkout elsewhere is a source checkout, not an installed
Herdr configuration. Follow the setup's installation instructions before using it.

`npm test` runs JavaScript tests across setups. Poke also provides:

```sh
npm run test:poke:helper
npm run test:poke:scripts
npm run preflight:poke
```

`npm run install:poke` installs Poke integrations and selects its layout. Installing
Poke is optional for other setups; it edits shared app and shell settings, so read
its installation guide first.

The helper binary, state directory and launchd job retain their `herdr-poke` names.
The local checkout and GitHub repository are named `herdr`. The `origin` remote is
`git@github.com:keoneokeoneo/herdr.git`.
