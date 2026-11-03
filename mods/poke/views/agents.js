// Sidebar: every agent, grouped by what it needs from Keno.
import { div } from "gpui-kit";
import { h_flex, v_flex } from "gpui-base";
import { theme, run } from "herdr";
import { rows } from "./data.js";
import { avatar, pill, C, alpha } from "./ui.js";
import { groups, chips } from "../lib/model.js";
import { ago } from "../lib/format.js";

export function init(_s, cx) {
  cx.timer.every(20000, (cx) => cx.notify());
}

function subline(r, all) {
  if (r.reviewOf) {
    const tg = all.find((x) => x.address === r.reviewOf);
    return `↳ reviewing ${tg?.tabName ?? "a session"}${r.findings != null ? ` · ${r.findings} found` : ""}`;
  }
  if (r.status === "blocked") return r.title || "Waiting for you";
  if (r.status === "done") return "New · not checked yet";
  return [r.model, r.effort, r.ctx != null ? `ctx ${r.ctx}%` : null].filter(Boolean).join(" · ");
}

function row(t, r, all, now) {
  const head = h_flex().items_center().gap(6).min_w(0)
    .child(div().flex_none().px(5).rounded(4).bg(alpha(t.muted, "22")).text_size(9).font_weight(700).text_color(t.muted)
      .child((r.spaceName || "?").charAt(0).toUpperCase()))
    .child(div().truncate().text_size(12).font_weight(600).text_color(t.foreground).child(r.tabName || r.address));
  for (const c of chips(r, now)) head.child(pill(c.text, C[c.color] ?? t.accent, { size: 9 }));
  head.child(div().flex_1()).child(div().flex_none().text_size(10).text_color(t.dim).child(ago(now - r.since)));
  const sub = subline(r, all);
  return h_flex().id(`ag-${r.address}`).items_center().gap(8).px(6).py(5).rounded(8)
    .bg(r.focused ? t.selected : "transparent").hover((x) => x.bg(t.hover)).cursor_pointer()
    .on_click(() => run("agent.focus", { terminal: r.address }))
    .child(avatar(r, { size: 26, key: "ag" }))
    .child(v_flex().flex_1().min_w(0).gap(1).child(head)
      .child(div().truncate().text_size(10).text_color(r.status === "blocked" ? t.blocked : t.dim).child(sub || " ")));
}

const TITLES = [["needs", "Needs you", "blocked"], ["working", "Working", "working"], ["done", "Done", "done"], ["idle", "Idle", "idle"]];

export function render() {
  const t = theme();
  const now = Date.now();
  const all = rows();
  const g = groups(all);
  const list = v_flex().size_full().gap(2).px(8).py(8).bg(t.chrome).overflow_y_scroll();
  list.child(div().px(6).pb(2).text_size(10).font_weight(700).text_color(t.dim).child("AGENTS"));
  if (!all.length) list.child(div().px(6).text_size(11).text_color(t.dim).child("No agents yet. Start claude or codex in a pane."));
  for (const [k, title, tone] of TITLES) {
    const items = g[k];
    if (!items.length) continue;
    list.child(h_flex().px(6).pt(6).gap(6).items_center().text_size(10).font_weight(700).text_color(t[tone] ?? t.dim)
      .child(title).child(div().text_color(t.dim).child(String(items.length))));
    for (const r of [...items].sort((a, b) => a.since - b.since)) list.child(row(t, r, all, now));
  }
  return list;
}
