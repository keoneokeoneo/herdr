---
name: herdr-poke-dev
description: Change or check the herdr-poke Herdr setup in ~/.config/herdr-app — the poke layout, the mods/poke views, or the Go helper herdr-poke (labels, limits, sprites, Codex attach). Use for "fix this in the sidebar/tab bar/panel", "why does X show Y", "check the spec for Z", or after a migration to confirm the setup works.
---

# herdr-poke: change → check → look → commit

1. Read `docs/guide.md`. Use its "What lives where" table to pick the files, and open
   only those. Product intent and earlier decisions are in
   `docs/specs/2026-11-03-herdr-poke-design.md`. Herdr's API: search the generated
   `AGENTS.md` and `mods/poke/views/gpui-kit.d.ts`. Don't read them whole.
2. Before editing, check the guide's Gotchas. The usual traps:
   - writing into `mods/poke/` at runtime;
   - showing GIFs that are not wrapped in `alive()` or not normalized;
   - forgetting the folded-sidebar form (`narrow()`) when changing a sidebar view;
   - a colour written as `"transparent"`.
3. Logic changes get a failing test first: `npm test` for `mods/poke/lib`,
   `cd helper && go test ./...` for the helper.
4. After the edit:
   - Run `herdr-app check mods/poke` and `herdr-app status`. Expect `check passed` and
     `"errors": []`.
   - If the helper changed, rebuild it with
     `cd helper && go build -o ~/.local/bin/herdr-poke .`.
5. Look at the result. Use `herdr-app shot`, or `screencapture -l <Herdr window id>` when
   the traffic lights matter. Crop to the region you changed before reading it. If you
   changed the sidebar, check it both expanded (262px) and folded (92px), then restore
   the width.
6. Commit with `~/.claude/commit-convention.md`
   (`fix|feat(herdr-poke): 한국어 명사형`, body lines ending ~함/~됨, no AI signature),
   then `git push`.
7. Tell the user what changed and what you verified. Name anything you could not see,
   for example a sprite that was not on screen.

If the change adds or moves a responsibility, a file, or a gotcha, update
`docs/guide.md` in the same commit. If the product behaviour changes, also update
"Current state" in the spec.
