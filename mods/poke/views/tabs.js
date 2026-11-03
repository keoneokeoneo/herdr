import { div } from "gpui-kit";
import { h_flex } from "gpui-base";
import { theme, run, shared, spaces } from "herdr";
import { rows } from "./data.js";
import { avatar, pill, tone, C, alpha } from "./ui.js";
import { byUrgency, tabPill, typeOf } from "../lib/model.js";

export const isWorktree = (cwd) => !!cwd && cwd.includes("/worktrees/");

export function tabType(branch, repoType) {
  return typeOf(branch ?? "", shared("poke.typeSeed") ?? 7, shared("poke.typeMode") ?? "random", repoType);
}

export function render(_s, _cx) {
  const t = theme();
  const sp = spaces().find((s) => s.active);
  const all = rows();
  const items = (sp?.tabs ?? []).map((tab) => {
    const list = all.filter((r) => r.tab === tab.id);
    const top = byUrgency(list)[0];
    const p = tabPill(list);
    const branch = list.find((r) => r.branch)?.branch ?? sp.branch;
    const ty = tabType(branch);
    const wt = list.some((r) => isWorktree(r.cwd));
    let el = h_flex().id(`tab-${tab.id}`).flex_none().items_center().gap(7).h(32).px(10).rounded(9)
      .cursor_pointer().hover((s) => s.bg(t.hover))
      .on_click(() => run("tab.select", { id: tab.id }));
    if (tab.active) el = el.bg(t.raised);
    if (top) el = el.child(avatar(top, { size: 22, key: "tab", noBadge: true }));
    el = el.child(div().text_size(13).font_weight(500).text_color(tab.active ? t.foreground : t.muted).child(tab.name));
    if (wt) el = el.child(div().text_size(11).text_color(t.dim).child("⎇"));
    if (p.text) el = el.child(p.tone === "dim" ? div().text_size(11).text_color(t.dim).child(p.text) : pill(p.text, tone(t, p.tone)));
    // Type colour as a soft underline.
    return div().flex_none().flex().flex_col().gap(2).child(el)
      .child(div().h(2).mx(10).rounded_full().bg(ty ? alpha(C[ty], "66") : "#00000000"));
  });
  return h_flex().size_full().items_center().gap(4).px(10).bg(t.chrome)
    .children(items)
    .child(div().id("tab-new").px(8).text_size(16).text_color(t.dim).cursor_pointer()
      .on_click(() => run("tab.new", {})).child("+"));
}
