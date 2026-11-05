import { test } from 'node:test';
import assert from 'node:assert/strict';
import { walkArt, walkMissing, moodOf, planStep, positionAt, pose, SLEEP_AFTER } from '../../../mods/poke/lib/walk.js';

const files = ['25.gif', 'walk-25-r.gif', 'walk-25-l.gif', 'walk-25-idle.gif', 'walk-25-sleep.gif', 'walk-999.none'];

test('walkArt finds a complete set, else null', () => {
  assert.deepEqual(walkArt(25, files), {
    r: 'poke-assets/walk-25-r.gif', l: 'poke-assets/walk-25-l.gif',
    idle: 'poke-assets/walk-25-idle.gif', sleep: 'poke-assets/walk-25-sleep.gif',
  });
  assert.equal(walkArt(25, ['walk-25-l.gif']), null);
  assert.equal(walkArt(null, files), null);
});

test('walkMissing: neither the set nor the none marker', () => {
  assert.equal(walkMissing(25, files), false);
  assert.equal(walkMissing(999, files), false);
  assert.equal(walkMissing(7, files), true);
});

test('moodOf: working walks, blocked and done stand, idle strolls then sleeps', () => {
  const now = 1e12;
  assert.equal(moodOf('working', now, now), 'walk');
  assert.equal(moodOf('blocked', now, now), 'stand');
  assert.equal(moodOf('done', now, now), 'stand');
  assert.equal(moodOf('idle', now - 1000, now), 'stroll');
  assert.equal(moodOf('idle', now - SLEEP_AFTER - 1, now), 'sleep');
  assert.equal(moodOf('idle', null, now), 'sleep'); // idle since before Herdr started
  assert.equal(moodOf('weird', now, now), 'stand');
});

const opts = { range: 60, speed: { walk: 70, stroll: 30 } };
const seq = (...xs) => { let i = 0; return () => xs[i++ % xs.length]; };

test('a walker picks a leg inside its range and faces the way it goes', () => {
  const p = planStep(null, 'walk', 0, seq(0.9, 0.5), opts);
  assert.equal(p.from, 0);
  assert.ok(p.to > 0 && p.to <= 60, `to ${p.to}`);
  assert.equal(p.face, 'r');
  assert.equal(p.dur, Math.round(Math.abs(p.to) / 70 * 1000));
  assert.ok(p.rest > p.start + p.dur, 'pauses after arriving');
  const back = planStep({ ...p }, 'walk', p.rest, seq(0.0, 0.5), opts);
  assert.ok(back.to < p.to);
  assert.equal(back.face, 'l');
  assert.equal(back.from, p.to);
});

test('legs are never shorter than a third of the range', () => {
  for (const r of [0, 0.3, 0.49, 0.5, 0.51, 0.7, 1]) {
    const p = planStep({ from: 0, to: 0, start: 0, dur: 0, rest: 0, face: 'r' }, 'walk', 10, seq(r, 0.5), opts);
    assert.ok(Math.abs(p.to - p.from) >= 20, `rand ${r}: ${p.from} → ${p.to}`);
    assert.ok(Math.abs(p.to) <= 60);
  }
});

test('a walker keeps its leg until it rests, then is unchanged until the rest ends', () => {
  const p = planStep(null, 'walk', 0, seq(0.9, 0.5), opts);
  assert.equal(planStep(p, 'walk', p.dur / 2, Math.random, opts), p);
  assert.equal(planStep(p, 'walk', p.rest - 1, Math.random, opts), p);
  assert.notEqual(planStep(p, 'walk', p.rest, seq(0.1, 0.5), opts), p);
});

test('strollers rest far longer than walkers', () => {
  const w = planStep(null, 'walk', 0, seq(0.9, 1), opts);
  const s = planStep(null, 'stroll', 0, seq(0.9, 1), opts);
  assert.ok(s.rest - s.start - s.dur > w.rest - w.start - w.dur);
});

test('stopping mid-walk freezes where it is', () => {
  const p = { from: 0, to: 60, start: 0, dur: 1000, rest: 2000, face: 'r' };
  assert.equal(positionAt(p, 500), 30);
  const s = planStep(p, 'stand', 500, Math.random, opts);
  assert.equal(s.from, 30);
  assert.equal(s.to, 30);
  assert.equal(s.dur, 0);
  assert.equal(planStep(s, 'sleep', 900, Math.random, opts), s, 'a still one stays put');
});

test('pose: moving shows the facing walk, else sleep or idle', () => {
  const p = { from: 0, to: 60, start: 0, dur: 1000, rest: 2000, face: 'l' };
  assert.equal(pose(p, 'walk', 500), 'l');
  assert.equal(pose(p, 'walk', 1500), 'idle');
  assert.equal(pose(p, 'sleep', 1500), 'sleep');
  assert.equal(pose(null, 'stand', 0), 'idle');
});
