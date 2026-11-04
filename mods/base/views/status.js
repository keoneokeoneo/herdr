// Status bar: both providers' limits, today's tokens, unread done, jump to waiting.
import { div, image } from "gpui-kit";
import { h_flex } from "gpui-base";
import { theme, run } from "herdr";
import { rows } from "./data.js";
import { keepFresh, STATE_DIR, WIN_LABEL } from "./store.js";
import { loadUsage, USAGE_DIR } from "./usagefiles.js";
import { pctOrDash, tokensM } from "../lib/format.js";
import { usageState } from "../lib/usage.js";
import { waiting } from "../lib/model.js";
import { start, jump, notes } from "../behaviours.js";

export function init(s, cx) {
  keepFresh(s, cx, STATE_DIR, loadUsage, [USAGE_DIR]);
  start(s, cx);
  cx.timer.every(60000, (cx) => cx.notify());
}

// Key commands (base.jump, base.notes) reach shown copies; only this always-shown copy acts on them.
export function command(s, c) {
  if (c === "jump") jump(s);
  else if (c === "notes") notes();
}

const sep = (t) => div().w(1).h(14).bg(t.border);

function prov(t, name, icon, p, now) {
  return h_flex().gap(6).items_center()
    .child(image(icon).w(13).h(13).flex_none())
    .child(div().text_color(t.muted).child(name))
    .child(div().text_color(t.foreground).child((p?.windows ?? []).filter((w) => w.minutes === 300 || w.minutes === 10080).map((w) => `${WIN_LABEL(w.minutes)} ${pctOrDash(w, now)}`).join(" · ") || "–"));
}

export function render(s, cx) {
  const t = theme();
  const now = Date.now();
  const L = s.limits ?? {};
  const all = rows();
  const wait = waiting(all).length;
  const fresh = all.filter((r) => r.status === "done").length;
  const { today, updatedText, stale } = usageState(s.usage, now);
  const bar = h_flex().size_full().items_center().gap(12).px(12).bg(t.chrome).border_t_1().border_color(t.border).text_size(11)
    .child(prov(t, "Claude", "icons/claude.svg", L.claude, now)).child(sep(t))
    .child(prov(t, "Codex", "icons/codex.svg", L.codex, now)).child(sep(t))
    .child(h_flex().gap(6).child(div().text_color(t.muted).child("Today"))
      .child(div().text_color(t.foreground).child(today ? tokensM(today.claude + today.codex) : "–")));
  if (stale) bar.child(div().text_size(10).text_color(t.dim).child(updatedText));
  if (fresh) bar.child(sep(t)).child(div().text_color(t.accent).child(`✓ ${fresh} new`));
  bar.child(div().flex_1());
  if (wait) {
    bar.child(div().id("jump").flex_none().px(10).py(2).rounded_full().bg(t.blocked + "26").text_color(t.blocked).font_weight(600).cursor_pointer()
      .on_click(() => jump(s))
      .child(`Jump to waiting · ${wait}  ⌘J`));
  }
  bar.child(div().id("palette").flex_none().text_color(t.dim).cursor_pointer().hover((x) => x.text_color(t.foreground))
    .on_click(() => run("app.palette")).child("⌘⇧P"));
  return bar;
}
