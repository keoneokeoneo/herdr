// The partner strip: one Poke partner per agent session along the bottom of the
// window. Pure data and planning; views/strip.js draws it.
import { MAX_SPECIES } from './partner.js';
import { walkArt, moodOf, planStep, pose } from './walk.js';

// Rows sorted by address, never by status, so activity changes don't reshuffle the strip.
export function stripRows(agents, terminals, spaces = []) {
  const byAddress = new Map(terminals.map((t) => [t.address, t]));
  const changed = new Map(spaces.flatMap((s) => s.tabs ?? []).map((t) => [t.id, t.changed_at ?? null]));
  return agents.map((a) => {
    const raw = byAddress.get(a.address)?.labels?.['poke.partner'];
    const id = Number(raw);
    const partner = raw != null && raw !== '' && Number.isInteger(id) && id >= 1 && id <= MAX_SPECIES ? id : null;
    return {
      address: a.address, pane: a.pane || null, partner, status: a.status,
      name: a.tab_name && !/^\d+$/.test(a.tab_name) ? a.tab_name : a.title || a.address,
      space: a.space_name || '', changedAt: changed.get(a.tab) ?? null,
    };
  }).sort((a, b) => a.address.localeCompare(b.address));
}

// n columns of at most `cap` px, centred in `width`; homes are column centres.
export function slots(n, width, cap) {
  if (n <= 0) return { width: cap, homes: [] };
  const w = Math.min(cap, Math.floor(width / n));
  const left = Math.round((width - w * n) / 2);
  return { width: w, homes: Array.from({ length: n }, (_, i) => left + w * i + Math.round(w / 2)) };
}

const STATES = {
  working: { label: 'Working', detail: 'Task in progress', mark: '···', tone: 'water' },
  blocked: { label: 'Needs you', detail: 'Waiting for your input', mark: '!', tone: 'electric' },
  done: { label: 'Done', detail: 'Finished while you were away', mark: '✓', tone: 'grass' },
  idle: { label: 'Idle', detail: 'Waiting for the next task', mark: '', tone: 'normal' },
  unknown: { label: 'Unknown', detail: 'Status unavailable', mark: '?', tone: 'psychic' },
};
const NAPPING = { label: 'Napping', detail: 'Idle for a while', mark: 'zZ', tone: 'normal' };

// `asleep`: an idle partner whose tab has been still long enough to lie down (walk.js moodOf).
export function partnerState(status, asleep = false) {
  if (status === 'idle' && asleep) return NAPPING;
  return STATES[Object.hasOwn(STATES, status) ? status : 'unknown'];
}

// Plans every partner's walk and says when the strip must be redrawn: only when a
// pose, mood, planned spot or the set of sessions changes (the walk itself is native).
// opts: { range: () => px, speed: { walk, stroll } px/s, rand }.
export function stripTicker(opts) {
  const walk = new Map();
  let last = '';
  return {
    get: (address) => walk.get(address),
    step(rows, files, now) {
      const live = new Set();
      let key = '';
      for (const row of rows) {
        live.add(row.address);
        const mood = moodOf(row.status, row.changedAt, now);
        const prev = walk.get(row.address);
        const plan = walkArt(row.partner, files)
          ? planStep(prev?.plan ?? null, mood, now, opts.rand ?? Math.random, { range: opts.range(), speed: opts.speed })
          : null;
        walk.set(row.address, { plan, mood });
        key += `${row.address}:${row.status}:${mood}:${plan ? `${pose(plan, mood, now)}@${plan.to}` : '-'};`;
      }
      for (const k of walk.keys()) if (!live.has(k)) walk.delete(k);
      if (key === last) return false;
      last = key;
      return true;
    },
  };
}
