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
