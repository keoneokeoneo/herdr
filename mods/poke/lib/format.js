export function ago(ms) {
  const s = Math.max(0, Math.round(ms / 1000));
  if (s < 60) return `${s}s`;
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m`;
  return `${Math.floor(m / 60)}h ${m % 60}m`;
}

export function resetsIn(epochSec, nowMs) {
  const s = Math.round(epochSec - nowMs / 1000);
  if (!(s > 0)) return '–';
  const d = Math.floor(s / 86400), h = Math.floor((s % 86400) / 3600), m = Math.floor((s % 3600) / 60);
  if (d) return `${d}d ${h}h`;
  if (h) return `${h}h ${m}m`;
  return `${m}m`;
}

// A window is {used, resets_at}; missing or already reset means we don't know.
export const live = (w, nowMs) => !!w && w.resets_at * 1000 > nowMs;
export const pctOrDash = (w, nowMs) => (live(w, nowMs) ? `${Math.round(w.used)}%` : '–');

export function tokensM(n) {
  if (n >= 1e9) return `${(n / 1e9).toFixed(1)}B`;
  return `${(n / 1e6).toFixed(1)}M`;
}
