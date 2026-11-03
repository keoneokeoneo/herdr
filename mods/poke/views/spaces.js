// Sidebar: one row per space, with the faces of the agents working in it.
import { div } from "gpui-kit";
import { h_flex, v_flex } from "gpui-base";
import { theme, spaces, run } from "herdr";
import { rows } from "./data.js";
import { avatar, pill, alpha } from "./ui.js";
import { byUrgency, waiting } from "../lib/model.js";

function stack(list) {
  const shown = byUrgency(list).slice(0, 4);
  const row = h_flex().flex_none().items_center();
  shown.forEach((r, i) => {
    row.child(div().id(`sp-av-${r.address}`).ml(i ? -8 : 0).cursor_pointer()
      .on_click(() => run("agent.focus", { terminal: r.address }))
      .child(avatar(r, { size: 22, key: "sp", noBadge: true, still: true })));
  });
  if (list.length > 4) row.child(div().ml(4).text_size(10).text_color(theme().dim).child(`+${list.length - 4}`));
  return row;
}

export function render() {
  const t = theme();
  const all = rows();
  const list = v_flex().size_full().gap(2).px(8).py(8).bg(t.chrome).overflow_y_scroll();
  list.child(div().px(6).pb(4).text_size(10).font_weight(700).text_color(t.dim).child("SPACES"));
  for (const s of spaces()) {
    const mine = all.filter((r) => r.space === s.id);
    const wait = waiting(mine).length;
    const fresh = mine.filter((r) => r.status === "done").length;
    const letter = (s.name || "?").trim().charAt(0).toUpperCase();
    const chipC = s.active ? t.accent : t.muted;
    let meta = h_flex().gap(6).items_center().text_size(10).text_color(t.dim)
      .child(div().font_family("IBM Plex Mono").truncate().child(s.branch ? `⎇ ${s.branch}` : s.folder?.replace(/^.*\//, "") ?? ""));
    if (wait) meta = meta.child(pill(`${wait} waiting`, t.blocked, { size: 9 }));
    if (fresh) meta = meta.child(pill(`${fresh} new`, t.accent, { size: 9 }));
    list.child(
      h_flex().id(`space-${s.id}`).items_center().gap(8).px(6).py(6).rounded(8)
        .bg(s.active ? t.selected : "transparent").hover((x) => x.bg(t.hover)).cursor_pointer()
        .on_click(() => run("space.select", { id: s.id }))
        .child(div().flex_none().size(22).rounded(6).bg(alpha(chipC, "2e")).text_color(chipC)
          .text_size(11).font_weight(700).flex().items_center().justify_center().child(letter))
        .child(v_flex().flex_1().min_w(0).gap(1)
          .child(h_flex().items_center().gap(6)
            .child(div().truncate().text_size(12).font_weight(600).text_color(t.foreground).child(s.name))
            .child(div().text_size(10).text_color(t.dim).child(s.index >= 1 && s.index <= 9 ? `⌃${s.index}` : "")))
          .child(meta))
        .child(stack(mine)),
    );
  }
  return list;
}
