// How a Garden partner moves: pure planning, no drawing. Each partner wanders
// around its own spot (an offset in px within ±range), so the garden layout
// stays put and only the walk is animated (natively, with transition("left")).

export const SLEEP_AFTER = 3 * 60_000; // idle this long (tab unchanged): lies down

// The walking set cut by the helper (`herdr-poke assets --walk <id>`), or null.
export function walkArt(id, files) {
  if (id == null) return null;
  const have = new Set(files || []);
  const art = {};
  for (const k of ['r', 'l', 'idle', 'sleep']) {
    if (!have.has(`walk-${id}-${k}.gif`)) return null;
    art[k] = `poke-assets/walk-${id}-${k}.gif`;
  }
  return art;
}

// True until the helper made the set or marked the species as having none.
export const walkMissing = (id, files) =>
  !(files || []).includes(`walk-${id}-r.gif`) && !(files || []).includes(`walk-${id}.none`);

// working → walk; idle → stroll, then sleep once its tab has been still for a while
// (null changed_at: nothing changed since Herdr started, so idle all along);
// blocked, done and anything else stand still.
export function moodOf(status, changedAt, now) {
  if (status === 'working') return 'walk';
  if (status === 'idle') return changedAt == null || now - changedAt > SLEEP_AFTER ? 'sleep' : 'stroll';
  return 'stand';
}

const moving = (p, now) => p != null && now < p.start + p.dur;

export function positionAt(p, now) {
  if (!p) return 0;
  if (p.dur <= 0 || now >= p.start + p.dur) return p.to;
  if (now <= p.start) return p.from;
  return Math.round(p.from + (p.to - p.from) * ((now - p.start) / p.dur));
}

// The next plan, or the same object when nothing changes (so callers can skip a redraw).
// opts: { range, speed: { walk, stroll } } in px and px/s.
export function planStep(p, mood, now, rand, opts) {
  if (mood !== 'walk' && mood !== 'stroll') {
    if (p && !moving(p, now) && p.dur === 0) return p;
    const at = positionAt(p, now);
    return { from: at, to: at, start: now, dur: 0, rest: now, face: p?.face ?? 'r' };
  }
  if (p && now < p.rest) return p;
  const from = positionAt(p, now);
  const { range } = opts;
  const min = range / 3;
  // Pick a spot in [-range, range] at least `min` away, on the side the coin says
  // (or the other side when there's no room left on that one).
  let dir = rand() < 0.5 ? -1 : 1;
  const room = (d) => (d > 0 ? range - from : from + range);
  if (room(dir) < min) dir = -dir;
  const span = Math.max(min, room(dir));
  const to = Math.round(from + dir * (min + (span - min) * rand()));
  const clamped = Math.max(-range, Math.min(range, to));
  const dur = Math.round((Math.abs(clamped - from) / opts.speed[mood]) * 1000);
  const pause = mood === 'walk' ? 300 + 1500 * rand() : 4000 + 8000 * rand();
  return { from, to: clamped, start: now, dur, rest: now + dur + Math.round(pause), face: clamped >= from ? 'r' : 'l' };
}

// Which of the four GIFs to show.
export function pose(p, mood, now) {
  if (moving(p, now)) return p.face;
  return mood === 'sleep' ? 'sleep' : 'idle';
}
