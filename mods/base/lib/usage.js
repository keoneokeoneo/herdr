// daily.json uses local calendar dates and is normally collected every 10 minutes.
import { ago } from './format.js';

const STALE_AFTER = 20 * 60_000;

export function usageState(snapshot, nowMs) {
  const now = new Date(nowMs);
  const date = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`;
  const today = snapshot?.days?.find((day) => day.date === date) ?? null;
  const updated = snapshot?.updated;
  const age = nowMs - updated * 1000;
  const known = Number.isFinite(updated) && updated > 0 && age >= 0;
  return {
    today,
    stale: !today || !known || age >= STALE_AFTER,
    updatedText: !snapshot ? 'No usage data yet' : !known ? 'Update time unavailable'
      : age < 60_000 ? 'Updated just now' : `Updated ${ago(age)} ago`,
  };
}

const md = (date) => { const [, m, d] = date.split('-'); return `${Number(m)}/${Number(d)}`; };

// Average over days that have any usage (empty days before the first log would drag it down).
export function dailySummary(days) {
  const used = (days ?? []).filter((d) => d.claude + d.codex > 0);
  const total = used.reduce((a, d) => a + d.claude + d.codex, 0);
  return {
    avg: used.length ? total / used.length : 0,
    active: used.length,
    from: days?.length ? md(days[0].date) : null,
    to: days?.length ? md(days[days.length - 1].date) : null,
  };
}

// At the current rate, the share used when the window resets, and seconds until 100% (null if not before reset).
// Too early in the window the rate is noise, and in its last 10% the forecast says nothing new.
export function pace(w, minutes, nowMs) {
  const now = nowMs / 1000;
  if (!w || !(w.resets_at > now)) return null;
  const span = minutes * 60, left = w.resets_at - now, elapsed = span - left;
  if (elapsed < span * 0.1 || left < span * 0.1) return null; // too early to tell, or about to reset anyway
  const rate = w.used / elapsed;
  const atReset = Math.round(w.used + rate * left);
  const hit = rate > 0 && w.used < 100 ? (100 - w.used) / rate : null;
  return { atReset, hitsInSec: hit != null && hit < left ? Math.round(hit) : null };
}

// How close a window is to its limit; views colour the number, not the bar (the bar keeps the provider colour).
export const level = (used) => (used >= 80 ? 'limit' : used >= 60 ? 'high' : null);

const NAMES = { claude: 'Claude', codex: 'Codex' };
const WIN = { 300: 'session', 10080: 'weekly' };

// Alerts not yet shown (keys in `seen`): a live window at 80%+, or one that will hit 100% before it resets.
// Keys carry resets_at, so each new window can alert again.
export function limitAlerts(limits, nowMs, seen) {
  const out = [];
  for (const [prov, p] of Object.entries(limits ?? {})) {
    for (const w of p?.windows ?? []) {
      if (!WIN[w.minutes] || !(w.resets_at * 1000 > nowMs)) continue;
      const base = `${prov}:${w.minutes}:${w.resets_at}`;
      const name = `${NAMES[prov] ?? prov} ${WIN[w.minutes]}`;
      if (w.used >= 80) {
        if (!seen.includes(`${base}:80`)) out.push({ key: `${base}:80`, provider: prov, title: `${name} at ${Math.round(w.used)}%`, level: 'warning' });
        continue;
      }
      const p2 = pace(w, w.minutes, nowMs);
      if (p2?.hitsInSec != null && !seen.includes(`${base}:pace`)) {
        out.push({ key: `${base}:pace`, provider: prov, title: `${name} may hit its limit before it resets`, level: 'warning' });
      }
    }
  }
  return out;
}

// Why a provider card has less than both windows: no writer yet, or a plan without a 5h limit.
export function limitNote(provider, p) {
  if (!p) return provider === 'claude' ? 'Claude limits come from the Claude statusline' : `${NAMES[provider] ?? provider} limits need the herdr-poke helper`;
  return (p.windows ?? []).some((w) => w.minutes === 300) ? null : 'No 5h limit on this plan';
}
