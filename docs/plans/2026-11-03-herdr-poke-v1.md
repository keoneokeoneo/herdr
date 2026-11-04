# herdr-poke v1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn Keno's Herdr 2 window into the prototype-v6 workspace (Pokémon avatars, always-visible Claude/Codex limits, pokeTokenBar companion), fed by a Go helper.

**Architecture:** A Go binary `herdr-poke` (statusline / codex wrapper / usage / assets) writes per-session facts as herdr-core terminal labels (`poke.*`) and account facts as JSON in `~/.local/state/herdr-poke/`. One Herdr mod `poke` renders every region, chosen by `props.view`; its logic lives in dependency-free modules under `mods/poke/lib/` so Node can unit-test them. A new layout `layouts/poke.json` places the views.

**Tech Stack:** Go 1.27 (stdlib only), Herdr app 0.1.12 mods (QuickJS + gpui-kit), Node 22 `node --test` for mod logic, `herdr-app check` / `herdr-app shot`.

**Spec:** `docs/specs/2026-11-03-herdr-poke-design.md` (readable: https://claude.ai/artifact/6FnMqh5cpHSRHKiWziVrmb). Visual reference: `docs/prototype/herdr-poke-template.html` (published https://claude.ai/artifact/6mQsSo45wkmNmEQGS2WjEZ).

## Global Constraints

- Repo root is `~/.config/herdr-app/`. Never edit shipped files (`AGENTS.md`, `*.d.ts`, shipped `mods/*`, `layouts/main.json`…); they're gitignored and restored by Herdr.
- pokeTokenBar files are read-only. Never open `usage-cache.json`.
- Statuses are exactly `working | blocked | done | idle`. No error state anywhere.
- No timer-driven animation in mods; timers only for data, ≥ 2 s apart. Animate with GIFs, `.loop`, `.transition`.
- Keys: `cmd-j` jump to next blocked (oldest first, cycles), `cmd-k` toggle right panel. Nothing else bound.
- Partners: species 1–1025, never in Keno's dex (`dex[].chainOrder`) nor the active line (`active.plannedPathIDs`); stable per terminal; no evolution. Sprite ≤ 649 = PokeAPI BW animated GIF, else PokeAPI static PNG.
- Chips (plain words only): `ctx N%` when ctx ≥ 80; `waiting Nm` when blocked ≥ 5 min.
- Palette tokens: bg `#1b1d26`, surface `#20232c`, raised `#272b36`, line `#313545`, text `#e6e8ef`, muted `#8e94a6`, dim `#646a7c`, working `#7dd3b0`, blocked `#f6a58a`, done `#b9a6f5`, accent `#f3d17a`, sky `#8ec5f5`. Fonts: UI `IBM Plex Sans KR`, terminal `IBM Plex Mono`.
- Helper failures never break Claude's statusline output or Codex's execution/exit code.
- Commits follow `~/.claude/commit-convention.md` (e.g. `feat(herdr-poke): …`, Korean noun-ending titles, no AI signature).

## Review Focus

1. Claude's statusline fires many times per second while streaming → labels written only when a value changes; `statusline` returns in < 50 ms even when `hc` is slow (label writes run detached).
2. A Codex run that ends before its first turn (no rollout file ever) → wrapper exits with Codex's code, writes nothing, leaves no background process.
3. Terminals closing → stale `poke.*` state must not show; mods only render terminals present in `terminals()`.
4. `rate_limits` windows missing or reset time in the past → show `–`, never a stale %.
5. Herdr window inactive / app restarted → partner stays the same (read from label), GIF just pauses.

---

### Task 1: Spikes that can change later tasks

**Files:**
- Create: `docs/plans/spike-notes.md`

- [ ] **Step 1:** In a real Herdr terminal run `codex --yolo --no-daemon`, send one prompt, then `lsof -a -p <codex pid> -Fn | grep rollout-`. Repeat with two Codex runs in the same folder started 1 s apart. Record whether each pid maps to exactly one rollout path.
- [ ] **Step 2:** Check whether an existing terminal's display `name` can be changed (`hc run --name X`, `hc label`, Herdr commands `herdr-app commands | grep -i name`). Trigger a blocked state and record what Herdr's toast says.
- [ ] **Step 3:** Time `hc label $HC_TERMINAL_ID poke.test=1` (10 runs, report p50/max).
- [ ] **Step 4:** Make a throwaway mod outside the repo with two `layout` blocks using `{"mod":"x","props":{"view":"a"}}` and `{"view":"b"}`; confirm `props().view` differs per copy (`herdr-app check`).
- [ ] **Step 5:** Write findings into `docs/plans/spike-notes.md`, one line each with the fallback chosen:
  - lsof fails → Task 5 uses the new-file fallback only.
  - name not changeable → keep Herdr's toast wording; partner shows in our UI only.
  - label p50 > 30 ms → statusline writes labels in a detached child.
  - per-copy props not supported → split into mods `poke-tabs`, `poke-side`, `poke-pane`, `poke-panel`, `poke-status` sharing copied `lib/`.
- [ ] **Step 6:** Commit `docs(herdr-poke): 사전 검증 결과 추가`.

### Task 2: Go module, state files, label writer

**Files:**
- Create: `helpers/poke/go.mod` (module `herdr-poke`), `helpers/poke/main.go`, `helpers/poke/internal/state/state.go`, `helpers/poke/internal/state/state_test.go`, `helpers/poke/internal/labels/labels.go`, `helpers/poke/internal/labels/labels_test.go`, `scripts/poke/build.sh`

**Interfaces:**
- Produces:
  - `state.Dir() string` → `$HERDR_POKE_STATE` or `~/.local/state/herdr-poke` (created 0700).
  - `state.WriteJSON(name string, v any) error` — atomic (temp + rename).
  - `state.UpdateLimits(provider string, l ProviderLimits) error` — read-modify-write `limits.json` under a `flock`.
  - `type Window struct{ Minutes int; Used float64; ResetsAt int64 }`, `type ProviderLimits struct{ Plan string; Windows []Window; Updated int64 }`. JSON keys: `minutes`, `used`, `resets_at`, `plan`, `windows`, `updated`. File shape `{"claude": ProviderLimits, "codex": ProviderLimits}`.
  - `labels.Set(terminal string, kv map[string]string) error` — runs `hc label <terminal> k=v…`, skips keys whose value equals the last value cached in `state.Dir()/labels-<terminal>.json`; no-op when `terminal==""`.
  - `main.go` dispatches subcommands `statusline | codex | usage | assets`; unknown → usage text, exit 2.

- [ ] **Step 1:** Tests: `TestWriteJSONAtomic` (concurrent writers leave valid JSON), `TestUpdateLimitsKeepsOtherProvider` (write claude then codex → both present), `TestLabelsSkipsUnchanged` (fake `hc` on PATH logging argv; second identical `Set` makes no call; changed value → one call with only that key).
- [ ] **Step 2:** `cd helpers/poke && go test ./...` → FAIL (undefined).
- [ ] **Step 3:** Implement the signatures above. `scripts/poke/build.sh`: `go build -o ~/.local/bin/herdr-poke ./helper`.
- [ ] **Step 4:** `go test ./...` → PASS; `scripts/poke/build.sh && herdr-poke` prints usage, exit 2.
- [ ] **Step 5:** Commit `feat(herdr-poke): go helper 골격과 상태 파일 추가`.

### Task 3: `herdr-poke statusline` (Claude)

**Files:**
- Create: `helpers/poke/internal/claude/statusline.go`, `helpers/poke/internal/claude/statusline_test.go`, `helpers/poke/internal/claude/testdata/*.json`, `helpers/poke/internal/claude/testdata/gen-golden.sh`

**Interfaces:**
- Consumes: `state.UpdateLimits`, `labels.Set`.
- Produces: `claude.Render(in Input, env Env) string`; `claude.Run(stdin io.Reader, stdout io.Writer) int` (always exit 0).
  - `Input` mirrors stdin JSON: `model.display_name`, `model.id`, `effort.level`, `cwd`, `context_window.{context_window_size,current_usage{input_tokens,cache_creation_input_tokens,cache_read_input_tokens}}`, `rate_limits.{five_hour,seven_day}.{used_percentage,resets_at}`, `output_style.name`, `cost.total_duration_ms`.
  - `Env`: `Home`, `Now time.Time`, `ClaudeJSONPath`, `AutoCompactWindow` (`CLAUDE_CODE_AUTO_COMPACT_WINDOW`), `Git func(dir) (branch string, dirty bool, ok bool)`.
  - Labels on `$HC_TERMINAL_ID`: `poke.model`, `poke.effort`, `poke.ctx` (int %).
  - Limits: provider `claude`, windows `{300, five_hour}` and `{10080, seven_day}`; omitted when absent or `resets_at` ≤ now.

- [ ] **Step 1:** `gen-golden.sh` runs `node ~/.claude/statusline.mjs` on each `testdata/in-*.json` with a temp `HOME` holding a fixture `.claude.json` and a temp git repo (clean and dirty), saving `golden-*.txt`. Fixtures cover: full input; no rate_limits; reset in the past; `[1m]` model id; output style `default` vs other; ctx 49/50/79/80%.
- [ ] **Step 2:** `TestRenderMatchesGolden` (byte-equal, including SGR codes, no trailing newline), `TestRunWritesLabelsAndLimits` (fake `hc`, temp state dir), `TestRunBadJSONStillPrints` (garbage stdin → prints empty line, exit 0).
- [ ] **Step 3:** `go test ./internal/claude` → FAIL.
- [ ] **Step 4:** Port `statusline.mjs` segment by segment (account, folder, git, model:effort, ctx, 5h/7d bars + countdown, output style, elapsed). Label and limit writes must not delay output beyond 50 ms (Review Focus 1; detached child if spike says so).
- [ ] **Step 5:** `go test ./...` → PASS.
- [ ] **Step 6:** Commit `feat(herdr-poke): claude statusline 이식`.

### Task 4: Codex rollout parser

**Files:**
- Create: `helpers/poke/internal/codex/rollout.go`, `helpers/poke/internal/codex/rollout_test.go`, `helpers/poke/internal/codex/testdata/rollout-*.jsonl` (redacted: no prompt text)

**Interfaces:**
- Produces:
  - `type Snapshot struct{ Model, Effort string; CtxPct int; HasCtx bool; Limits state.ProviderLimits; HasLimits bool }`
  - `codex.Apply(s *Snapshot, line []byte)` — updates from one JSONL line: `turn_context.payload.{model,effort}`; `event_msg` `token_count` → ctx from `info.last_token_usage.total_tokens` and `info.model_context_window` (or `task_started.payload.model_context_window`); `payload.rate_limits.{primary,secondary}` → windows by `window_minutes`, plan `plan_type`.
  - `codex.CtxPct(window, used int) int` = `100 - round(100*(W-12000-max(0,T-12000))/(W-12000))`, clamped 0–100.
  - `codex.SessionMeta(line []byte) (Meta, bool)` with `ID, Cwd, Originator, ParentThreadID, ForkedFromID string; Timestamp time.Time`.

- [ ] **Step 1:** Tests: `TestCtxPct` (W=258400: T=12000→0, T=135200→50, T=300000→100), `TestApplyRealSample` (fixture → model `gpt-6-astra`, effort set, window `{10080, 39.0, 1793780927}`, secondary absent), `TestApplyNullInfo` (token_count with `info:null` leaves ctx unset), `TestSessionMetaFilters`.
- [ ] **Step 2:** FAIL → **Step 3:** implement → **Step 4:** PASS.
- [ ] **Step 5:** Commit `feat(herdr-poke): codex 세션 로그 파서 추가`.

### Task 5: `herdr-poke codex -- <args>` wrapper

**Files:**
- Create: `helpers/poke/internal/codex/wrap.go`, `helpers/poke/internal/codex/find.go`, `helpers/poke/internal/codex/wrap_test.go`, `helpers/poke/internal/codex/find_test.go`

**Interfaces:**
- Consumes: Task 4 `Apply`, `SessionMeta`; Task 2 `labels.Set`, `state.UpdateLimits`.
- Produces: `codex.Wrap(args []string) int`; `codex.FindRollout(ctx, pid int, cwd string, started time.Time, opts FindOpts) (path string, certain bool, err error)` with `FindOpts{ SessionsDir string; Lsof func(pid int) ([]string, error); Poll time.Duration }`.

- [ ] **Step 1:** Tests:
  - `TestWrapPassesExitCode` (`Wrap([]string{"sh","-c","exit 3"})` → 3, stdio inherited).
  - `TestWrapNoRolloutNoLeftovers` (child exits immediately; returns within 1 s; no goroutine left — Review Focus 2).
  - `TestFindViaLsof` (fake Lsof returns the path → certain=true).
  - `TestFindFallbackPicksMatchingCwd` (two new files, different cwd → right one, certain=false).
  - `TestFindFallbackSameCwdOrder` (two files same cwd, spawn times 1 s apart → paired by order).
  - `TestFindIgnoresForks` (file with `forked_from_id` is skipped).
- [ ] **Step 2:** FAIL.
- [ ] **Step 3:** Implement: start child (`exec.Cmd`, inherited stdin/stdout/stderr, forward SIGINT/SIGTERM/SIGWINCH), record start time + cwd; goroutine waits for a rollout (`lsof -a -p <pid> -Fn` every 1 s, fallback scan of `~/.codex/sessions/YYYY/MM/DD/` for files whose `session_meta` matches), tails it (follow by offset every 1 s), on each change → `labels.Set($HC_TERMINAL_ID, poke.model/effort/ctx)` and `UpdateLimits("codex", …)`. Never write OSC sequences. `codex resume`: look up `~/.codex/state_5.sqlite` only if spike Step 1 showed lsof unusable; otherwise lsof covers it.
- [ ] **Step 4:** PASS. Manual: in a Herdr terminal `herdr-poke codex -- codex --yolo --no-daemon`, one prompt → `hc get $HC_TERMINAL_ID --json` shows `poke.model`, `poke.ctx`; `cat ~/.local/state/herdr-poke/limits.json` shows codex weekly.
- [ ] **Step 5:** Commit `feat(herdr-poke): codex 실행 래퍼 추가`.

### Task 6: `herdr-poke usage`

**Files:**
- Create: `helpers/poke/internal/usage/usage.go`, `helpers/poke/internal/usage/usage_test.go`, `helpers/poke/internal/usage/testdata/`

**Interfaces:**
- Produces: `usage.Run(now time.Time, roots Roots) error` writing `usage.json` = `{"days":[{"date":"YYYY-MM-DD","claude":int,"codex":int}…14 entries, oldest first],"updated":unix}`; `Roots{ ClaudeProjects, CodexSessions string }`; offsets in `usage-offsets.json` (`path → {size, offset}`; reset when size shrinks).
- Claude: lines `type=="assistant"`, tokens = `input + output + cache_creation + cache_read`, dedupe `message.id`, include `*/subagents/*.jsonl`, date = local date of `timestamp`.
- Codex: `type=="token_usage_record"`, tokens = `payload.usage.total_tokens`, dedupe `response_id`.

- [ ] **Step 1:** Tests: `TestClaudeDedupe` (3 lines same id → counted once), `TestCodexDedupe`, `TestFourteenDaysZeroFilled`, `TestIncrementalOnlyReadsNew` (append a line, second run adds only it), `TestLocalDateBoundary` (23:59 vs 00:01 local).
- [ ] **Step 2:** FAIL → **Step 3:** implement → **Step 4:** PASS; real run `herdr-poke usage && cat ~/.local/state/herdr-poke/usage.json` — today's claude+codex within 10% of pokeTokenBar `last-snapshot.json` totals (log the diff, don't fail).
- [ ] **Step 5:** Commit `feat(herdr-poke): 14일 사용량 집계 추가`.

### Task 7: `herdr-poke assets`

**Files:**
- Create: `helpers/poke/internal/assets/assets.go`, `helpers/poke/internal/assets/assets_test.go`
- Modify: `.gitignore` (add `mods/poke/assets/`)

**Interfaces:**
- Produces: `herdr-poke assets <id>` → ensures `mods/poke/assets/<id>.gif` (≤ 649: `https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/pokemon/versions/generation-v/black-white/animated/<id>.gif`) or `<id>.png` (`…/sprites/pokemon/<id>.png`), `<id>.mp3` (`https://play.pokemonshowdown.com/audio/cries/<slug>.mp3`, slug from `https://pokeapi.co/api/v2/pokemon-species/<id>` name with `-` removed), and `names.json` entry `{id: englishName}`. Idempotent; prints the sprite path relative to `~/.config/herdr-app`.
- `herdr-poke assets --companion` → copies the active companion's sprite from `~/Library/Application Support/PokeTokenBar/sprites/<id>-[sh]a.gif` (fallback `-s.png`) into `mods/poke/assets/companion-<id>.gif|png`.

- [ ] **Step 1:** Tests with `httptest.Server` base URLs: `TestAssetsGifUnder650`, `TestAssetsPngAbove649`, `TestAssetsIdempotent` (second call makes no requests), `TestCompanionCopyShiny`.
- [ ] **Step 2:** FAIL → **Step 3:** implement (base URLs overridable by env for tests) → **Step 4:** PASS.
- [ ] **Step 5:** Commit `feat(herdr-poke): 스프라이트·울음소리 내려받기 추가`.

### Task 8: Mod logic library (pure JS)

**Files:**
- Create: `mods/poke/lib/model.js`, `mods/poke/lib/partner.js`, `mods/poke/lib/companion.js`, `mods/poke/lib/format.js`, `mods/poke/lib/voice.js`, `tests/poke/mod-lib/*.test.mjs`, `package.json` (`"type":"module"`, script `test: node --test tests/*/mod-lib/*.test.mjs`)

**Interfaces (all plain ES modules, no imports from `herdr`/`gpui-kit`):**
- `model.js`: `URG = {blocked:0, working:1, done:2, idle:3}`; `byUrgency(list)`; `groups(agents) → {needs, working, done, idle}`; `waiting(agents) → blocked sorted by since asc`; `chips(agent, nowMs) → [{text, tip, color}]` (`ctx N%` / `waiting Nm`); `tabPill(agents) → {text, tone}`; `typeOf(branch, seed, mode, repoType)` (hash `h=seed; h=(h*31+code)>>>0`, types `grass fire water electric psychic normal`).
- `partner.js`: `excluded(companionState) → Set<int>` (all `dex[].chainOrder` ∪ `active.plannedPathIDs`); `pickPartner(randomFn, excludedSet) → int` in 1..1025.
- `companion.js`: `GRAD = {common:750e6, uncommon:1875e6, rare:3e9, legendary:6e9}`; `stageCost(rarity, k, stageIndex, difficulty=1)`; `progress(active, difficulty) → {pct, toNext, nextId}`.
- `format.js`: `ago(ms)` (`43s`, `6m`, `1h 5m`); `resetsIn(epochSec, nowMs)` (`4h 28m`, `5d 2h`, `–` when past); `pctOrDash(window|undefined)`; `tokensM(n)` (`100.4M`).
- `voice.js`: table `V` copied from the prototype (`docs/prototype/herdr-poke-template.html`, `const V=`), `say(nature, key, ...args)`; keys used in v1: `level`, `evoS`, `evoE`, `evening`.

- [ ] **Step 1:** Tests with spec values: `stageCost('rare',3,1)===1e9`; `progress({rarity:'rare',totalForms:3,stageIndex:1,usedAtStage:708705343,plannedPathIDs:[4,5,6]},1).pct` ≈ 70.9 and `nextId===6`; `excluded` of `{dex:[{chainOrder:[1,2,3]}],active:{plannedPathIDs:[4,5,6]}}` = {1..6}; `pickPartner` never returns excluded over 10 000 draws; `chips` → `ctx 83%` at 83, none at 79, `waiting 6m` at 6 min blocked, none at 4 min; `resetsIn` past → `–`; `typeOf` stable for same branch+seed.
- [ ] **Step 2:** `npm test` → FAIL → **Step 3:** implement → **Step 4:** PASS.
- [ ] **Step 5:** Commit `feat(herdr-poke): mod 로직 라이브러리 추가`.

### Task 9: Layout, theme, fonts, mod skeleton, keys

**Files:**
- Create: `layouts/poke.json`, `themes/poke.json`, `mods/poke/mod.json` (`{"place":"none","title":"Poke"}`), `mods/poke/main.js`, `mods/poke/views/*.js` (stubs), `mods/poke/fonts/` (IBM Plex Sans KR 400/500/600/700, IBM Plex Mono 400/500 `.ttf` + `OFL.txt`)
- Modify: `keys.json` (`global`: `"cmd-j": "poke.jump"`, `"cmd-k": "layout.toggle:poke-panel"`)

**Interfaces:**
- Produces: `main.js` default `View` routing on `props().view` ∈ `tabs | spaces | agents | companion | pane | panel | status` to `views/<view>.js` exports `render(state, cx) → element`; `nextCommand()` handles `"jump"`.
- Layout (`layouts/poke.json`): root `row`, `theme: "poke"`, `ui_font: "IBM Plex Sans KR"`, `font: "IBM Plex Mono"`; sidebar column 262 px (`spaces`, `agents`, `companion` blocks); work column (`tabs` 40 px, `main` tab block with pane header `{"content":{"mod":"poke","props":{"view":"pane"}},"height":40,"position":"top","reveal":"always"}` and footer if supported else header only, `status` 32 px); right block id `poke-panel` 290 px `{"mod":"poke","props":{"view":"panel"}}`.

- [ ] **Step 1:** `herdr-app check mods/poke` → passes with stub views; `cat layout.error` absent after `herdr-app` loads the layout (`run layout.switch` to `poke` manually or via palette).
- [ ] **Step 2:** `herdr-app shot /tmp/poke-skel.png` shows every region placeholder; ⌘K hides/shows the right panel.
- [ ] **Step 3:** Commit `feat(herdr-poke): 레이아웃과 mod 골격 추가`.

### Task 10: Tab bar and pane header

**Files:**
- Create/Modify: `mods/poke/views/tabs.js`, `mods/poke/views/pane.js`, `mods/poke/views/avatar.js`

**Interfaces:**
- Consumes: `lib/model.js`, `lib/format.js`; labels `poke.partner`, `poke.model`, `poke.effort`, `poke.ctx`, `poke.review_of`, `poke.findings` (optional; count shown only when present).
- Produces: `avatar(agentRow, {size}) → element` (sprite `image("mods/poke/assets/<id>.gif|png")`, provider badge from `icons/claude.svg|codex.svg`, motion: working `.loop("top",…)` bob, blocked `!` badge, idle grey (opacity), done-unread yellow dot).

- [ ] **Step 1:** Tabs: per tab avatar of most urgent agent, name, ⎇ when cwd is a worktree, pill (`! needs you` / `● N working` / `✓ new` / `✓ done` / `idle`), 2 px underline at 40% of type colour; click → `run("tab.select", …)`.
- [ ] **Step 2:** Pane header: avatar, program, status pill with symbol + elapsed, chips, model, effort pill, ctx meter; review strip (`↳ Reviewing <tab> · N found` on reviewer, `↳ Codex is reviewing this` on target, click → `agent.focus`).
- [ ] **Step 3:** `herdr-app check mods/poke` passes; `herdr-app shot` compared side by side with prototype v6 — same order of elements, colours from tokens.
- [ ] **Step 4:** Commit `style(herdr-poke): 탭 바와 pane 머리 추가`.

### Task 11: Sidebar (spaces, agents, companion)

**Files:**
- Create/Modify: `mods/poke/views/spaces.js`, `mods/poke/views/agents.js`, `mods/poke/views/companion.js`

**Interfaces:**
- Consumes: Task 8 libs; `readTextFile(home()+"/Library/Application Support/PokeTokenBar/companion-state.json")` + `watchFolder` on that folder; `defaults`-free difficulty (read `growthDifficulty` via `exec("defaults",["read","io.github.chattymin.poketokenbar","growthDifficulty"])`, default 1).
- Git info: same commands as the shipped `mods/spaces/main.js` (read it; don't import it).

- [ ] **Step 1:** Spaces rows: chip letter, name, branch, `⌃N`, `N waiting`, `N new`, avatar stack (max 4 + `+n`, by urgency, click → `agent.focus`).
- [ ] **Step 2:** Agents groups with sub-lines per spec.
- [ ] **Step 3:** Companion card: sprite (`assets --companion` output), name, ✦ shiny, level, nature, ability, types (from pokeTokenBar `pokemon-details-v1/<id>.json`), next form + progress bar (`lib/companion.progress`), fed today (`last-snapshot.json`), Dex/Seen counts. Missing pokeTokenBar → card says "pokeTokenBar not found" and the rest of the sidebar still renders.
- [ ] **Step 4:** check + shot; commit `style(herdr-poke): 사이드바 추가`.

### Task 12: Right panel and status bar

**Files:**
- Create/Modify: `mods/poke/views/panel.js`, `mods/poke/views/status.js`

**Interfaces:**
- Consumes: `limits.json`, `usage.json` (via `readTextFile` + `watchFolder(state dir)`), `lib/format.js`.

- [ ] **Step 1:** Panel tabs Changes / Usage / Dex per spec. Usage cards label windows by minutes (300 → `5h window`, 10080 → `Weekly`), `–` + "not on <plan>" when missing; Today split; 14 bars + average. Dex: Caught from `dex`, Seen from shared state `poke.seen` (`setShared`).
- [ ] **Step 1b:** Rail: when `layout.toggle:poke-panel` can't fold to a narrow width, ⌘K hides the panel entirely and the status bar carries the limits (spec's rail dropped). If `layout.resize` can set 44 px, render the rail (Claude/Codex 5h·wk) when the block is < 120 px wide.
- [ ] **Step 2:** Status bar: Claude `5h x% · wk y%`, Codex same, Today, `✓ N new`, "Jump to waiting · N ⌘J".
- [ ] **Step 3:** Stale input check (Review Focus 4): put a `resets_at` in the past in `limits.json` → both show `–`.
- [ ] **Step 4:** check + shot; commit `style(herdr-poke): 오른쪽 패널과 상태바 추가`.

### Task 13: Behaviours (partners, jump, sounds, toasts, evening)

**Files:**
- Create/Modify: `mods/poke/behaviours.js`, `mods/poke/main.js`, `settings.json` (`"sounds": {"enabled": false}`)

**Interfaces:**
- Consumes: `lib/partner.js`, `lib/voice.js`, Task 7 CLI.

- [ ] **Step 1:** Partner assignment (only in the `status` view copy, so it runs once): for each terminal in `terminals()` with an agent status and no `poke.partner`, pick → `hc(["label", addr, "poke.partner=<id>"])` → `exec("herdr-poke",["assets",id])`. Name the terminal after the partner if Task 1 found a way.
- [ ] **Step 2:** `poke.jump`: next of `lib/model.waiting(agents())`, cycling, `run("agent.focus",{terminal})`.
- [ ] **Step 3:** On an agent's status turning `blocked` → `playSound("mods/poke/assets/<id>.mp3")`. Companion level or stage increase (diff of `companion-state.json`) → `notify` with `say(nature, 'level'|'evoS'|'evoE', …)`.
- [ ] **Step 4:** Evening: 18:00–06:00 (data timer every 60 s) → dimmer surface tokens + drowsy idle/done avatars.
- [ ] **Step 5:** Seen: add each partner id to shared `poke.seen`.
- [ ] **Step 5b:** Type reroll: `nextCommand()` `"reroll"` → shared `poke.typeSeed = (seed*7+13)%9973`; mode in shared `poke.typeMode` (`random` default / `repo` / `off`). Both listed in the command palette via the mod's commands.
- [ ] **Step 6:** Manual check in Herdr: new Claude tab gets a partner not in the excluded set; restart Herdr → same partner (Review Focus 5); a permission prompt plays one cry and shows one toast; ⌘J cycles blocked sessions.
- [ ] **Step 7:** Commit `feat(herdr-poke): 짝꿍·이동·알림 동작 추가`.

### Task 14: Install, apply to Keno's setup, uninstall, README

**Files:**
- Create: `scripts/poke/install.sh`, `scripts/poke/uninstall.sh`, `README.md`

**Interfaces:**
- `install.sh`: build helper (`scripts/poke/build.sh`); for each of `~/.claude/settings.json`, `~/.zshrc`, `~/.config/herdr-app/settings.json`: copy to `<file>.herdr-poke-backup-<YYYYmmddHHMMSS>` then edit — Claude `statusLine.command` = `~/.local/bin/herdr-poke statusline` (JSON edit, other keys untouched), `.zshrc` line `alias codex='codex --yolo'` → `alias codex='herdr-poke codex -- codex --yolo --no-daemon'`, Herdr `sounds.enabled=false`; run `herdr-poke assets --companion` and `herdr-poke usage`; add a launchd agent or Herdr-side timer to rerun `usage` every 10 min (pick launchd `~/Library/LaunchAgents/dev.keno.herdr-poke.usage.plist`); show the layout (`herdr-app` command `layout.switch` to `poke`).
- `uninstall.sh`: restore the newest backup of each file, unload/remove the launchd plist, switch Herdr back to `main`, remove `~/.local/bin/herdr-poke`. Leaves the repo.
- README sections: What it is, Prerequisites (`brew install --cask herdr-app`, `brew install --cask chattymin/tap/poke-token-bar`, `brew install go`, Claude Code, Codex CLI), Install (launch Herdr once → `git init`/`remote add`/`fetch`/`checkout` in `~/.config/herdr-app` → `scripts/poke/install.sh`), Undo, Troubleshooting (`herdr-app status`, `*.error` files, `ctrl-alt-shift-escape`).

- [ ] **Step 1:** Dry run: `HOME=$(mktemp -d)` with copies of the three files → `install.sh` then `uninstall.sh` → files byte-identical to originals (`cmp`).
- [ ] **Step 2:** Real run on Keno's machine (Keno approved). Open a new zsh, start a Claude session and a Codex session in Herdr; confirm: Claude statusline looks exactly as before; both panes show model/effort/ctx; status bar shows Claude 5h/wk and Codex wk; usage chart filled.
- [ ] **Step 3:** Commit `chore(herdr-poke): 설치·되돌리기 스크립트와 README 추가`.
