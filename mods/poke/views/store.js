// Reads the helper's state files (limits.json, usage.json) and keeps them fresh.
import { home, readTextFile, watchFolder } from "herdr";

export const STATE_DIR = () => `${home()}/.local/state/herdr-poke`;
export const PTB_DIR = () => `${home()}/Library/Application Support/PokeTokenBar`;

export async function readJSON(path) {
  try { const t = await readTextFile(path); return t ? JSON.parse(t) : null; } catch { return null; }
}

// Load now, then on every change in the folder(s), plus a slow timer in case watching fails.
// `extra` lists more folders (functions) to watch, e.g. the usage data contract folder.
export function keepFresh(s, cx, dir, load, extra = []) {
  let busy = false;
  let previous;
  const refresh = async (cx) => {
    if (busy) return;
    busy = true;
    try {
      await load(s);
      const next = JSON.stringify([s.limits, s.usage, s.dex]);
      if (next !== previous) { previous = next; cx.notify(); }
    } finally { busy = false; }
  };
  const again = (cx) => cx.spawn(refresh);
  again(cx);
  cx.timer.every(15000, again);
  for (const d of [dir, ...extra]) {
    cx.spawn(async (cx) => {
      for (;;) {
        const paths = await watchFolder(d()).catch(() => null);
        if (!paths) return;
        if (paths.some((p) => p === d() || /(?:limits|usage|claude|codex|daily|companion-state)\.json$/.test(p))) await refresh(cx);
      }
    });
  }
}

export const WIN_LABEL = (m) => (m === 300 ? "Session" : m === 10080 ? "Weekly" : `${Math.round(m / 60)}h window`);
export const win = (prov, minutes) => prov?.windows?.find((w) => w.minutes === minutes) ?? null;
