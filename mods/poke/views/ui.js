// Poke's own drawing pieces: palette, pills, animated sprites.
import { div, image } from "gpui-kit";
import { theme, shared, layout } from "herdr";

export const C = {
  grass: "#86c97e", fire: "#f2a072", water: "#82b4f0", electric: "#efcf6e", psychic: "#ee9cc4", normal: "#a9adbb",
};
export const alpha = (hex, aa) => (hex?.length === 7 ? hex + aa : hex);

export function pill(text, color, opts = {}) {
  return div().flex_none().whitespace_nowrap().text_size(opts.size ?? 11).font_weight(600)
    .px(8).py(1).rounded_full().bg(alpha(color, "26")).text_color(color).child(text);
}

// Outside mods/: Herdr reloads a mod on any write inside its folder.
export const SPRITE_DIR = "poke-assets";

// Sprite files on disk, published by the persona host as shared("poke.sprites").
export function spritePath(id) {
  const have = shared("poke.sprites") ?? [];
  if (have.includes(`${id}.gif`)) return `${SPRITE_DIR}/${id}.gif`;
  if (have.includes(`${id}.png`)) return `${SPRITE_DIR}/${id}.png`;
  return null;
}

// GIF frames only advance when the window redraws. A sub-pixel native loop keeps
// redrawing while the sprite is shown (no script timer; it stops when hidden).
export function alive(el, src) {
  if (!src?.endsWith(".gif")) return el;
  return div().relative().left(0).loop("left", { from: 0, to: 0.01, period: 1000 }).child(el);
}

// A sprite for a species id (or asset key like "dex-6"), animated; null when not on disk.
export function sprite(key, size) {
  const src = spritePath(key);
  return src ? alive(image(src).w(size).h(size).flex_none(), src) : null;
}

function find(node, id) {
  if (!node || typeof node !== "object") return null;
  if (node.id === id) return node;
  for (const c of node.children ?? []) { const f = find(c, id); if (f) return f; }
  return null;
}
// True when the sidebar is folded to its rail.
export const narrow = () => (find(layout(), "sidebar")?.size?.px ?? 262) < 120;
