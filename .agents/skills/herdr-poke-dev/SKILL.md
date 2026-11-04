---
name: herdr-poke-dev
description: Change or check the Herdr setups in ~/.config/herdr-app — the neutral base (mods/base: sidebar, tab bar, usage bar, right panel), the poke persona (mods/poke), the layouts, or the Go helper herdr-poke (Codex labels and limits, daily usage, sprites). Use for "fix this in the sidebar/tab bar/panel", "why does X show Y", "check the spec for Z", or after a migration to confirm the setup works.
---

# base / poke / herdr-poke: change → check → look → commit

1. Read `docs/guide.md`, then the setup guide: `docs/modes/base.md` for the sidebar, tab bar, usage bar, panel and
   the usage data contract; `docs/modes/poke.md` for the companion, Dex, cries and the helper. Use its "What lives
   where" table to pick the files, and open only those. Product intent and earlier decisions are in
   `docs/specs/2026-11-03-herdr-poke-design.md`. Herdr's API: search the generated
   `AGENTS.md` and `mods/base/gpui-kit.d.ts`. Don't read them whole.
2. Before editing, check the guide's Gotchas. The usual traps:
   - writing into `mods/base/` or `mods/poke/` at runtime;
   - importing across mods (mods cannot; use shared state, labels, commands or the data files);
   - showing GIFs that are not wrapped in `alive()` or not normalized;
   - forgetting the folded-sidebar form (`narrow()`) when changing a sidebar view;
   - a colour written as `"transparent"`.
3. Logic changes get a failing test first: `npm test` for `mods/base/lib` and `mods/poke/lib` (it also checks layouts and docs),
   `cd helpers/poke && go test ./...` for the helper.
4. After the edit:
   - Run `herdr-app check mods/base`, `herdr-app check mods/poke` and `herdr-app status`. Expect `check passed` and
     `"errors": []`.
   - If the helper changed, rebuild it with
     `cd helpers/poke && go build -o ~/.local/bin/herdr-poke .`.
5. Look at the result. Use `herdr-app shot`, or `screencapture -l <Herdr window id>` when
   the traffic lights matter. Crop to the region you changed before reading it. If you
   changed the sidebar, check it both expanded (262px) and folded (92px), then restore
   the width.
6. Commit in the repo style (`fix|feat|refactor(herdr|herdr-poke): 한국어 명사형`, body lines ending ~함/~됨,
   no AI signature); push only when asked.
7. Tell the user what changed and what you verified. Name anything you could not see,
   for example a sprite that was not on screen.

If the change adds or moves a responsibility, a file, or a gotcha, update
the setup guide (`docs/modes/base.md` or `docs/modes/poke.md`) and, for a new setup, `scripts/setups.json` and the README
in the same commit. If the product behaviour changes, also update
"Current state" in the spec.
