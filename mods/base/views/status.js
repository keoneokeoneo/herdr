// Status bar: both providers' limits, today's tokens, unread done, jump to waiting.
import { div, image } from "gpui-kit";
import { h_flex } from "gpui-base";
import { theme, run, layout, notify } from "herdr";
import { rows } from "./data.js";
import { keepFresh, WIN_LABEL } from "./store.js";
import { loadUsage, USAGE_DIR } from "./usagefiles.js";
import { pctOrDash, tokensM, live } from "../lib/format.js";
import { usageShown } from "../lib/panel.js";
import { usageState, level, limitAlerts } from "../lib/usage.js";
import { waiting, agentSummary, nextDone } from "../lib/model.js";
import { start, jump, notes } from "../behaviours.js";

export function init(s, cx) {
  keepFresh(s, cx, USAGE_DIR, loadUsage);
  start(s, cx);
  cx.timer.every(60000, (cx) => cx.notify());
}

// Limit alerts, once per window (keys carry resets_at). Checked on every render, which follows each
// usage change and the 60 s tick; keys are marked first so a re-render never posts twice.
function alertLimits(s, cx) {
  const fresh = limitAlerts(s.limits, Date.now(), (s.alerted ??= []));
  if (!fresh.length) return;
  s.alerted.push(...fresh.map((a) => a.key));
  cx.spawn(async () => {
    for (const a of fresh) notify({ title: a.title, body: "Open the Usage tab for the forecast.", level: a.level, icon: `icons/${a.provider}.svg`, timeout: 10000, action: { name: "layout.show", args: { id: "panel" } } });
  });
}

// Key commands (base.jump, base.notes) reach shown copies; only this always-shown copy acts on them.
export function command(s, c) {
  if (c === "jump") jump(s);
  else if (c === "review") { const d = nextDone(rows()); if (d) run("agent.focus", { terminal: d.address }); }
  else if (c === "notes") notes();
}

const sep = (t) => div().w(1).h(14).bg(t.border);

// Each window coloured by how close it is to the limit (same thresholds as the panel bars).
function prov(t, name, icon, p, now) {
  const el = h_flex().gap(6).items_center()
    .child(image(icon).w(13).h(13).flex_none())
    .child(div().text_color(t.muted).child(name));
  const ws = (p?.windows ?? []).filter((w) => w.minutes === 300 || w.minutes === 10080);
  if (!ws.length) return el.child(div().text_color(t.dim).child("–"));
  ws.forEach((w, i) => {
    const lv = live(w, now) ? level(w.used) : null;
    const c = lv === "limit" ? t.blocked : lv === "high" ? t.accent : t.foreground;
    if (i) el.child(div().text_color(t.dim).child("·"));
    el.child(div().text_color(t.muted).child(WIN_LABEL(w.minutes)))
      .child(div().font_weight(lv ? 600 : 400).text_color(c).child(`${lv === "limit" ? "⚠ " : ""}${pctOrDash(w, now)}`));
  });
  return el;
}

export function render(s, cx) {
  alertLimits(s, cx);
  const t = theme();
  const now = Date.now();
  const L = s.limits ?? {};
  const all = rows();
  const wait = waiting(all).length;
  const fresh = all.filter((r) => r.status === "done").length;
  const { today, updatedText, stale } = usageState(s.usage, now);
  const bar = h_flex().size_full().items_center().gap(12).px(12).bg(t.chrome).border_t_1().border_color(t.border).text_size(11);
  // While the panel shows Usage, the limits are right there; the bar keeps only today's total.
  if (!usageShown(layout())) bar.child(prov(t, "Claude", "icons/claude.svg", L.claude, now)).child(sep(t))
    .child(prov(t, "Codex", "icons/codex.svg", L.codex, now)).child(sep(t));
  bar.child(h_flex().gap(6).child(div().text_color(t.muted).child("Today"))
      .child(div().text_color(t.foreground).child(today ? tokensM(today.claude + today.codex) : "–")));
  if (stale) bar.child(div().text_size(10).text_color(t.dim).child(updatedText));
  // Agent counts keep the bar useful when the sidebar is folded or the limits are in the panel.
  const sum = agentSummary(all);
  if (sum) bar.child(sep(t)).child(div().text_color(wait ? t.blocked : fresh ? t.accent : t.muted).child(sum));
  bar.child(div().flex_1());
  // Finished agents: open the oldest one; viewing it clears its "new" mark.
  const done = nextDone(all);
  if (done && !wait) {
    bar.child(div().id("review").flex_none().px(10).py(2).rounded_full().bg(t.accent + "26").text_color(t.accent).font_weight(600).cursor_pointer()
      .tooltip("Show the oldest finished agent (⌘⇧J)")
      .on_click(() => run("agent.focus", { terminal: done.address }))
      .child(`Review new · ${fresh}  ⌘⇧J`));
  }
  if (wait) {
    bar.child(div().id("jump").flex_none().px(10).py(2).rounded_full().bg(t.blocked + "26").text_color(t.blocked).font_weight(600).cursor_pointer()
      .tooltip("Show the agent that has waited longest (⌘J)")
      .on_click(() => jump(s))
      .child(`Jump to waiting · ${wait}  ⌘J`));
  }
  bar.child(div().id("palette").flex_none().text_color(t.dim).cursor_pointer().hover((x) => x.text_color(t.foreground))
    .tooltip("Command palette").on_click(() => run("app.palette")).child("⌘⇧P"));
  return bar;
}
