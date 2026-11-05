// The partner strip along the bottom of the Poke layout: every agent session's
// partner (the same one the sidebar and Dex show) wanders around its own column.
// Working partners walk, idle ones stroll and nap, blocked and done ones stop and
// show a mark. Click one to go to its session.
import { div, image } from "gpui-kit";
import { h_flex } from "gpui-base";
import { agents, terminals, spaces, theme, shared, settings, run } from "herdr";
import { stripRows, slots, partnerState, stripTicker } from "../lib/strip.js";
import { walkArt, pose } from "../lib/walk.js";
import { sprite, alive, alpha, C } from "./ui.js";

const SIZE = 64;        // sprite box
const SLOT_CAP = 200;   // widest column per partner
const GROUND = 30;      // the grass band along the bottom, px
// A night meadow under the calm Poke theme: a dusk sky over a grass band.
const SKY = ["#1b1d26", "#1c2027"];
const GRASS = { band: "#1f2c27", edge: "#2b3f34", tuft: "#3a5444", shadow: "#0e1a1466" };
const zoom = () => (Number(settings()?.zoom) > 0 ? Number(settings().zoom) : 1);
const rowsNow = () => stripRows(agents(), terminals(), spaces());

export function init(s, cx) {
  s.width = 0;
  // Each partner wanders within its column, leaving room for its name tag's neighbours.
  s.ticker = stripTicker({
    range: () => Math.max(8, Math.round(slots(rowsNow().length, s.width, SLOT_CAP).width / 2 - SIZE / 2 - 6)),
    speed: { walk: 60, stroll: 26 },
  });
  s.ticker.step(rowsNow(), shared("poke.sprites") ?? [], Date.now());
  cx.timer.every(250, (cx) => {
    if (s.width > 0 && s.ticker.step(rowsNow(), shared("poke.sprites") ?? [], Date.now())) cx.notify();
  });
}

function figure(s, row, t) {
  const w = s.ticker.get(row.address);
  const art = walkArt(row.partner, shared("poke.sprites") ?? []);
  const shadow = div().absolute().left(SIZE * 0.17).top(SIZE - 7).w(SIZE * 0.66).h(7).rounded_full().bg(GRASS.shadow);
  const box = div().relative().flex_none().w(SIZE).h(SIZE).child(shadow);
  if (art && w?.plan) {
    const src = art[pose(w.plan, w.mood, Date.now())];
    return box.child(div().absolute().left(0).top(0).child(alive(image(src).w(SIZE).h(SIZE).flex_none(), src)));
  }
  const still = sprite(row.partner, SIZE);
  return box.child(div().absolute().left(0).top(0).w(SIZE).h(SIZE).flex().items_center().justify_center()
    .child(still ?? div().text_size(20).text_color(t.dim).child("?")));
}

// A small speech bubble over the head, tinted with the status colour.
function badge(state, status, t) {
  if (!state.mark) return div().h(18);
  const blocked = status === "blocked";
  const color = state.mark === "zZ" ? t.muted : t[status] ?? C[state.tone] ?? t.muted;
  let el = div().h(18).px(7).rounded_full().flex().items_center().text_size(blocked ? 12 : 10.5).font_weight(700)
    .border_1().border_color(alpha(color, blocked ? "ff" : "66"))
    .bg(blocked ? color : alpha(t.raised, "e6")).text_color(blocked ? t.canvas ?? "#1b1d26" : color)
    .child(state.mark);
  if (blocked) el = el.loop("opacity", { from: 0.75, to: 1, period: 900 });
  else if (status === "working") el = el.loop("opacity", { from: 0.6, to: 1, period: 1400 });
  return el;
}

// A name tag on the grass: status dot and session name in a soft pill.
function tag(row, colW, t, hovered) {
  const dot = t[row.status] ?? t.idle ?? t.muted;
  return h_flex().max_w(colW - 10).h(18).px(8).gap(5).items_center().rounded_full()
    .bg(alpha(t.canvas ?? "#1b1d26", "cc")).border_1().border_color(hovered ? alpha(t.accent, "99") : alpha(dot, "40"))
    .child(div().flex_none().size(6).rounded_full().bg(dot))
    .child(div().truncate().text_size(10.5).font_weight(500).text_color(hovered ? t.foreground : t.muted).child(row.name));
}

function partner(s, row, home, colW, names, t) {
  const w = s.ticker.get(row.address);
  const state = partnerState(row.status, w?.mood === "sleep");
  const species = row.partner == null ? "Partner arriving…" : names[row.partner] ?? `#${row.partner}`;
  const detail = `${row.name} · ${row.space} — ${species}: ${state.detail}`;
  const offset = w?.plan?.to ?? 0;
  const hovered = s.hover === row.address;
  return div().id(`strip-${row.address}`).absolute().top(2).left(home + offset - colW / 2).w(colW)
    .transition("left", { duration: Math.max(1, w?.plan?.dur ?? 0), easing: "linear" })
    .flex().flex_col().items_center()
    .cursor_pointer().tooltip(detail).accessibility_label(detail)
    .on_hover((over, cx) => { const next = over ? row.address : s.hover === row.address ? null : s.hover; if (next !== s.hover) { s.hover = next; cx.notify(); } })
    .on_click(() => run("agent.focus", { terminal: row.address }))
    .child(badge(state, row.status, t))
    .child(figure(s, row, t))
    .child(tag(row, colW, t, hovered));
}

export function render(s) {
  const t = theme();
  const rows = rowsNow();
  const { width: colW, homes } = slots(rows.length, s.width, SLOT_CAP);
  const names = shared("poke.names") ?? {};
  const root = div().id("poke-strip").relative().size_full().overflow_hidden()
    .bg(SKY[1])
    .on_size_change((size, cx) => {
      const width = Math.round(size.width / zoom());
      if (width !== s.width) { s.width = width; cx.notify(); }
    })
    .child(div().absolute().left(0).right(0).top(0).h(1).bg(t.border))
    .child(div().absolute().left(0).right(0).bottom(0).h(GROUND).bg(GRASS.band))
    .child(div().absolute().left(0).right(0).bottom(GROUND - 1).h(2).bg(GRASS.edge));
  // Grass tufts at fixed spots, so redraws don't make the meadow jump.
  for (let i = 0; i < 28; i++) {
    const x = `${(i * 37) % 97 + 1.5}%`;
    root.child(div().absolute().left(x).bottom(GROUND - 1 + (i % 3)).w(2).h(5 + (i % 4)).rounded_full().bg(GRASS.tuft));
  }
  if (!rows.length) {
    return root.child(div().absolute().top(36).w_full().text_center().text_size(12).text_color(t.dim)
      .child("No agents yet. Each agent you start brings its partner here."));
  }
  if (s.width > 0) rows.forEach((row, i) => root.child(partner(s, row, homes[i], colW, names, t)));
  return root;
}
