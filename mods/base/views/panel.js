// Right panel body: Usage (limits + 14 days). Changes/Files are the shipped mods.
import { div, image } from "gpui-kit";
import { h_flex, v_flex } from "gpui-base";
import { theme } from "herdr";
import { keepFresh, WIN_LABEL, win } from "./store.js";
import { loadUsage, USAGE_DIR } from "./usagefiles.js";
import { pctOrDash, resetsIn, live, tokensM } from "../lib/format.js";
import { usageState } from "../lib/usage.js";

export function init(s, cx) {
  cx.timer.every(60000, (cx) => cx.notify());
  keepFresh(s, cx, USAGE_DIR, loadUsage);
}

const kv = (t, k, v, color) => h_flex().text_size(11).child(div().text_color(t.muted).child(k)).child(div().flex_1())
  .child(div().font_family("IBM Plex Mono").text_color(color ?? t.foreground).truncate().child(v));
const card = (t) => v_flex().gap(4).p(10).rounded(10).bg(t.surface).border_1().border_color(t.border);

function limitCard(t, name, prov, now, color, icon) {
  const c = card(t).child(h_flex().items_center().gap(6)
    .child(image(icon).w(14).h(14).flex_none())
    .child(div().text_size(12).font_weight(700).text_color(t.foreground).child(name)));
  for (const m of [300, 10080]) {
    const w = win(prov, m);
    // A window the provider never reports (Codex on some plans has no 5h limit) is left out.
    if (prov && !w) continue;
    const ok = live(w, now);
    const pct = ok ? Math.min(100, w.used) : 0;
    const barC = pct >= 80 ? t.blocked : pct >= 60 ? t.accent : color;
    c.child(v_flex().gap(3).pt(4)
      .child(h_flex().text_size(11)
        .child(div().text_color(t.muted).child(WIN_LABEL(m)))
        .child(div().flex_1())
        .child(div().font_weight(600).text_color(ok ? t.foreground : t.dim).child(pctOrDash(w, now))))
      .child(div().w_full().h(5).rounded_full().bg(t.raised).overflow_hidden().child(div().h(5).rounded_full().bg(barC).w(`${Math.max(ok ? 2 : 0, Math.round(pct))}%`)))
      .child(div().text_size(10).text_color(t.dim).child(ok ? `resets in ${resetsIn(w.resets_at, now)}` : w ? "no current data" : "no data yet")));
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
  const { today, updatedText, stale } = usageState(s.usage, now);
  const tc = card(t).child(div().text_size(12).font_weight(700).text_color(t.foreground).child("Today"));
  if (today) tc.child(kv(t, "Claude", tokensM(today.claude), t.done)).child(kv(t, "Codex", tokensM(today.codex), "#8ec5f5"));
  else tc.child(div().text_size(10).text_color(t.dim).child("No usage recorded for today yet."));
  tc.child(div().text_size(10).text_color(stale ? t.muted : t.dim).child(updatedText));
  if (!days.length) return box.child(tc);
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

export function render(s, cx) {
  const t = theme();
  return v_flex().size_full().gap(12).px(12).pt(4).pb(12).bg(t.chrome).border_l_1().border_color(t.border).overflow_y_scroll()
    .child(usage(t, s));
}
