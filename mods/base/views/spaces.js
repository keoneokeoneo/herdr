import { narrow, toggleSidebar } from "./brand.js";
// Sidebar: one row per space, with the faces of the agents working in it.
import { div, svg } from "gpui-kit";
import { h_flex, v_flex } from "gpui-base";
import { theme, spaces, run, openOverlay } from "herdr";
import { rows } from "./data.js";
import { avatar, pill, alpha } from "./ui.js";
import { waiting, spaceMeta, stackAgents } from "../lib/model.js";

function stack(list) {
  const active = stackAgents(list);
  const shown = active.slice(0, 4);
  const row = h_flex().flex_none().items_center();
  shown.forEach((r, i) => {
    row.child(div().id(`sp-av-${r.address}`).ml(i ? -8 : 0).cursor_pointer()
      .on_click(() => run("agent.focus", { terminal: r.address }))
      .child(avatar(r, { size: 22, key: "sp", noBadge: true, still: true })));
  });
  if (active.length > 4) row.child(div().ml(4).text_size(10).text_color(theme().dim).child(`+${active.length - 4}`));
  return row;
}

export function render(st) {
  const t = theme();
  const all = rows();
  const list = v_flex().size_full().gap(2).px(8).py(8).bg(t.chrome).overflow_y_scroll();
  if (narrow()) list.child(h_flex().justify_center().pb(4)
    .child(div().id("sidebar-expand").size(26).flex_none().flex().items_center().justify_center().rounded(6)
      .text_color(t.muted).cursor_pointer().hover((x) => x.bg(t.hover).text_color(t.foreground))
      .on_click(toggleSidebar).child(svg("mods/workspace-chrome/chevron-right.svg").size(14))));
  else list.child(h_flex().items_center().px(6).pb(4)
    .child(div().flex_1().text_size(10).font_weight(700).text_color(t.dim).child("SPACES"))
    .child(div().id("space-new").size(22).flex_none().flex().items_center().justify_center().rounded(6)
      .text_color(t.muted).cursor_pointer().hover((x) => x.bg(t.hover).text_color(t.foreground))
      .on_click(() => run("space.new")).child(svg("mods/workspace-chrome/plus.svg").size(13))));
  const all_ = spaces();
  if (narrow()) return rail(st, t, list, all, all_);
  all_.forEach((s, i) => {
    const mine = all.filter((r) => r.space === s.id);
    const wait = waiting(mine).length;
    const fresh = mine.filter((r) => r.status === "done").length;
    const letter = (s.name || "?").trim().charAt(0).toUpperCase();
    const chipC = s.active ? t.accent : t.muted;
    let meta = h_flex().gap(6).items_center().text_size(10).text_color(t.dim)
      .child(div().font_family("IBM Plex Mono").truncate().child(spaceMeta(s, mine)));
    if (wait) meta = meta.child(pill(`${wait} waiting`, t.blocked, { size: 9 }));
    if (fresh) meta = meta.child(pill(`${fresh} new`, t.accent, { size: 9 }));
    list.child(
      h_flex().id(`space-${s.id}`).items_center().gap(8).px(6).py(6).rounded(8)
        .bg(s.active ? t.selected : "#00000000").hover((x) => x.bg(t.hover)).cursor_pointer()
        .on_mouse_down("left", (e, cx) => { st.pressed = i; if (e.click_count === 2) { cx.stop_propagation(); run("space.rename", { id: s.id }); } })
        .on_mouse_up("left", () => {
          const from = st.pressed; st.pressed = null;
          if (from != null && from !== i) run("space.move", { id: all_[from].id, index: i + 1 });
        })
        .on_mouse_down("right", (_e, cx) => { cx.stop_propagation(); openOverlay({ mod: "menu", kind: "popup", width: 200, height: 260, props: { tab: "space", target: { id: s.id } } }); })
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
  });
  return list;
}

// Folded sidebar: one letter chip per space, a dot when something there needs a look.
function rail(st, t, list, all, all_) {
  list.px(0).items_center();
  all_.forEach((s, i) => {
    const mine = all.filter((r) => r.space === s.id);
    const dot = waiting(mine).length ? t.blocked : mine.some((r) => r.status === "done") ? t.accent : null;
    const chipC = s.active ? t.accent : t.muted;
    const chip = div().relative().flex_none().size(30).rounded(8).bg(alpha(chipC, s.active ? "33" : "1f")).text_color(chipC)
      .text_size(12).font_weight(700).flex().items_center().justify_center().child((s.name || "?").trim().charAt(0).toUpperCase());
    if (dot) chip.child(div().absolute().top(-2).right(-2).size(8).rounded_full().bg(dot));
    list.child(div().id(`space-${s.id}`).p(3).rounded(10).bg(s.active ? t.selected : "#00000000").hover((x) => x.bg(t.hover)).cursor_pointer()
      .on_mouse_down("right", (_e, cx) => { cx.stop_propagation(); openOverlay({ mod: "menu", kind: "popup", width: 200, height: 260, props: { tab: "space", target: { id: s.id } } }); })
      .on_click(() => run("space.select", { id: s.id })).child(chip));
  });
  return list;
}
