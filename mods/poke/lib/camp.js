import { MAX_SPECIES } from './partner.js';

export function gardenWindowSize(viewport) {
  return { width: Math.min(1000, viewport.width * 0.88), height: Math.min(700, viewport.height * 0.8) };
}

// A fresh snapshot: visitors are sessions, not deduplicated species.
export function campGroups(agents, terminals, spaces = []) {
  const groups = { blocked: [], working: [], done: [], idle: [], unknown: [] };
  const byAddress = new Map(terminals.map((t) => [t.address, t]));
  const changed = new Map(spaces.flatMap((s) => s.tabs ?? []).map((t) => [t.id, t.changed_at ?? null]));
  for (const a of agents) {
    const raw = byAddress.get(a.address)?.labels?.['poke.partner'];
    const id = Number(raw);
    const partner = raw != null && raw !== '' && Number.isInteger(id) && id >= 1 && id <= MAX_SPECIES ? id : null;
    const key = Object.hasOwn(groups, a.status) ? a.status : 'unknown';
    groups[key].push({ address: a.address, partner, status: a.status,
      name: a.tab_name && !/^\d+$/.test(a.tab_name) ? a.tab_name : a.title || a.address, space: a.space_name || '',
      changedAt: changed.get(a.tab) ?? null });
  }
  return groups;
}

// Sort by identity, never by status: activity changes don't reshuffle the garden.
export function gardenVisitors(groups) {
  return Object.values(groups).flat().sort((a, b) => a.address.localeCompare(b.address))
    .map((row, i) => ({ ...row, x: [8, 29, 50, 71][i % 4] + (Math.floor(i / 4) % 2 ? 4 : 0),
      y: 130 + Math.floor(i / 4) * 200 + [14, 0, 25, 8][i % 4] }));
}

const CAMP_STATES = {
  working: { label: 'Working', detail: 'Task in progress', mark: '···', tone: 'water' },
  blocked: { label: 'Needs you', detail: 'Waiting for your input', mark: '!', tone: 'electric' },
  done: { label: 'Done', detail: 'Finished while you were away', mark: '✓', tone: 'grass' },
  idle: { label: 'Idle', detail: 'Waiting for the next task', mark: 'z', tone: 'normal' },
  unknown: { label: 'Unknown', detail: 'Status unavailable', mark: '?', tone: 'psychic' },
};
const NAPPING = { label: 'Napping', detail: 'Idle for a while', mark: 'zZ', tone: 'normal' };
// `asleep`: an idle partner whose tab has been still long enough to lie down (lib/walk.js moodOf).
export function campState(status, asleep = false) {
  if (status === 'idle' && asleep) return NAPPING;
  return CAMP_STATES[Object.hasOwn(CAMP_STATES, status) ? status : 'unknown'];
}
