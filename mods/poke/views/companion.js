// Sidebar: the pokeTokenBar partner Keno is raising right now (read-only).
import { div, image } from "gpui-kit";
import { h_flex, v_flex } from "gpui-base";
import { theme, home, appHome, readTextFile, watchFolder, exec, shared } from "herdr";
import { pill, spritePath, alive, C } from "./ui.js";
import { progress } from "../lib/companion.js";
import { tokensM } from "../lib/format.js";
import { narrow } from "./brand.js";

const DIR = () => `${home()}/Library/Application Support/PokeTokenBar`;
const cap = (s) => (s ? s.split("-").map((w) => w[0].toUpperCase() + w.slice(1)).join(" ") : "");

async function json(path) {
  try { const t = await readTextFile(path); return t ? JSON.parse(t) : null; } catch { return null; }
}

async function load(s) {
  const st = await json(`${DIR()}/companion-state.json`);
  if (!st) { s.missing = true; return; }
  s.missing = false;
  if (!st.active) {
    // Between partners pokeTokenBar keeps an egg: active is null until it hatches.
    if (!s.eggImg) {
      const dst = `${appHome()}/poke-assets`;
      const r = await exec("sh", ["-c", `mkdir -p "$1" && { [ -f "$1/companion-egg.png" ] || cp -p "$2" "$1/companion-egg.png"; }`, "sh", dst, `${DIR()}/sprites/egg.png`]).catch(() => null);
      s.eggImg = r?.code === 0;
    }
    const snap = await json(`${DIR()}/last-snapshot.json`);
    s.c = null;
    s.egg = { fed: st.eggUsage ?? 0, today: snap?.todayTotalTokens ?? null, dex: st.dex?.length ?? 0 };
    return;
  }
  s.egg = null;
  const a = st.active;
  const id = a.pathIDs?.[a.pathIDs.length - 1] ?? a.baseID;
  const p = progress(a, s.difficulty ?? 1);
  const det = await json(`${DIR()}/pokemon-details-v1/${id}.json`);
  const next = p.nextId ? await json(`${DIR()}/pokemon-details-v1/${p.nextId}.json`) : null;
  const snap = await json(`${DIR()}/last-snapshot.json`);
  s.c = {
    id, shiny: !!a.isShiny, level: a.profile?.level, nature: a.nature, ability: a.profile?.abilityName,
    name: cap(det?.details?.name) || `#${id}`, types: det?.details?.types ?? [],
    next: p.nextId ? cap(next?.details?.name) || `#${p.nextId}` : null, pct: p.pct, toNext: p.toNext,
    fed: snap?.todayTotalTokens ?? null, dex: st.dex?.length ?? 0,
  };
}

export function init(s, cx) {
  s.difficulty = 1;
  cx.timer.every(30000, (cx) => cx.spawn(async (cx) => { await load(s); cx.notify(); }));
  cx.spawn(async (cx) => {
    const r = await exec("defaults", ["read", "io.github.chattymin.poketokenbar", "growthDifficulty"]).catch(() => null);
    const d = Number(r?.stdout?.trim());
    if (r?.code === 0 && d > 0) s.difficulty = d;
    await load(s); cx.notify();
    // A rejected watch (folder missing, watch refused) must not spin: fall back to the timer.
    for (;;) {
      const ok = await watchFolder(DIR()).then(() => true, () => false);
      if (!ok) return;
      await load(s); cx.notify();
    }
  });
}

const TYPE_C = { fire: C.fire, water: C.water, grass: C.grass, electric: C.electric, psychic: C.psychic, fairy: C.psychic };

export function render(s) {
  const t = theme();
  const card = v_flex().size_full().justify_center().gap(6).px(12).py(10).bg(t.chrome).border_t_1().border_color(t.border);
  if (s.missing) return card.child(div().text_size(11).text_color(t.dim).child("pokeTokenBar not found"));
  if (s.egg) return egg(t, card, s);
  const c = s.c;
  if (!c) return card;
  const src = spritePath(`companion-${c.id}`) ?? spritePath(c.id);
  if (narrow()) return card.px(0).items_center().gap(4)
    .child(src ? alive(image(src).w(52).h(52)) : div().text_color(t.dim).child(`#${c.id}`))
    .child(div().text_size(10).text_color(t.muted).child(`Lv.${c.level ?? "?"}`))
    .child(div().w(48).h(4).rounded_full().bg(t.raised).overflow_hidden().child(div().h(4).rounded_full().bg(t.accent).w(`${Math.max(2, Math.round(c.pct))}%`)));
  const face = div().flex_none().size(64).rounded(12).bg(t.surface).flex().items_center().justify_center()
    .child(src ? alive(image(src).w(60).h(60)) : div().text_color(t.dim).child(`#${c.id}`));
  const title = h_flex().items_center().gap(6)
    .child(div().text_size(14).font_weight(700).text_color(t.foreground).child(c.name))
    .child(c.shiny ? div().text_color(t.accent).child("✦") : div())
    .child(div().flex_1())
    .child(div().text_size(11).text_color(t.muted).child(`Lv.${c.level ?? "?"}`));
  const types = h_flex().gap(4);
  for (const ty of c.types) types.child(pill(cap(ty), TYPE_C[ty] ?? C.normal, { size: 9 }));
  const info = v_flex().flex_1().min_w(0).gap(3).child(title)
    .child(div().truncate().text_size(10).text_color(t.dim).child(`${cap(c.nature)} · ${cap(c.ability)}`))
    .child(types);
  const fill = Math.max(2, Math.round(c.pct));
  const meter = div().w_full().h(6).rounded_full().bg(t.raised).overflow_hidden()
    .child(div().h(6).rounded_full().bg(t.accent).w(`${fill}%`));
  const seen = (shared("poke.seen") ?? []).length;
  return card
    .child(h_flex().gap(10).items_center().child(face).child(info))
    .child(h_flex().text_size(10).text_color(t.muted)
      .child(div().child(c.next ? `Next: ${c.next}` : "Final form · to graduation"))
      .child(div().flex_1())
      .child(div().child(`${Math.floor(c.pct)}% · ${tokensM(c.toNext)} to go`)))
    .child(meter)
    .child(h_flex().text_size(10).text_color(t.dim)
      .child(div().child(c.fed != null ? `Fed today ${tokensM(c.fed)}` : ""))
      .child(div().flex_1())
      .child(div().child(`Dex ${c.dex} · Seen ${seen}`)));
}

function egg(t, card, s) {
  const e = s.egg;
  if (narrow()) return card.px(0).items_center()
    .child(s.eggImg ? image("poke-assets/companion-egg.png").w(44).h(44) : div().text_size(22).text_color(t.dim).child("◓"));
  const face = div().flex_none().size(64).rounded(12).bg(t.surface).flex().items_center().justify_center()
    .child(s.eggImg
      ? image("poke-assets/companion-egg.png").w(56).h(56)
      : div().text_size(22).text_color(t.dim).child("◓"));
  const info = v_flex().flex_1().min_w(0).gap(3)
    .child(div().text_size(14).font_weight(700).text_color(t.foreground).child("Egg"))
    .child(div().text_size(10).text_color(t.dim).child("Something is about to hatch…"))
    .child(e.fed > 0 ? div().text_size(10).text_color(t.muted).child(`Warmed ${tokensM(e.fed)} so far`) : div());
  const seen = (shared("poke.seen") ?? []).length;
  return card
    .child(h_flex().gap(10).items_center().child(face).child(info))
    .child(h_flex().pt(4).text_size(10).text_color(t.dim)
      .child(div().child(e.today != null ? `Fed today ${tokensM(e.today)}` : ""))
      .child(div().flex_1())
      .child(div().child(`Dex ${e.dex} · Seen ${seen}`)));
}
