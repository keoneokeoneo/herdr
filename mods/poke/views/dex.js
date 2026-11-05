// Right panel body for Poke's layout: the Dex (caught, active partners, tab-type controls).
import { div } from "gpui-kit";
import { h_flex, v_flex } from "gpui-base";
import { theme, shared, setShared } from "herdr";
import { sprite } from "./ui.js";
import { minimap, initWalkers } from "./camp.js";
import { readJSON, PTB_DIR } from "./store.js";

async function loadDex(s) {
  const st = await readJSON(`${PTB_DIR()}/companion-state.json`);
  s.dex = (st?.dex ?? []).map((d) => {
    const id = d.finalID ?? d.chainOrder?.at(-1) ?? d.baseID;
    return { id, shiny: !!d.isShiny, name: d.names?.[String(id)]?.en ?? `#${id}` };
  });
}

export function init(s, cx) {
  const load = (cx) => cx.spawn(async (cx) => { await loadDex(s); cx.notify(); });
  load(cx);
  cx.timer.every(15000, load);
  initWalkers(s, cx, true, () => shared("poke.gardenEnabled") !== false);
}

function tile(t, key, fallbackId, label, shiny) {
  const face = sprite(key, 40) ?? (fallbackId != null ? sprite(fallbackId, 40) : null);
  return v_flex().w(56).items_center().gap(2)
    .child(div().relative().size(48).rounded(10).bg(t.surface).flex().items_center().justify_center()
      .child(face ?? div().text_size(9).text_color(t.dim).child(`#${fallbackId ?? "?"}`))
      .child(shiny ? div().absolute().top(2).right(4).text_size(10).text_color(t.accent).child("✦") : div()))
    .child(label ? div().w_full().text_center().truncate().text_size(9).text_color(t.muted).child(label) : div());
}

function dex(t, s) {
  const caught = s.dex ?? [];
  const seen = shared("poke.seen") ?? [];
  const names = shared("poke.names") ?? {};
  const gardenEnabled = shared("poke.gardenEnabled") !== false;
  const grid = () => h_flex().flex_wrap().gap(6);
  const box = v_flex().flex_none().gap(10)
    .child(div().text_size(12).font_weight(700).text_color(t.foreground).child(`Caught · ${caught.length}`));
  if (caught.length) {
    const g = grid();
    for (const c of caught) g.child(tile(t, `dex-${c.id}${c.shiny ? "-shiny" : ""}`, c.shiny ? null : c.id, c.name, c.shiny));
    box.child(g);
  } else box.child(div().text_size(11).text_color(t.dim).child("Nothing caught yet. Keep feeding your partner."));
  box.child(h_flex().pt(6).items_center().gap(8)
    .child(div().text_size(12).font_weight(700).text_color(t.foreground).child(`Active · ${seen.length}`))
    .child(div().flex_1())
    .child(div().id("garden-toggle").px(8).py(4).rounded(6).bg(t.surface).text_size(11)
      .text_color(gardenEnabled ? t.accent : t.muted).cursor_pointer().hover((x) => x.bg(t.hover))
      .tooltip(gardenEnabled ? "Hide the garden minimap" : "Show the garden minimap")
      .accessibility_label(gardenEnabled ? "Garden enabled. Click to disable" : "Garden disabled. Click to enable")
      .on_click(() => setShared("poke.gardenEnabled", !gardenEnabled))
      .child(gardenEnabled ? "Garden: On" : "Garden: Off")));
  if (seen.length) {
    const g = grid();
    for (const id of seen) g.child(tile(t, id, id, names[id] ?? null, false));
    box.child(g);
  } else box.child(div().text_size(11).text_color(t.dim).child("Partners from open agent sessions appear here."));
  const mode = shared("poke.typeMode") ?? "random";
  box.child(h_flex().pt(10).gap(8).text_size(10).text_color(t.dim)
    .child(div().child(`Tab types: ${mode}`))
    .child(div().id("reroll").text_color(t.muted).cursor_pointer().hover((x) => x.text_color(t.foreground))
      .on_click(() => setShared("poke.typeSeed", (Number(shared("poke.typeSeed") ?? 7) * 7 + 13) % 9973)).child("↻ reroll"))
    .child(div().id("tmode").text_color(t.muted).cursor_pointer().hover((x) => x.text_color(t.foreground))
      .on_click(() => setShared("poke.typeMode", mode === "random" ? "off" : "random")).child(mode === "random" ? "turn off" : "turn on")));
  return box;
}

export function render(s, cx) {
  const t = theme();
  return v_flex().size_full().gap(12).px(12).pt(4).pb(12).bg(t.chrome).border_l_1().border_color(t.border).overflow_y_scroll()
    .child(dex(t, s))
    .child(shared("poke.gardenEnabled") !== false ? minimap(s) : div());
}
