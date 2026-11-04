// Shared drawing pieces: palette, pills, avatars.
import { div, image, svg } from "gpui-kit";
import { h_flex } from "gpui-base";
import { theme, shared } from "herdr";
import { SPRITE_DIR } from "./data.js";
import { spriteSrc, letterFor } from "../lib/avatar.js";

export const C = {
  grass: "#86c97e", fire: "#f2a072", water: "#82b4f0", electric: "#efcf6e", psychic: "#ee9cc4", normal: "#a9adbb",
};
// Provider colours follow their icons (Claude orange, Codex blue).
export const PROV = { claude: "#d97757", codex: "#8ec5f5" };
export const tone = (t, name) => (name === "accent" ? t.accent : t[name] ?? C[name] ?? t.dim);
export const alpha = (hex, aa) => (hex?.length === 7 ? hex + aa : hex);

export function pill(text, color, opts = {}) {
  return div().flex_none().whitespace_nowrap().text_size(opts.size ?? 11).font_weight(600)
    .px(8).py(1).rounded_full().bg(alpha(color, "26")).text_color(color).child(text);
}

export const SYMBOL = { working: "●", blocked: "!", done: "✓", idle: "‖" };

// Persona seam: sprite files are published by Poke as shared("poke.sprites"); absent means letter avatars.
export const spritePath = (id) => spriteSrc(shared("poke.sprites"), id, SPRITE_DIR);

const ICON = { claude: "icons/claude.svg", codex: "icons/codex.svg" };

// GIF frames only advance when the window redraws. A sub-pixel native loop keeps
// redrawing while the sprite is shown (no script timer; it stops when hidden).
export function alive(el, src) {
  if (!src?.endsWith(".gif")) return el;
  return div().relative().left(0).loop("left", { from: 0, to: 0.01, period: 1000 }).child(el);
}

export function avatar(r, opts = {}) {
  const t = theme();
  const size = opts.size ?? 30;
  const st = r.status;
  const ring = st === "blocked" ? t.blocked : st === "working" ? t.working : st === "done" ? t.done : t.idle;
  const src = spritePath(r.partner);
  let sprite = src
    // Sprites differ in how much of their square canvas they fill: fit the canvas, never crop it.
    ? alive(image(src).w(size).h(size).flex_none(), src)
    : div().text_size(size * 0.42).font_weight(700).text_color(ring).child(letterFor(r.provider));
  const eve = shared("poke.evening") === true;
  if (st === "idle") sprite = sprite.opacity(eve ? 0.35 : 0.45);
  else if (st === "done" && eve) sprite = sprite.opacity(0.7);
  let face = div().id(`av-${r.address}-${opts.key ?? ""}`).size(size).rounded_full()
    .flex().items_center().justify_center().bg(alpha(ring, "33")).child(sprite);
  if (st === "working" && !opts.still) face = face.relative().top(0).loop("top", { from: 0, to: -2, period: 1100 });
  const box = div().relative().flex_none().size(size).child(face);
  if (ICON[r.provider] && !opts.noBadge) {
    const b = Math.round(size * 0.46);
    box.child(div().absolute().right(-3).bottom(-3).size(b).rounded_full().bg("#101117")
      .flex().items_center().justify_center().child(image(ICON[r.provider]).w(b * 0.66).h(b * 0.66)));
  }
  if (st === "blocked") {
    box.child(div().absolute().top(-6).right(-5).px(4).rounded(6).bg(t.blocked).text_color("#3a1608")
      .text_size(9).font_weight(800).child("!").loop("top", { from: -6, to: -9, period: 900 }));
  } else if (st === "done") {
    box.child(div().absolute().top(-2).right(-2).size(8).rounded_full().bg(t.accent));
  }
  return box;
}
