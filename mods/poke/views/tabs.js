import { div, svg } from "gpui-kit";
import { h_flex } from "gpui-base";
import { theme, run, shared, spaces, openOverlay } from "herdr";
import { rows } from "./data.js";
import { avatar, pill, tone, C, alpha } from "./ui.js";
import { byUrgency, tabPill, typeOf } from "../lib/model.js";

export const isWorktree = (cwd) => !!cwd && cwd.includes("/worktrees/");

export function tabType(branch, repoType) {
  return typeOf(branch ?? "", shared("poke.typeSeed") ?? 7, shared("poke.typeMode") ?? "random", repoType);
}

export function render(st, _cx) {
  const t = theme();
  const sp = spaces().find((s) => s.active);
  const all = rows();
  const items = (sp?.tabs ?? []).map((tab, i) => {
    const list = all.filter((r) => r.tab === tab.id);
    const top = byUrgency(list)[0];
    const p = tabPill(list);
    const branch = list.find((r) => r.branch)?.branch ?? sp.branch;
    const ty = tabType(branch);
    const wt = list.some((r) => isWorktree(r.cwd));
    let el = h_flex().id(`tab-${tab.id}`).flex_none().items_center().gap(7).h(32).px(10).rounded(9)
      .cursor_pointer().hover((s) => s.bg(t.hover))
      // Press on one tab and release on another to move it there (like the shipped tabs bar).
      .on_mouse_down("left", (e, cx) => { st.pressed = i; if (e.click_count === 2) { cx.stop_propagation(); run("tab.rename", { id: tab.id }); } })
      .on_mouse_up("left", () => {
        const from = st.pressed; st.pressed = null;
        if (from != null && from !== i) run("tab.move", { id: sp.tabs[from].id, index: i + 1 });
      })
      .on_mouse_down("right", (_e, cx) => { cx.stop_propagation(); openOverlay({ mod: "menu", kind: "popup", width: 200, props: { tab: "tab", target: { id: tab.id, space: sp.id } } }); })
      .on_click(() => run("tab.select", { id: tab.id }));
    if (tab.active) el = el.bg(t.raised);
    if (top) el = el.child(avatar(top, { size: 22, key: "tab", noBadge: true }));
    el = el.child(div().max_w(170).truncate().text_size(13).font_weight(500).text_color(tab.active ? t.foreground : t.muted).child(tab.zoomed ? `${tab.name} Z` : tab.name));
    if (wt) el = el.child(div().text_size(11).text_color(t.dim).child("⎇"));
    if (p.text) el = el.child(p.tone === "dim" ? div().text_size(11).text_color(t.dim).child(p.text) : pill(p.text, tone(t, p.tone)));
    if (tab.active) {
      el = el.child(div().id(`tab-close-${tab.id}`).p(2).rounded(4).text_color(t.dim).hover((x) => x.bg(t.hover).text_color(t.foreground))
        .on_mouse_down("left", (_e, cx) => cx.stop_propagation())
        .on_click((_e, cx) => { cx.stop_propagation(); run("tab.close", { id: tab.id }); })
        .child(svg("mods/pane/close.svg").size(11)));
    }
    // Type colour as a soft underline.
    return div().flex_none().flex().flex_col().gap(2).child(el)
      .child(div().h(2).mx(10).rounded_full().bg(ty ? alpha(C[ty], "66") : "#00000000"));
  });
  return h_flex().size_full().items_center().gap(4).px(10).bg(t.chrome)
    .children(items)
    .child(div().id("tab-new").px(8).text_size(16).text_color(t.dim).cursor_pointer()
      .on_click(() => openOverlay({ mod: "new-menu", kind: "popup", width: 250, height: 512 })).child("+"));
}
