// Per-id retry with doubling backoff (1 min → 30 min). Records the attempt when due.
export function retryDue(tried, id, nowMs) {
  const t = tried.get(id);
  if (t && nowMs < t.at + t.wait) return false;
  tried.set(id, { at: nowMs, wait: t ? Math.min(30 * 60_000, t.wait * 2) : 60_000 });
  return true;
}

// A partner is complete with a sprite (animated or still) and a name.
export const assetsMissing = (id, files, names) =>
  !(files.includes(`${id}.gif`) || files.includes(`${id}.png`)) || !names?.[id];
