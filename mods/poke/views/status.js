// Status bar: both providers' limits, today's tokens, unread done, jump to waiting.
import { div } from "gpui-kit";
import { h_flex } from "gpui-base";
import { theme } from "herdr";
import { rows } from "./data.js";
import { keepFresh, loadUsage, STATE_DIR, win } from "./store.js";
import { pctOrDash, tokensM } from "../lib/format.js";
import { waiting } from "../lib/model.js";
import { start } from "../behaviours.js";

export function init(s, cx) {
  keepFresh(s, cx, STATE_DIR, loadUsage);
  start(s, cx);
}

// Behaviours (Task 13) hook in here; commands reach only this copy.
export function command(s, c, cx) {
  s.onCommand?.(c, cx);
}

const sep = (t) => div().w(1).h(14).bg(t.border);

function prov(t, name, color, p, now) {
  return h_flex().gap(6).items_center()
    .child(div().size(7).rounded_full().bg(color))
    .child(div().text_color(t.muted).child(name))
    .child(div().text_color(t.foreground).child(`5h ${pctOrDash(win(p, 300), now)} · wk ${pctOrDash(win(p, 10080), now)}`));
}

export function render(s, cx) {
  const t = theme();
  const now = Date.now();
  const L = s.limits ?? {};
  const all = rows();
  const wait = waiting(all).length;
  const fresh = all.filter((r) => r.status === "done").length;
  const today = s.usage?.days?.at(-1);
  const bar = h_flex().size_full().items_center().gap(12).px(12).bg(t.chrome).border_t_1().border_color(t.border).text_size(11)
    .child(prov(t, "Claude", t.done, L.claude, now)).child(sep(t))
    .child(prov(t, "Codex", "#8ec5f5", L.codex, now)).child(sep(t))
    .child(h_flex().gap(6).child(div().text_color(t.muted).child("Today"))
      .child(div().text_color(t.foreground).child(today ? tokensM(today.claude + today.codex) : "–")));
  if (fresh) bar.child(sep(t)).child(div().text_color(t.accent).child(`✓ ${fresh} new`));
  bar.child(div().flex_1());
  if (wait) {
    bar.child(div().id("jump").flex_none().px(10).py(2).rounded_full().bg(t.blocked + "26").text_color(t.blocked).font_weight(600).cursor_pointer()
      .on_click(() => s.onCommand?.("jump"))
      .child(`Jump to waiting · ${wait}  ⌘J`));
  }
  return bar;
}
