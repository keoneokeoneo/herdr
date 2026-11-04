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
  if (mode == null || mode === 'off') return null; // unset (undefined or null) means no type
  if (mode === 'repo') return repoType || null;
  let h = seed >>> 0;
  for (const c of branch || '') h = (h * 31 + c.charCodeAt(0)) >>> 0;
  return TYPES[h % TYPES.length];
}

// One quiet line under an agent row: what it needs, or (for agents that report them) model, effort, context.
export function agentSubline(r, all) {
  if (r.reviewOf) {
    const tg = all.find((x) => x.address === r.reviewOf);
    return `↳ reviewing ${tg?.tabName ?? 'a session'}${r.findings != null ? ` · ${r.findings} found` : ''}`;
  }
  if (r.status === 'blocked') return r.title || 'Waiting for you';
  if (r.status === 'done') return 'New · not checked yet';
  const facts = [r.model, r.effort, r.ctx != null ? `ctx ${r.ctx}%` : null].filter(Boolean).join(' · ');
  return facts || [folderOf(r.cwd), r.branch].filter(Boolean).join(' · ');
}

const folderOf = (cwd) => (cwd ?? '').replace(/\/+$/, '').split('/').pop() || '';

// Tabs are named "1", "2"... by default. Then the session title says more; failing that a folder,
// unless it only repeats the space's name.
export function placeLabel(name, { title, cwd, space } = {}) {
  if (name && !/^\d+$/.test(name)) return name;
  const clean = (title ?? '').replace(/^[^\p{L}\p{N}]+/u, '').trim();
  if (clean) return clean;
  const f = folderOf(cwd);
  return f && f !== space ? f : name || '';
}
