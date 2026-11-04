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
// Too early in the window (under 10% elapsed) the rate is noise, so there is no forecast.
export function pace(w, minutes, nowMs) {
  const now = nowMs / 1000;
  if (!w || !(w.resets_at > now)) return null;
  const span = minutes * 60, left = w.resets_at - now, elapsed = span - left;
  if (elapsed < span * 0.1) return null;
  const rate = w.used / elapsed;
  const atReset = Math.round(w.used + rate * left);
  const hit = rate > 0 && w.used < 100 ? (100 - w.used) / rate : null;
  return { atReset, hitsInSec: hit != null && hit < left ? Math.round(hit) : null };
}
