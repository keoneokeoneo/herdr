export const URG = { blocked: 0, working: 1, done: 2, idle: 3 };
export const TYPES = ['grass', 'fire', 'water', 'electric', 'psychic', 'normal'];
const M = 60e3;

export const byUrgency = (list) => [...list].sort((a, b) => URG[a.status] - URG[b.status]);

export function groups(agents) {
  const g = { needs: [], working: [], done: [], idle: [] };
  for (const a of agents) (a.status === 'blocked' ? g.needs : g[a.status] || g.idle).push(a);
  return g;
}

export const waiting = (agents) => agents.filter((a) => a.status === 'blocked').sort((a, b) => a.since - b.since);

export function chips(a, nowMs) {
  const r = [];
  if (a.status === 'blocked') {
    const m = Math.floor((nowMs - a.since) / M);
    if (m >= 5) r.push({ text: `waiting ${m}m`, tip: 'Waiting for you over 5 min', color: 'electric' });
  }
  if (a.ctx != null && a.ctx >= 80) r.push({ text: `ctx ${a.ctx}%`, tip: 'Context almost full — try /compact', color: 'psychic' });
  return r;
}

export function tabPill(agents) {
  const n = { blocked: 0, working: 0, done: 0, idle: 0 };
  for (const a of agents) n[a.status] = (n[a.status] || 0) + 1;
  if (n.blocked) return { text: '! needs you', tone: 'blocked' };
  if (n.working) return { text: `● ${n.working} working`, tone: 'working' };
  if (n.done) return { text: '✓ new', tone: 'accent' };
  if (agents.length) return { text: 'idle', tone: 'dim' };
  return { text: '', tone: 'dim' };
}

export function typeOf(branch, seed, mode, repoType) {
  if (mode === 'off') return null;
  if (mode === 'repo') return repoType || null;
  let h = seed >>> 0;
  for (const c of branch || '') h = (h * 31 + c.charCodeAt(0)) >>> 0;
  return TYPES[h % TYPES.length];
}

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
