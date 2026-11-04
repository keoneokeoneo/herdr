// Right panel header: picks which child of the poke-panel-body stack shows.
import { div } from "gpui-kit";
import { h_flex } from "gpui-base";
import { theme, layout, run } from "herdr";

const TABS = [["poke-usage", "Usage"], ["poke-changes", "Changes"], ["poke-files", "Files"], ["poke-dex", "Poke"], ["poke-obsidian", "Obsidian"]];

function find(node, id) {
  if (!node || typeof node !== "object") return null;
  if (node.id === id) return node;
  for (const c of node.children ?? []) { const f = find(c, id); if (f) return f; }
  return null;
}

export function render() {
  const t = theme();
  const active = find(layout(), "poke-panel-body")?.active ?? "poke-usage";
  const head = h_flex().gap(3).p(3).rounded(8).bg(t.surface).w_full();
  for (const [id, label] of TABS) {
    const on = active === id;
    head.child(div().id(`ptab-${id}`).flex_1().py(4).rounded(6).text_center().text_size(11).font_weight(600)
      .bg(on ? t.raised : "#00000000").text_color(on ? t.foreground : t.muted).cursor_pointer()
      .hover((x) => x.text_color(t.foreground))
      .on_click(() => run("layout.select", { stack: "poke-panel-body", child: id }))
      .child(label));
  }
  return h_flex().size_full().items_center().px(12).pt(8).bg(t.chrome).border_l_1().border_color(t.border).child(head);
}
