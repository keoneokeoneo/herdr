# Obsidian reader

A read-only native Herdr mod for local Markdown vaults. Requires macOS, Python 3
and Herdr with `TextView.markdown`. Obsidian does not need to be running.

## Poke integration (recommended)

```sh
bash scripts/obsidian/install.sh --poke
herdr-app do layout.switch '{"name":"poke"}'
```

The right side is one panel with peer tabs: **Usage / Changes / Files / Poke /
Obsidian**. Only one tab's contents are visible. The top-right panel button or
**Cmd-K** hides/shows the whole panel; **Cmd-Shift-O** opens the Obsidian tab.
The reader's **×** closes the whole panel. Tab switching and closing retain the
current note, search and expanded folders within the same layout.

The reader shows the vault name, a collapsible folder tree and note titles without
full paths. Expand folders to browse; searching expands matching paths. An empty
search restores the entire indexed tree, rather than the first 200 notes.
**Open in Obsidian** opens the current note in the desktop application using its
[official URI](https://obsidian.md/help/uri). It does not launch until clicked.

The optional `--poke` install updates the Poke layout, tab bar, panel tabs and command host, and merges only
the new shortcut into installed keys. An existing conflicting shortcut is kept.
Backups live under `data/obsidian/backups/<timestamp>/`. This expects Poke already
installed; it does not install Poke's helper or shell integrations.

## Standalone layout

```sh
bash scripts/obsidian/install.sh
herdr-app do layout.switch '{"name":"obsidian"}'
```

The layout places the active terminal tab beside the reader; drag the divider to
resize. Use **Vault** to choose a vault discovered from Obsidian's macOS
registry, or enter an absolute folder path (also accepts `~/...`) and choose
**Use folder**. This supports a future personal-vault without source changes.
Vaults added manually and the last selected vault persist in Herdr shared state
under `obsidian.vaults` and `obsidian.root`. The registry and vault are never written.

Enter a title, relative path or body search and press Enter or **Search**. Empty
search browses the folder tree. Click a note to read Markdown, follow wiki links or relative
Markdown links, use **←** to return, and **Refresh** to reload a changed note.
**Search** also refreshes the file list. HTTP(S) links open in the default browser.
Use **Back to Poke** to return to Poke; existing Cmd-Shift-L cycles layouts.
The mod accepts `obsidian.search`, `obsidian.vault:<folder>` and
`obsidian.note:<relative path>` commands through `herdr-app do` for keyboard or
CLI integration. `pane.go` focuses its search input.
The mod can also be used in another layout as `{"mod":"obsidian"}`.

## Limits

This is a Markdown reader, not the Obsidian application. Plugins, Dataview,
Canvas, graph view, attachment embeds and editing are not implemented. Wiki
embeds are displayed as navigation links; heading/block links open the note
without scrolling to the anchor. Duplicate short link names require a more
specific path. Hidden directories and symlinks are omitted during indexing.
Reads are restricted to Markdown files inside the selected vault. Files over
1 MiB are rejected; the tree indexes at most 20,000 notes; nonempty searches display at most 200
results (filename matches first). Search runs on submit, not every keystroke.

The installer copies only this mod/layout and the Poke theme, backing up changed
existing files beside them. It does not alter keys, shell settings, launchd,
Obsidian settings or the current layout. Runtime state lives in Herdr's shared
storage. To remove, switch to another layout, delete `mods/obsidian` and
`layouts/obsidian.json` from the app home; retain the shared Poke theme.

## Verification

```sh
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tests/obsidian/scripts
herdr-app check mods/obsidian
herdr-app status
```

Also inspect the running UI: vault selection, search, note rendering, internal
links, back navigation and a failed folder path. Mod checks only render the
initial state and do not exercise asynchronous reading.
