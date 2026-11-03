import { div, svg } from "gpui-kit";
import { h_flex } from "gpui-base";
import { theme, props, panes, run, home } from "herdr";
import { rows } from "./data.js";
import { avatar, pill, SYMBOL, alpha, C } from "./ui.js";
import { chips } from "../lib/model.js";
import { ago } from "../lib/format.js";
import { isWorktree } from "./tabs.js";

const LABEL = { working: "Working", blocked: "Needs you", done: "Done", idle: "Idle" };

function btn(t, id, icon, command, args) {
  return div().id(id).flex_none().p(3).rounded(4).text_color(t.dim).cursor_pointer()
    .hover((x) => x.bg(t.hover).text_color(t.foreground))
    .on_click((_e, cx) => { cx.stop_propagation(); run(command, args); })
    .child(svg(`mods/pane/${icon}.svg`).size(13));
}

function actions(t, pane, zoomed) {
  return h_flex().flex_none().items_center().gap(1).pl(4)
    .child(btn(t, `pa-r-${pane}`, "split-right", "pane.split", { direction: "right", pane }))
    .child(btn(t, `pa-d-${pane}`, "split-down", "pane.split", { direction: "down", pane }))
    .child(btn(t, `pa-z-${pane}`, zoomed ? "unzoom" : "zoom", "pane.zoom", { pane }))
    .child(btn(t, `pa-x-${pane}`, "close", "pane.close", { pane }));
}

export function init(_s, cx) {
  // Elapsed times only change by the minute.
  cx.timer.every(20000, (cx) => cx.notify());
}

function meter(t, pct) {
  const c = pct >= 80 ? C.psychic : pct >= 60 ? t.accent : t.working;
  return div().w(46).h(5).rounded_full().bg(t.raised).overflow_hidden()
    .child(div().h(5).w(Math.round(46 * Math.min(100, pct) / 100)).bg(c));
}

export function render(_s, _cx) {
  const t = theme();
  const term = props()?.terminal;
  const p = panes().find((x) => x.pane === props()?.pane) ?? {};
  const all = rows();
  const r = all.find((x) => x.address === term);
  const bar = h_flex().size_full().items_center().gap(8).px(10).bg(t.chrome).text_size(12).overflow_hidden();
  if (!r) {
    return bar.child(div().text_color(t.muted).truncate().child((p.cwd ?? "").replace(home(), "~") || "Terminal"))
      .child(div().flex_1())
      .child(div().text_color(t.dim).font_family("IBM Plex Mono").text_size(11).child(p.branch ?? ""))
      .child(actions(t, props()?.pane, p.zoomed));
  }
  const col = t[r.status] ?? t.dim;
  let left = bar.child(avatar(r, { size: 24, key: "pane" }))
    .child(div().font_weight(700).text_color(t.foreground).flex_none().child(r.provider === "codex" ? "Codex" : r.provider === "claude" ? "Claude Code" : (r.program?.name ?? "Agent")))
    .child(pill(`${SYMBOL[r.status] ?? ""} ${LABEL[r.status] ?? r.status} · ${ago(Date.now() - r.since)}`, col));
  for (const c of chips(r, Date.now())) left = left.child(pill(c.text, C[c.color] ?? t.accent, { size: 10 }));
  // Review link: reviewer → target, or target ← reviewer.
  const target = r.reviewOf && all.find((x) => x.address === r.reviewOf);
  const reviewer = all.find((x) => x.reviewOf === r.address);
  const link = target ? { to: target, text: `↳ Reviewing ${target.tabName}` } : reviewer ? { to: reviewer, text: "↳ Codex is reviewing this" } : null;
  if (link) {
    const n = (target ? r : reviewer).findings;
    left = left.child(div().id("review-link").flex_none().cursor_pointer().on_click(() => run("agent.focus", { terminal: link.to.address }))
      .child(pill(n == null ? link.text : `${link.text} · ${n} found`, t.accent ?? C.water, { size: 10 })));
  }
  left = left.child(div().flex_1());
  const right = h_flex().flex_none().items_center().gap(6).text_size(11).text_color(t.muted);
  if (r.branch) right.child(div().font_family("IBM Plex Mono").text_color(isWorktree(r.cwd) ? "#8ec5f5" : t.dim).child(`${isWorktree(r.cwd) ? "⎇ " : ""}${r.branch}`));
  if (r.model) right.child(div().child(r.model));
  if (r.effort) right.child(pill(r.effort, r.provider === "claude" ? t.done : "#8ec5f5", { size: 10 }));
  if (r.ctx != null) right.child(div().text_color(t.dim).child("ctx")).child(meter(t, r.ctx)).child(div().child(`${r.ctx}%`));
  return left.child(right).child(actions(t, props()?.pane, p.zoomed));
}
