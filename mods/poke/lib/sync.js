// When to re-sync from pokeTokenBar's folder, and how to avoid piling up runs.

const trim = (p) => p.replace(/\/+$/, "");

// True when a watchFolder batch touches what the companion/Dex sync reads:
// companion-state.json or anything under sprites/. A returned root path means
// "refresh everything". Snapshots and the usage cache change often and are ignored.
export function ptbChanged(paths, dir) {
  if (!paths?.length) return false;
  const d = trim(dir);
  return paths.some((raw) => {
    const p = trim(raw);
    return p === d || p === `${d}/companion-state.json` || p === `${d}/sprites` || p.startsWith(`${d}/sprites/`);
  });
}

// The companion card also reads today's snapshot and species details. These
// events should not run the heavier companion/Dex asset sync above.
export function companionChanged(paths, dir) {
  if (ptbChanged(paths, dir)) return true;
  if (!paths?.length) return false;
  const d = trim(dir);
  return paths.some((raw) => {
    const p = trim(raw);
    return p === `${d}/last-snapshot.json` || p === `${d}/pokemon-details-v1`
      || p.startsWith(`${d}/pokemon-details-v1/`);
  });
}

// Wraps an async job so concurrent triggers never overlap: a trigger that
// arrives mid-run asks for exactly one more run after the current one.
export function coalesce(job) {
  let running = null;
  let again = false;
  return function trigger() {
    if (running) { again = true; return running; }
    running = (async () => {
      do {
        again = false;
        try { await job(); } catch {}
      } while (again);
      running = null;
    })();
    return running;
  };
}
