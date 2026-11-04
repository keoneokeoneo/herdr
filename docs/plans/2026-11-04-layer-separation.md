# Layer separation (base / Poke / Obsidian) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make LLM usage, the sidebar/tab bar shell and Obsidian work without any Poke code, and shrink `mods/poke` to persona only.

**Architecture:** A neutral usage data contract (`~/.local/state/herdr-app/usage/{claude,codex,daily}.json`, one writer each) feeds a new neutral mod `base` (sidebar, tab bar, pane header, panel tabs, bottom bar, Usage tab). `mods/poke` keeps companion, Dex and a persona host. Layouts `base.json` and `poke.json` are two hand-maintained files guarded by a parity test. Poke is reached only through documented shared-state keys. Setups (`base`, `obsidian`, `poke`) are installed selectively by a manifest-driven installer (`scripts/setups.json` + `scripts/setup.py`); key bindings live in per-layout `layout:<name>` sections of `keys.json`, so an unselected setup costs nothing. Docs are kept current by a test that compares them with the manifest.

**Tech Stack:** Herdr JS mods (gpui-kit), Node test runner, Go (`helpers/poke`, plus Keno's `~/.claude/statusline`), Python 3 installers and their unittest suites.

**Spec:** `docs/specs/2026-11-04-layer-separation-design.md` (read its "Refinements found while planning" section; it supersedes earlier calls).

## Global Constraints

- Native resources stay in `layouts/`, `mods/`, `themes/`; runtime state never under `mods/` (writes there reload mods). Do not edit Herdr-shipped files or generated `AGENTS.md`/`*.d.ts`.
- Neutral state dir `~/.local/state/herdr-app/usage/`; files `claude.json`, `codex.json`, `daily.json`; env override `HERDR_USAGE_STATE`. Provider file shape = existing `ProviderLimits` (`{plan, windows:[{minutes, used, resets_at}], updated}`); `daily.json` = today's `usage.json` shape.
- Helper binary name `herdr-poke`, launchd identifier, shell alias, install journals keep their names (no reinstall).
- Keno's global statusline Herdr write is best-effort: after output is printed, bounded timeout, all errors ignored; it never delays or breaks the status line.
- `poke.*` labels and shared keys keep their names. Allowed `poke.*` references outside Poke code: shared `poke.sprites`, `poke.evening`, `poke.typeMode`, `poke.typeSeed`; labels `poke.partner`, `poke.review_of`, `poke.findings`, `poke.owner`, `poke.provider`, `poke.model`, `poke.effort`, `poke.ctx`.
- Claude agent rows lose context %, model and effort (Keno's decision); Codex rows keep them. Rows must render with missing labels.
- Layout ids lose the `poke-` prefix except persona-only blocks (`poke-companion`, `poke-dex`, `poke-host`). Keys: `cmd-j`→`base.jump`, `cmd-k`→`layout.toggle:panel`, `cmd-shift-o`→`base.notes`, declared in `keys.json` sections `layout:base` and `layout:poke` (Herdr applies `layout:<name>` while `layouts/<name>.json` shows), not in `global`.
- Selective install (Keno's request): setups are `base`, `obsidian`, `poke`; `poke` requires `base`; `obsidian` stands alone (its embedded panel tab works only where `base` is installed). Choosing a setup never installs another one's helper, launchd job, shell alias or app-wide settings. Only `poke` touches `~/.zshrc`, `programs.json` (Codex wrapper) and `settings.json` sounds.
- Docs must describe the final state (README setup table, structure, install commands, `docs/guide.md`, `docs/modes/*`, the `herdr-poke-dev` skill); a test fails when a manifest setup, layout, mod or theme is missing from the README or a relative doc link is broken.
- Visual output of `layouts/poke.json` must not change. Back up shared settings before edits. Never touch unrelated user terminals. No hash in docs, no AI signature in commits/PRs, no commit unless Keno asks (the commit steps below run only when authorized). Commit style follows the repo history: `type(scope): 한국어 요약` (`docs/guide.md` points to `~/.claude/commit-convention.md`, which does not exist; the file that does is `~/Documents/work/commit-convention.md` and applies only under `~/Documents/work`).
- Verify mods with `herdr-app check mods/<name>`; `npm test` stays green; each task leaves the repo working.

## Review Focus

- Neutral file missing, empty, corrupt JSON or half-written, or a window whose `resets_at` is past: views show "–"/"no current data"/"No usage data yet", never a stale percentage, never throw (Task 1).
- Claude session without `rate_limits` (API-key use) or only one window: the statusline must not blank or overwrite good data; an older snapshot from a second session never replaces a newer `claude.json` (Task 3).
- Base layout with no Poke running: no `poke.sprites`/labels present, rows and tabs still render letter avatars and no type pills (Task 4).
- Installing only `obsidian`, or only `base`, into an empty app home works and leaves `~/.zshrc`, `programs.json` and `settings.json` untouched; installing `poke` pulls in `base`; running an install twice changes nothing (Task 8).
- An existing different binding for `cmd-j`/`cmd-k`/`cmd-shift-o` in the user's `keys.json` is kept and reported, not overwritten (Task 8).

## File map

- Create `mods/base/`: `main.js`, `mod.json`, `lib/{contract,format,usage,model}.js`, `views/{brand,spaces,agents,tabs,pane,paneltabs,panel,status,ui,data,store}.js`, `behaviours.js` (jump, notes, Codex attach probe).
- Shrink `mods/poke/`: `main.js`, `mod.json`, `views/{companion,dex,host,ui}.js`, `lib/{companion,sync,partner,voice,cry}.js`, `behaviours.js` (persona only).
- Create `layouts/base.json`, `themes/base.json`; rewrite `layouts/poke.json`, `layouts/obsidian.json`; edit `keys.json`, `mods/obsidian/main.js`.
- Go: `helpers/poke/internal/{state,usage,codex}`; later delete `internal/claude` + `statusline` subcommand. Outside repo: `~/.claude/statusline/main.go`.
- Scripts: `scripts/poke/{settings_edit.py,install_files.py,uninstall.sh}`, `scripts/obsidian/install.sh` (thin wrapper), new `scripts/setups.json` (manifest) and `scripts/setup.py` (selective installer), tests in `tests/setup/test_setup.py`.
- Tests move to `tests/base/mod-lib/` (format, usage, telemetry, model, store, contract) and stay in `tests/poke/mod-lib/` (companion, sync, partner, voice, cry).
- Docs: `docs/modes/base.md` (new, contains the contract), `poke.md`, `poke-setup.md`, `obsidian.md`, `docs/guide.md`, `README.md`.

---

### Task 1: Contract reader in the app (JS)

**Files:**
- Create: `mods/base/lib/contract.js` (new folder; Task 4 moves the rest of the shell beside it)
- Modify: `mods/poke/views/store.js` (`loadUsage` uses the contract; Task 4 moves this file to `mods/base/views/store.js`)
- Test: `tests/base/mod-lib/contract.test.mjs`

**Interfaces:**
- Produces: `pickFresher(a, b): object|null` (larger numeric `updated` wins; missing/non-object loses); `resolveLimits(neutral: {claude?, codex?}, legacy: {claude?, codex?}): {claude: object|null, codex: object|null}`; `resolveDaily(neutral: object|null, legacy: object|null): object|null`; `USAGE_DIR(): string`.
- Consumes: existing `usageState`, `win`, `live`.

- [ ] **Step 1: Write failing tests** `pickFresher_prefers_newer_updated`, `pickFresher_ignores_non_objects_and_missing_updated`, `resolveLimits_uses_legacy_only_when_neutral_missing_or_older`, `resolveDaily_prefers_neutral_when_updated_not_older`, `corrupt_input_returns_null_not_throw` (feed `null`, `""`, `42`, `{}`).
- [ ] **Step 2: Run** `node --test tests/base/mod-lib/contract.test.mjs` → FAIL (module missing).
- [ ] **Step 3: Implement** the four exports in `mods/base/lib/contract.js` (pure functions, no `herdr` import). In `store.js`, `loadUsage(s)` reads `USAGE_DIR()/{claude,codex,daily}.json` and the legacy `~/.local/state/herdr-poke/{limits,usage}.json`, then sets `s.limits`/`s.usage` through the resolvers; every read wrapped so failure yields `null`. `keepFresh` also watches `USAGE_DIR()`.
- [ ] **Step 4: Run** `npm test` → PASS.
- [ ] **Step 5: Commit** `feat(herdr): usage 데이터 계약 읽기 추가`

### Task 2: Helper writes the neutral files (Go)

**Files:**
- Modify: `helpers/poke/internal/state/state.go`, `internal/usage/usage.go`, `internal/codex/wrap.go`, `helpers/poke/main.go`
- Test: `internal/state/state_test.go`, `internal/usage/usage_test.go`, `internal/codex/wrap_test.go`

**Interfaces:**
- Produces: `state.UsageDir() string` (`$HERDR_USAGE_STATE`, else `~/.local/state/herdr-app/usage`, created 0700); `state.WriteUsageJSON(name string, v any) error` (atomic temp+rename in `UsageDir()`); `state.SeedDaily() error` (copy `Dir()/usage.json` to `UsageDir()/daily.json` only when the latter is absent).
- Changes: provider limits are written as the object itself to `UsageDir()/<provider>.json` (no map), keeping `MergeLimits`/`UpdateLimitsLatest` semantics (state.go 147–215: skip older data, skip unchanged windows); `usage.Run` writes `daily.json`; the `usage` subcommand calls `SeedDaily` first.

- [ ] **Step 1: Write failing tests**: `TestUsageDirEnvOverride`, `TestUpdateLimitsLatestWritesProviderFile` (codex data lands in `codex.json`, not `limits.json`), `TestOlderSnapshotDoesNotReplaceNewer`, `TestSeedDailyCopiesOnceAndNeverOverwrites`, `TestUsageRunWritesDaily`.
- [ ] **Step 2: Run** `cd helpers/poke && go test ./...` → FAIL.
- [ ] **Step 3: Implement** as above; leave `Dir()`/label caches/locks and the old `statusline` code untouched in this task.
- [ ] **Step 4: Run** `cd helpers/poke && go test ./...` → PASS. Run `bash scripts/poke/build.sh` and confirm the binary builds.
- [ ] **Step 5: Commit** `refactor(herdr-poke): 한도와 사용량을 중립 경로에 기록`

### Task 3: Global statusline writes Claude limits (outside the repo)

**Files:**
- Modify: `~/.claude/statusline/main.go` (back it up to `main.go.bak-layer-separation` first; the directory is not part of this repo, so no commit here)
- Test: `~/.claude/statusline/main_test.go` (create)

**Interfaces:**
- Produces: `limitsFor(in input, now time.Time) (windows []window, ok bool)` — same rules as `limitsFrom` in `helpers/poke/internal/claude/statusline.go:404`: only reported windows (5h=300, 7d=10080 minutes) with `resets_at` in the future; `ok=false` when none. `writeLimits(dir string, windows []window, now time.Time, plan string) error` — atomic temp+rename to `<dir>/claude.json`, skipped when an existing file has a newer `updated`, or identical windows.
- Wired in `main()` after the line is written to stdout, with a 300 ms bound, recover and ignore every error; dir = `$HERDR_USAGE_STATE` else `~/.local/state/herdr-app/usage`.

- [ ] **Step 1: Write failing tests**: `TestLimitsForDropsPastAndUnreportedWindows`, `TestNoRateLimitsWritesNothing`, `TestWriteDoesNotOverwriteNewer`, `TestWriteIsAtomicJSONShape` (parses back to `ProviderLimits` shape), `TestOutputUnchangedWhenDirUnwritable` (render output identical, exit 0).
- [ ] **Step 2: Run** `cd ~/.claude/statusline && go test ./...` → FAIL.
- [ ] **Step 3: Implement** the two functions and the post-output call. The printed text must stay byte-identical (existing behaviour).
- [ ] **Step 4: Rebuild** with the command in the file header; run `go test ./...`; pipe a sample JSON with `rate_limits` into `./statusline` and confirm `~/.local/state/herdr-app/usage/claude.json` appears and the printed line is unchanged. Expected: file has 1–2 windows, `updated` ≈ now.
- [ ] **Step 5:** no commit (outside repo); keep the `.bak` and note the change in `docs/modes/base.md` (Task 10).

### Task 4: Create `mods/base` (move shell + usage views)

**Files:**
- Create: `mods/base/**` per the file map. Move with `git mv` from `mods/poke`: `views/{spaces,agents,tabs,pane,panel,status,paneltabs,brand,ui,data,store}.js`, `lib/{format,usage,telemetry,model}.js` (model without `cryFor|retryDue|assetsMissing`, which go to `mods/poke/lib/cry.js`). Add `mods/base/main.js` (the `Poke` view class renamed `Base`, view map without companion/dex) and `mod.json` `{ "place": "none", "title": "Base" }`.
- Modify: moved files so no import reaches another mod and no `poke-` id remains: panel id `panel`, stack `panel-body`, children `usage`/`changes`/`files`/`obsidian`.
- Move tests to `tests/base/mod-lib/`, repointing imports; split `tests/poke/mod-lib/cry.test.mjs` imports to `mods/poke/lib/cry.js`.
- Test: `tests/base/mod-lib/{format,usage,telemetry,model,store,contract}.test.mjs`, new `tests/base/mod-lib/avatar-fallback.test.mjs`.

**Interfaces:**
- `brand` view takes `props.logo?: string` (image path); with none it shows only the "Herdr" text. Exports `narrow()`/`toggleSidebar()` unchanged.
- `paneltabs` takes `props.tabs: [[blockId, label], ...]` and `props.stack` (default `"panel-body"`); no hard-coded list.
- `tabType(branch, repoType)` defaults `shared("poke.typeMode")` to `"off"` (no type pill when unset).
- `subline` in agents rows drops model/effort/ctx for provider `claude`; `data.js` `rows()` keeps reading labels but tolerates absence.
- `status` view `command(s, c)` handles `"jump"` (cycle `waiting(rows())`, `run("agent.focus", {terminal})`) and `"notes"` (`run("layout.show", {id:"panel"})`, `run("layout.select", {stack:"panel-body", child:"obsidian"})`).

- [ ] **Step 1: Write failing tests**: `avatar_without_sprites_or_labels_renders_letter` (stub `shared()` → `{}`, row with no `poke.*` labels: no throw, letter `C`/`X`), `tabType_off_when_unset`, `paneltabs_uses_props_tabs` (extract pure `tabsFrom(props)`), `subline_claude_hides_ctx_model_effort`, `subline_codex_keeps_them`.
- [ ] **Step 2: Run** `npm test` → FAIL on new tests.
- [ ] **Step 3: Do the move and edits.** Make `ui.js` avatar/tab code import nothing Poke-named. Keep the persona seam keys per Global Constraints.
- [ ] **Step 4: Run** `npm test` → PASS; `herdr-app check mods/base` → OK.
- [ ] **Step 5: Commit** `refactor(herdr): 사이드바와 usage 화면을 base mod로 분리`

### Task 5: Persona host and slim `mods/poke`

**Files:**
- Modify: `mods/poke/main.js` (views: `companion`, `dex`, `host`; props.view `host` starts persona behaviours and renders an empty `div()`), `mods/poke/behaviours.js` (keep: partner assignment, cries, PokeTokenBar sync, evening flag, companion diff, reroll; remove jump, notes, Codex attach, usage), `mods/poke/views/companion.js` (own `narrow()` copy; own `ui.js` with `pill`, `spritePath`, `alive`, `C`)
- Create: `mods/poke/views/{dex,host,ui}.js`, `mods/poke/lib/cry.js`; move Dex rendering out of the old `panel.js`
- Modify: `mods/base/behaviours.js` — Codex attach probe moves here, started by the `status` view's copy only.
- Test: `tests/poke/mod-lib/*` (imports repointed), new `tests/poke/mod-lib/persona-defaults.test.mjs`

**Interfaces:**
- Produces: persona host sets `shared("poke.typeMode")` to `"random"` when unset on start (keeps today's Poke look); everything it publishes keeps its current key names.
- Consumes: nothing from `mods/base` by import.

- [ ] **Step 1: Write failing test** `typeMode_defaults_random_only_when_unset` and `persona_behaviours_do_not_reference_jump_or_codex_attach` (source scan of `mods/poke/behaviours.js`).
- [ ] **Step 2: Run** `npm test` → FAIL.
- [ ] **Step 3: Implement**; `mods/poke` must contain no usage/limits reading and no `layout`/`panel` ids.
- [ ] **Step 4: Run** `npm test` → PASS; `herdr-app check mods/poke` and `mods/base` → OK.
- [ ] **Step 5: Commit** `refactor(herdr-poke): poke mod를 컴패니언과 Dex 중심으로 축소`

### Task 6: Layouts, theme, keys and the guard tests

**Files:**
- Create: `layouts/base.json`, `themes/base.json` (copy of `themes/poke.json` colors, no persona names)
- Modify: `layouts/poke.json` (mod `base` for shared blocks, ids per Global Constraints, `poke-host` block `size: {px: 1}`, paneltabs `props.tabs` includes `["poke-dex","Poke"]`, brand `props.logo: "mods/poke/brand/pokeball.svg"`), `layouts/obsidian.json` (`"theme": "base"`), `keys.json` (remove `cmd-j`, `cmd-k`, `cmd-shift-o` from `global`; add sections `layout:base` and `layout:poke` each with `cmd-j: "base.jump"`, `cmd-k: "layout.toggle:panel"`, `cmd-shift-o: "base.notes"`)
- Test: `tests/base/mod-lib/layout-parity.test.mjs`, `tests/base/mod-lib/no-poke-coupling.test.mjs`

- [ ] **Step 1: Write failing tests**: `layouts_share_non_persona_block_ids` (ids of `base.json` ⊆ ids of `poke.json`, difference is exactly `poke-companion`, `poke-dex`, `poke-host`), `base_layout_has_no_poke_mod_or_id`, `non_poke_files_only_use_allowed_poke_keys` (scan `mods/base`, `mods/obsidian`, `layouts/base.json`, `layouts/obsidian.json`, `themes/base.json`, `keys.json` for `poke` and compare to the allowed list in Global Constraints), `keys_point_to_existing_commands_and_ids`, `global_keys_have_no_setup_bindings` (no `base.*`/`poke.*` action and no `layout.toggle:panel` in `global`).
- [ ] **Step 2: Run** `npm test` → FAIL.
- [ ] **Step 3: Write the layouts.** `base.json` = Poke layout minus persona blocks; sidebar stack stays `brand/spaces/agents/footer`; bottom bar is block `status` (`mod: base`, `view: status`), panel is `panel`. Keep sizes and ordering identical to the current Poke layout.
- [ ] **Step 4: Run** `npm test` → PASS. Validate JSON with `node -e` parse for each layout.
- [ ] **Step 5: Commit** `refactor(herdr): base 레이아웃 추가와 poke 레이아웃 id 중립화`

### Task 7: Decouple Obsidian

**Files:**
- Modify: `mods/obsidian/main.js` (line 105 hides `panel`; line 146 "Back to Poke" becomes "Back" → `layout.switch {name:"base"}`), `scripts/obsidian/install.sh` (becomes a one-line wrapper around `python3 scripts/setup.py install obsidian` once Task 8 exists; until then only drop the `--poke` patching of `layouts/poke.json` and `mods/poke/views/*`, which now ship natively, and copy `themes/base.json` instead of `themes/poke.json`), `docs/modes/obsidian.md`
- Test: `tests/obsidian/mod-lib/no-poke.test.mjs` (new)

- [ ] **Step 1: Write failing test** `obsidian_mod_has_no_poke_reference` (source scan of `mods/obsidian/*.js`).
- [ ] **Step 2: Run** `npm test` → FAIL.
- [ ] **Step 3: Implement**; keep the embedded close button and standalone behaviour otherwise unchanged.
- [ ] **Step 4: Run** `npm test` and `PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tests/obsidian/scripts` → PASS; `herdr-app check mods/obsidian` → OK.
- [ ] **Step 5: Commit** `refactor(herdr): Obsidian mod의 Poke 의존 제거`

### Task 8: Selective setup installer

**Files:**
- Create: `scripts/setups.json`, `scripts/setup.py`, `tests/setup/test_setup.py`
- Modify: `package.json` (`setup:list`, `setup:install`, `setup:status`, `test:setup` = `PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tests/setup`), `scripts/obsidian/install.sh` (wrapper), `scripts/poke/install.sh` (calls `setup.py install poke` for resource files and key sections, then keeps its own helper/zshrc/launchd/sounds steps)

**Interfaces:**
- Manifest `scripts/setups.json`: `{ "<name>": { "title": str, "requires": [name], "mods": [dir], "layouts": [file], "themes": [file], "keys_layouts": [layout name], "installer": null | "scripts/poke/install.sh" } }` for `base` (mods `base`, layout `base`, theme `base`, keys section `layout:base`), `obsidian` (mod `obsidian`, layout `obsidian`, theme `base`), `poke` (requires `base`; mod `poke`, layout `poke`, theme `poke`, keys `layout:poke`, installer `scripts/poke/install.sh`).
- `setup.py` CLI: `list` (names, titles, requires, installed or not), `install <name>...` (resolve `requires` first, ordered, no duplicates; `--select <layout>` runs `herdr-app do layout.switch`), `status`. Honors `HERDR_APP_HOME`; `HERDR_SETUP_DRY=1` skips `herdr-app` calls. Functions: `resolve(names, manifest) -> list[str]` (raises on unknown name or cycle), `install_files(setup, source, app) -> list[str]` (copies only when `source != app`, backing up a differing target as `<file>.setup-backup-<stamp>`), `merge_keys(source_keys, app_keys, sections) -> (merged, kept)` (adds only missing `layout:<name>` entries; a different existing binding is kept and listed in `kept`).
- Journal `~/.local/state/herdr-app/setups.json` records installed setup names and copied-file fingerprints; re-running changes nothing.

- [ ] **Step 1: Write failing tests** (unittest, temp `HERDR_APP_HOME`/`HOME`): `test_list_names_all_setups`, `test_resolve_adds_requires_in_order`, `test_resolve_rejects_unknown_and_cycles`, `test_install_obsidian_only_copies_obsidian_and_base_theme`, `test_install_obsidian_leaves_zshrc_programs_settings_untouched`, `test_install_poke_pulls_in_base`, `test_second_install_is_noop`, `test_existing_different_binding_is_kept_and_reported`, `test_in_place_checkout_copies_nothing_but_merges_keys`, `test_every_manifest_path_exists_in_repo`.
- [ ] **Step 2: Run** `npm run test:setup` → FAIL.
- [ ] **Step 3: Implement** `setup.py`; after copying, run `herdr-app check <app>/mods/<mod>` for each installed mod unless dry. The `poke` installer step is invoked only when `poke` is selected and is skipped under dry-run.
- [ ] **Step 4: Run** `npm run test:setup` → PASS; `python3 scripts/setup.py list` prints the three setups.
- [ ] **Step 5: Commit** `feat(herdr): 매니페스트 기반 선택 설치 도구 추가`

### Task 9: Installers stop touching global Claude settings

**Files:**
- Modify: `scripts/poke/settings_edit.py` (remove the `statusLine` install and restore paths, lines ~77–126; `sounds` handling stays; uninstall deletes a leftover `claude_statusLine` journal key without restoring it), `docs/modes/poke-setup.md` (resource copying is now `scripts/setup.py`; `install_files.py` keeps journaling only the helper binary and shims)
- Test: `tests/poke/scripts/test_settings.py`

- [ ] **Step 1: Write failing tests**: `test_install_leaves_claude_statusline_untouched`, `test_uninstall_does_not_restore_or_remove_statusline`, `test_legacy_journal_key_is_dropped`.
- [ ] **Step 2: Run** `npm run test:poke:scripts` → FAIL.
- [ ] **Step 3: Implement**; backups of shared files still made before any edit.
- [ ] **Step 4: Run** `npm run test:poke:scripts` → PASS. Add `test:base` (`node --test tests/base/mod-lib/*.test.mjs`) and keep `npm test` covering all.
- [ ] **Step 5: Commit** `fix(herdr-poke): 설치 스크립트가 글로벌 statusLine을 건드리지 않도록 수정`

### Task 10: Cutover cleanup and docs

**Files:**
- Modify: `mods/base/views/store.js` and `lib/contract.js` (remove the legacy fallback and `resolve*` legacy arguments; tests updated), `helpers/poke/main.go` (remove `statusline` subcommand and usage text)
- Delete: `helpers/poke/internal/claude/` (including golden `testdata`), `Dir()`-based `limits.json`/`usage.json` writers if any remain
- Create: `docs/modes/base.md` (contract, mod/layout/id/command names, statusline change description and the `.bak` note, verification), `tests/base/mod-lib/docs-current.test.mjs`
- Modify (rewrite to the final state, not patch): `README.md` (structure incl. `mods/base`, `scripts/setups.json`; setup table with Base/Obsidian/Poke, their requirements and what each install touches; "Choose what to install" with `npm run setup:list` / `setup:install -- base obsidian`; test commands), `docs/guide.md` (resource table, "Adding a setup" now includes a manifest entry, shared-settings section: keys are layout-scoped, Poke no longer edits `statusLine`, path migration note refreshed), `docs/modes/poke.md`, `docs/modes/poke-setup.md` (remove the statusLine bullet; install via `setup.py`/`install.sh`), `docs/modes/obsidian.md`, `.agents/skills/herdr-poke-dev/SKILL.md` (and its `.claude` link) wherever it names moved files, ids or commands

- [ ] **Step 1: Write failing test** `docs_current.test.mjs`: `readme_lists_every_manifest_setup_layout_mod_and_theme`, `relative_markdown_links_resolve` (README, `docs/**/*.md`), `docs_do_not_mention_removed_names` (`poke-panel`, `poke.jump`, `poke.notes`, `herdr-poke statusline`, `docs/superpowers`). Run `npm test` → FAIL.
- [ ] **Step 2: Update tests** so legacy fallback expectations become `legacy_files_are_ignored`; delete tests for the removed package.
- [ ] **Step 3: Run** `npm test` and `cd helpers/poke && go test ./...` → PASS after the removals; `grep -rn "herdr-poke statusline\|limits.json" --exclude-dir=.git .` shows only intentional history mentions, none in code.
- [ ] **Step 4: Write docs** until `docs_current.test.mjs` passes. No commit hashes. State plainly that Poke removal tooling is out of scope and that dated specs/plans are history and keep their original wording.
- [ ] **Step 5: Commit** `refactor(herdr): 레거시 fallback과 poke statusline 제거, 문서 최신화`

### Task 11: Live verification (with Keno's running Herdr)

- [ ] **Step 1:** Back up shared files (`keys.json`, `programs.json`, `settings.json`, `~/.claude/settings.json`) to a timestamped folder; install via `python3 scripts/setup.py install base obsidian` first, then `poke` (`docs/modes/poke-setup.md`), then `herdr-app check mods/base mods/poke mods/obsidian`. Confirm `base`+`obsidian` alone left `~/.zshrc`, `programs.json` and `settings.json` unchanged. Do not touch other terminals.
- [ ] **Step 2:** `herdr-app status`, then switch to `base` (`herdr-app do layout.switch '{"name":"base"}'`): bottom bar and Usage tab show both providers; trigger a Claude prompt and confirm `claude.json` `updated` advances and the Claude figures move; confirm `codex.json` and `daily.json` update.
- [ ] **Step 3:** Switch to `poke`: compare against the pre-change look (companion, Dex tab, brand, cries on a blocked agent, `cmd-j`, `cmd-k`, `cmd-shift-o`). Switch to `obsidian` and use Back.
- [ ] **Step 4:** Stop the `poke` layout path (use `base`) and confirm sidebar, tabs, usage and notes still work with no Poke host running. Report anything that differs; do not claim success without these checks.
