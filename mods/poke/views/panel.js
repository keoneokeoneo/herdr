// Right panel bodies: Usage (limits + 14 days) and Dex. Changes/Files are the shipped mods.
import { div, image } from "gpui-kit";
import { h_flex, v_flex } from "gpui-base";
import { theme, shared, setShared, props } from "herdr";
import { pill, spritePath, C, alpha } from "./ui.js";
import { keepFresh, loadUsage, readJSON, STATE_DIR, PTB_DIR, WIN_LABEL, win } from "./store.js";
import { pctOrDash, resetsIn, live, tokensM } from "../lib/format.js";


async function loadDex(s) {
  const st = await readJSON(`${PTB_DIR()}/companion-state.json`);
  s.dex = (st?.dex ?? []).map((d) => ({ id: d.finalID ?? d.chainOrder?.at(-1) ?? d.baseID, shiny: !!d.isShiny, name: d.names?.[String(d.finalID)]?.en ?? `#${d.finalID}` }));
}

export function init(s, cx) {
  keepFresh(s, cx, STATE_DIR, async (s) => { await loadUsage(s); await loadDex(s); });
}

const kv = (t, k, v, color) => h_flex().text_size(11).child(div().text_color(t.muted).child(k)).child(div().flex_1())
  .child(div().font_family("IBM Plex Mono").text_color(color ?? t.foreground).truncate().child(v));
const card = (t) => v_flex().gap(4).p(10).rounded(10).bg(t.surface).border_1().border_color(t.border);

function limitCard(t, name, prov, now, color, icon) {
  const c = card(t).child(h_flex().items_center().gap(6)
    .child(image(icon).w(14).h(14).flex_none())
    .child(div().text_size(12).font_weight(700).text_color(t.foreground).child(name))
    .child(div().flex_1())
    .child(div().text_size(10).text_color(t.dim).child(prov?.plan ?? "")));
  for (const m of [300, 10080]) {
    const w = win(prov, m);
    const ok = live(w, now);
    const pct = ok ? Math.min(100, w.used) : 0;
    const barC = pct >= 80 ? t.blocked : pct >= 60 ? t.accent : color;
    c.child(v_flex().gap(3).pt(4)
      .child(h_flex().text_size(11)
        .child(div().text_color(t.muted).child(WIN_LABEL(m)))
        .child(div().flex_1())
        .child(div().font_weight(600).text_color(ok ? t.foreground : t.dim).child(pctOrDash(w, now))))
      .child(div().w_full().h(5).rounded_full().bg(t.raised).overflow_hidden().child(div().h(5).rounded_full().bg(barC).w(`${Math.max(ok ? 2 : 0, Math.round(pct))}%`)))
      .child(div().text_size(10).text_color(t.dim).child(ok ? `resets in ${resetsIn(w.resets_at, now)}` : w ? "no current data" : prov ? `not on ${prov.plan || "this plan"}` : "no data yet")));
  }
  return c;
}

function usage(t, s) {
  const now = Date.now();
  const L = s.limits ?? {};
  const box = v_flex().gap(10)
    .child(limitCard(t, "Claude", L.claude, now, t.done, "icons/claude.svg"))
    .child(limitCard(t, "Codex", L.codex, now, "#8ec5f5", "icons/codex.svg"));
  const days = s.usage?.days ?? [];
  const today = days.at(-1);
  const tc = card(t).child(div().text_size(12).font_weight(700).text_color(t.foreground).child("Today"));
  if (!today) return box.child(tc.child(div().text_size(10).text_color(t.dim).child("Run herdr-poke usage to collect.")));
  tc.child(kv(t, "Claude", tokensM(today.claude), t.done)).child(kv(t, "Codex", tokensM(today.codex), "#8ec5f5"));
  const max = Math.max(1, ...days.map((d) => d.claude + d.codex));
  const avg = days.reduce((a, d) => a + d.claude + d.codex, 0) / Math.max(1, days.length);
  const bars = h_flex().h(60).items_end().gap(3).pt(6);
  for (const d of days) {
    const H = 56;
    const cl = Math.round((d.claude / max) * H), cx = Math.round((d.codex / max) * H);
    bars.child(v_flex().flex_1().justify_end().h(H)
      .child(div().w_full().h(cx).bg("#8ec5f5").rounded_t(2))
      .child(div().w_full().h(cl).bg(t.done)));
  }
  tc.child(bars).child(h_flex().text_size(10).text_color(t.dim).child(div().child("14 days")).child(div().flex_1()).child(div().child(`avg ${tokensM(avg)}/day`)));
  return box.child(tc);
}

function dex(t, s) {
  const caught = s.dex ?? [];
  const seen = shared("poke.seen") ?? [];
  const grid = (ids) => {
    const g = h_flex().flex_wrap().gap(6);
    for (const it of ids) {
      const src = spritePath(it.id);
      g.child(div().size(40).rounded(8).bg(t.surface).flex().items_center().justify_center()
        .child(src ? image(src).w(36).h(36) : div().text_size(9).text_color(t.dim).child(`#${it.id}`)));
    }
    return g;
  };
  const box = v_flex().gap(10)
    .child(div().text_size(12).font_weight(700).text_color(t.foreground).child(`Caught · ${caught.length}`));
  if (caught.length) {
    const list = v_flex().gap(4);
    for (const c of caught) list.child(h_flex().gap(6).text_size(11).child(div().text_color(t.foreground).child(c.name)).child(c.shiny ? div().text_color(t.accent).child("✦") : div()));
    box.child(list);
  } else box.child(div().text_size(11).text_color(t.dim).child("Nothing caught yet. Keep feeding your partner."));
  box.child(div().pt(6).text_size(12).font_weight(700).text_color(t.foreground).child(`Seen · ${seen.length}`));
  box.child(seen.length ? grid(seen.map((id) => ({ id }))) : div().text_size(11).text_color(t.dim).child("Session partners you meet show up here."));
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
  // Each copy is one child of the poke-panel-body stack (props.tab); paneltabs picks which shows.
  const body = props()?.tab === "dex" ? dex(t, s) : usage(t, s);
  return v_flex().size_full().gap(12).px(12).pt(4).pb(12).bg(t.chrome).border_l_1().border_color(t.border).overflow_y_scroll()
    .child(body);
}
