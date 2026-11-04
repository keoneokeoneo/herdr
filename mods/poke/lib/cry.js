// The sound for a session that starts waiting: its partner's cry, else Herdr's own.
export const cryFor = (partner, sprites) =>
  partner != null && (sprites || []).includes(`${partner}.mp3`) ? `poke-assets/${partner}.mp3` : 'blocked';

// Per-id retry with doubling backoff (1 min → 30 min). Records the attempt when due.
export function retryDue(tried, id, nowMs) {
  const t = tried.get(id);
  if (t && nowMs < t.at + t.wait) return false;
  tried.set(id, { at: nowMs, wait: t ? Math.min(30 * 60_000, t.wait * 2) : 60_000 });
  return true;
}

export const assetsMissing = (id, files, names) =>
  !(files.includes(`${id}.gif`) || files.includes(`${id}.png`)) || !files.includes(`${id}.mp3`) || !names?.[id];
