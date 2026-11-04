# Customizing Herdr from this repo

How to change the Herdr window with this repository, for people and for coding agents. Read
[the repository guide](guide.md) first for the file map; this page is the Herdr-specific part:
how the pieces work, the traps, and how to check a change.

## How Herdr uses these files

- **Layout** (`layouts/<name>.json`) arranges views into one whole window. `layout.switch {name}`
  picks one, `shared("layout")` is the shown name. Shipped layouts (`main`, `terminals`, `win98`) are
  restored on start; clone one rather than editing it.
- **Mod** (`mods/<name>/`) implements views and commands. `main.js` default-exports a View class and
  the layout chooses a view with `props.view`. Mods are separate folders and cannot import each other:
  share through `shared(...)`/`setShared(...)`, terminal labels, commands (`<mod>.<command>`) or a data file.
- **Theme** (`themes/<name>.json`) holds colours. **Keys** (`keys.json`) has `global`, `terminal` and
  `layout:<name>` sections; setups keep their own bindings in `layout:<name>`.
- Herdr's own API reference is generated into the installed home as `AGENTS.md`, `herdr.d.ts` and
  `gpui-kit.d.ts`. Search them for a function before guessing; never edit them (they are git-ignored here).

## Traps

- **Hot reload:** Herdr reloads a mod whenever a non-hidden file inside `mods/<name>/` is written. Runtime
  data goes in a state folder (`~/.local/state/...`) or an assets folder, never inside `mods/`. Writing there
  loops reloads until QuickJS runs out of memory.
- **Shipped mods are restored on every start.** An edited one moves to `<name>-edited`. Clone a shipped
  mod instead of editing it, and reuse shipped behaviour through commands: `openOverlay({mod:"menu"})`,
  `mod:"new-menu"`, `layout.toggle`, `layout.resize`, `terminal.pin`, `app.palette`.
- **`setShared` re-renders every view.** Call it only when the value really changed. `cx.notify()` rebuilds
  its owning view and redraws the window. Hidden views keep their timers after first display.
- **Colours:** `"transparent"` is not a colour; use `"#00000000"`.
- **macOS traffic lights:** `traffic_lights: [12, 10]` centres them on the 46px brand row. They reach x≈80,
  so the brand row pads 96px and the folded sidebar is 92px wide. Below 120px every sidebar view must switch
  to its rail form (`narrow()`), so test both widths when you touch a sidebar view.
- **Start Herdr from the Dock,** never from a Claude shell: a server started there inherits
  `CLAUDE_CODE_CHILD_SESSION=1` and every Claude in it stops saving transcripts.
- **The `hc` server has a minimal PATH** (no nvm, no zsh aliases), so programs started by `hc run` need
  absolute paths or the `~/.local/bin` shims.
- **Never touch the user's other terminals.**

## Checking a change

1. `herdr-app check mods/<name>` prints `check passed` (mods only; layouts are checked by switching to them).
2. `herdr-app status` shows `"errors": []` and the expected `layout`.
3. `npm test` (mod logic, layouts, keys, docs). Installer: `npm run test:setup`. Write the failing test first.
4. Look at it: `herdr-app shot out.png` saves the last frame without the traffic lights. For the lights, find
   the window id and use `screencapture -x -o -l <id> out.png`. Captures are 2×; crop before looking.
5. Sidebar rail: `herdr-app do layout.resize '{"id":"sidebar","px":92}'` folds, `262` restores. Put it back.
6. Switch to each affected layout with `herdr-app do layout.switch '{"name":"<name>"}'` and put the user's layout back.

## Adding or changing a setup

Follow "Adding a setup" in [the repository guide](guide.md), then the setup's own page:
[Base](modes/base.md), [Obsidian](modes/obsidian.md), [Poke](modes/poke.md). Setup-specific rules
(Poke's GIFs, Codex mirrors, sprites) live there, not here.
