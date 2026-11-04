import { test } from 'node:test';
import assert from 'node:assert/strict';
import { GRAD, stageCost, progress } from '../../../mods/poke/lib/companion.js';

test('rare 3-form stage 1 costs 1.0B', () => assert.equal(stageCost('rare', 3, 1), 1e9));
test('difficulty scales cost', () => assert.equal(stageCost('rare', 3, 1, 0.5), 5e8));
test('graduation totals', () => assert.deepEqual(GRAD, { common: 750e6, uncommon: 1875e6, rare: 3e9, legendary: 6e9 }));
test('charmeleon progress', () => {
  const p = progress({ rarity: 'rare', totalForms: 3, stageIndex: 1, usedAtStage: 708705343, plannedPathIDs: [4, 5, 6] }, 1);
  assert.ok(Math.abs(p.pct - 70.87) < 0.01, String(p.pct));
  assert.equal(p.nextId, 6);
  assert.equal(p.toNext, 1e9 - 708705343);
});
test('final stage has no next', () => {
  const p = progress({ rarity: 'rare', totalForms: 3, stageIndex: 2, usedAtStage: 1, plannedPathIDs: [4, 5, 6] }, 1);
  assert.equal(p.nextId, null);
});
