import { test } from 'node:test';
import assert from 'node:assert/strict';
import { ptbChanged, companionChanged, coalesce } from '../../mods/poke/lib/sync.js';

const D = '/Users/k/Library/Application Support/PokeTokenBar';

test('ptbChanged reacts to companion-state.json and sprites/', () => {
  assert.equal(ptbChanged([`${D}/companion-state.json`], D), true);
  assert.equal(ptbChanged([`${D}/sprites/6-a.gif`], D), true);
  assert.equal(ptbChanged([`${D}/sprites`], D), true);
});

test('ptbChanged ignores snapshot, usage cache, backups and details', () => {
  assert.equal(ptbChanged([`${D}/last-snapshot.json`, `${D}/usage-cache.json`], D), false);
  assert.equal(ptbChanged([`${D}/companion-state.backup-x.json`], D), false);
  assert.equal(ptbChanged([`${D}/pokemon-details-v1/6.json`], D), false);
  assert.equal(ptbChanged([`${D}/spritesheet.png`], D), false);
});

test('ptbChanged treats the root path (refresh everything) as a change', () => {
  assert.equal(ptbChanged([D], D), true);
  assert.equal(ptbChanged([`${D}/`], D), true);
});

test('ptbChanged with nothing is no change', () => {
  assert.equal(ptbChanged([], D), false);
  assert.equal(ptbChanged(null, D), false);
});

test('coalesce runs once more after a trigger that arrives mid-run, not once per trigger', async () => {
  let runs = 0;
  let release;
  const gate = new Promise((r) => { release = r; });
  const go = coalesce(async () => { runs++; if (runs === 1) await gate; });
  const first = go();
  go(); go(); go();
  release();
  await first;
  assert.equal(runs, 2);
  await go();
  assert.equal(runs, 3);
});

test('coalesce keeps working after the job throws', async () => {
  let runs = 0;
  const go = coalesce(async () => { runs++; throw new Error('x'); });
  await go();
  await go();
  assert.equal(runs, 2);
});


test('companionChanged includes snapshots, details and egg sprites without waking for unrelated files', () => {
  for (const file of ['companion-state.json', 'last-snapshot.json', 'pokemon-details-v1', 'pokemon-details-v1/1.json', 'sprites/egg.png'])
    assert.equal(companionChanged([`${D}/${file}`], D), true, file);
  assert.equal(companionChanged([D], D), true);
  assert.equal(companionChanged([`${D}/usage-cache.json`, `${D}/last-snapshot.backup.json`], D), false);
  assert.equal(companionChanged(null, D), false);
});
