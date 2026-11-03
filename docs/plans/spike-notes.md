# herdr-poke v1: spike notes

Run 2026-11-03 on herdr-app 0.1.12 (source 0.1.12), herdr-core `hc`, Codex CLI 0.166.0, in throwaway terminals started with `hc run --bg`. All throwaway terminals were killed and removed afterwards.

The Herdr desktop window was not running during the spikes (`herdr-app status`: connection refused, stale socket). Steps 2 and 4 are therefore answered from the app source with file:line and from shipped layouts, not from a screenshot. `src/...` paths below are in the herdr-app 0.1.12 source clone (not part of this repo).

## Step 1: Codex pid to rollout (lsof)

- **Result:** works with `--no-daemon`; fails in default (daemon) mode.
- **Evidence:**
  - Single run: `lsof -a -p <pid> -Fn | grep rollout-` printed exactly 1 path, and its `session_meta.cwd` matched the run's cwd.
  - Two runs in one folder started 1 s apart: pids 84476 and 84612 mapped to 2 different rollouts (`...T15-38-03-...` and `...T15-38-04-...`), 1 path each, stable over 5 samples.
  - Without `--no-daemon` (pid 85659): the TUI pid holds 0 rollouts. The shared daemon (`codex app-server --listen unix://`) holds them all, so there is no pid mapping.
  - Timing: the rollout file is created lazily on the first prompt, not at startup. Before the first prompt lsof shows nothing at +3 s and +6 s. The filename timestamp is the session start (about 10 s earlier than the first event), which is why same-folder runs order correctly by start time.
  - `codex resume <id> --no-daemon`: lsof shows the original rollout immediately, before any prompt, and no new file is created.
- **Fallback chosen:** keep lsof as the primary (certain=true), always launched with `--no-daemon` (the spec's alias already does this). Keep the new-file fallback (matching cwd, no `forked_from_id`, start order) for sessions started without `--no-daemon`, and for the window before the first prompt. `codex resume` is covered by lsof, so the `state_5.sqlite` lookup is not needed. A resumed session adds no new file, so the new-file fallback can never find it.

## Step 2: terminal display name and the "blocked" toast

- **Result:** a terminal's `name` cannot be changed after it is created. The toast says `<name> needs you`.
- **Evidence:**
  - `hc schema` lists 32 methods and none renames a terminal (`new_name` exists only on `machine.rename`).
  - `hc label spk-cx1 name=renamed` only added `labels.name`; `name` stayed `spk-cx1`.
  - `hc run --name spk-cx1` again returned `name_taken`.
  - In `herdr-app commands`, only `space.rename` and `tab.rename` exist. `terminal.open` accepts `name` at creation only.
  - Blocked state: `printf '\033]7501;state=blocked:app=test:kind=permission\033\\'` set `hc get` to `status {state: blocked, app: test, kind: permission}`. Clear it with `state=idle`.
  - Toast text from the source (`alert()`, `src/window.rs:2023-2057`): title `"{name} needs you"` (level `error`), body `"<space> / <tab>"`, click runs `agent.focus`.
  - `{name}` is the core terminal `name` if non-empty, else `agent_name(app)` ("Codex", "Claude Code"; `src/window.rs:4373`), else "An agent".
  - Toasts only fire for changes after the first listing, and not while the person is looking at that agent.
  - Not confirmed on screen (window not running).
- **Fallback chosen:** keep Herdr's toast wording. The partner name shows in our own UI only. Terminals started from the UI have an empty `name`, so the toast reads "Codex needs you".

## Step 3: `hc label` latency

- **Result:** p50 9.6 ms, max 13.4 ms over 10 runs. Under the 30 ms limit.
- **Evidence:** run inside an hc terminal (`HC_TERMINAL_ID=t7`), `hc label $HC_TERMINAL_ID poke.test=1`, timed with `perf_counter` around `subprocess.run`.
  - 10 runs: 13.4, 10.0, 9.3, 11.6, 9.3, 10.6, 9.0, 9.2, 10.8, 8.9 ms.
  - 30 more runs: p50 9.1, max 12.7 ms. 10 runs with a changing value: p50 9.9, max 11.0 ms.
  - Process-start floor: `hc --version` p50 4.1 ms, `hc get` p50 3.5 ms.
  - Measured with no Herdr window attached.
- **Fallback chosen:** none needed. The statusline writes labels inline, only when a value changed, with a short timeout. No detached child for labels.

## Step 4: per-copy `props` on one mod

- **Result:** supported. Each `layout` block gets its own `props()`. `herdr-app check` cannot show this.
- **Evidence:**
  - Throwaway mod `x` (outside the repo) reading `props()?.view`: `herdr-app check` passes and renders `view=(none)`. The checker loads a mod with a null-props default instance (`src/mods.rs:1139`), so it cannot show two copies.
  - Layout block content is `Content::Mod { name, props }` (`src/layout.rs:14-19`).
  - Each visible block becomes `Instance { name, props }` keyed by block id (`src/window.rs:1150`).
  - Each copy is loaded via `prepare_load(&copy.inst)` (`src/window.rs:1591`) with its own host module. `props()` returns that copy's props through a closure (`src/mods.rs:1074`).
  - Copies with different props are separate (`Instance::key()` includes props, `src/mods.rs:715-718`) and get separate localStorage files (`src/mods.rs:1159-1165`).
  - Shipped layouts already do it. `layouts/win98.json` has four blocks of mod `win98` with `role` = desktop / window / clippy / taskbar (`mods/win98/main.js:27-34` reads `props()?.role`). `layouts/main.json` uses `workspace-chrome` both with no props and with `{"footer": true}`.
  - Not proven live in a window (window not running).
- **Fallback chosen:** none. Use one `poke` mod, with each layout block giving `{"mod":"poke","props":{"view":"..."}}`. Block ids must be unique. Changing a block's props restarts that copy.
