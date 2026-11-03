// Reads the helper's state files (limits.json, usage.json) and keeps them fresh.
import { home, readTextFile, watchFolder } from "herdr";

export const STATE_DIR = () => `${home()}/.local/state/herdr-poke`;
export const PTB_DIR = () => `${home()}/Library/Application Support/PokeTokenBar`;

export async function readJSON(path) {
  try { const t = await readTextFile(path); return t ? JSON.parse(t) : null; } catch { return null; }
}

// Load now, then on every change in the folder, plus a slow timer in case watching fails.
export function keepFresh(s, cx, dir, load) {
  const again = (cx) => cx.spawn(async (cx) => { await load(s); cx.notify(); });
  again(cx);
  cx.timer.every(15000, again);
  cx.spawn(async (cx) => {
    for (;;) {
      const ok = await watchFolder(dir()).then(() => true, () => false);
      if (!ok) return;
      await load(s); cx.notify();
    }
  });
}

export async function loadUsage(s) {
  s.limits = (await readJSON(`${STATE_DIR()}/limits.json`)) ?? {};
  s.usage = (await readJSON(`${STATE_DIR()}/usage.json`)) ?? null;
}

export const WIN_LABEL = (m) => (m === 300 ? "Session" : m === 10080 ? "Weekly" : `${Math.round(m / 60)}h window`);
export const win = (prov, minutes) => prov?.windows?.find((w) => w.minutes === minutes) ?? null;
