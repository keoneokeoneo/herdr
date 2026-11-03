// Sidebar top: our mark in place of Herdr's, with the same search and sidebar buttons.
import { div, image, svg } from "gpui-kit";
import { h_flex } from "gpui-base";
import { theme, run, layout, window, shared, setShared } from "herdr";

function find(node, id) {
  if (!node || typeof node !== "object") return null;
  if (node.id === id) return node;
  for (const c of node.children ?? []) { const f = find(c, id); if (f) return f; }
  return null;
}
const narrow = () => (find(layout(), "sidebar")?.size?.px ?? 262) < 120;

// Same behaviour as the shipped workspace-chrome toggle: fold to a rail and back.
function toggleSidebar() {
  const width = find(layout(), "sidebar")?.size?.px;
  if (!narrow()) setShared("sidebar.width", width);
  setShared("sidebar.collapsed", !narrow());
  run("layout.resize", { id: "sidebar", px: narrow() ? (shared("sidebar.width") ?? 262) : window()?.platform === "macos" ? 76 : 44 });
}

function button(t, id, icon, action) {
  return div().id(id).size(26).flex_none().flex().items_center().justify_center().rounded(6).text_color(t.muted)
    .hover((x) => x.bg(t.hover).text_color(t.foreground))
    .on_mouse_down("left", (_e, cx) => cx.stop_propagation()).on_click(action)
    .child(svg(`mods/workspace-chrome/${icon}.svg`).size(15));
}

export function render() {
  const t = theme();
  const w = window();
  const mac = w?.platform === "macos" && !w?.fullscreen;
  const row = h_flex().size_full().items_center().gap(4).pr(10).bg(t.chrome).pl(narrow() ? 6 : mac ? 84 : 14);
  if (narrow()) return mac ? row : row.child(button(t, "sidebar-toggle", "sidebar", toggleSidebar));
  return row
    .child(h_flex().flex_1().gap(8).items_center()
      .child(image("mods/poke/brand/pokeball.svg").size(18).flex_none())
      .child(div().text_size(13.5).font_weight(600).text_color(t.foreground).child("Herdr")))
    .child(button(t, "search", "search", () => run("mod.open", { name: "picker" })))
    .child(button(t, "sidebar-toggle", "sidebar", toggleSidebar));
}
