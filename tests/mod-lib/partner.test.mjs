import { test } from 'node:test';
import assert from 'node:assert/strict';
import { excluded, pickPartner } from '../../mods/poke/lib/partner.js';

test('excluded = dex chains + active line', () => {
  const s = excluded({ dex: [{ chainOrder: [1, 2, 3] }], active: { plannedPathIDs: [4, 5, 6] } });
  assert.deepEqual([...s].sort((a, b) => a - b), [1, 2, 3, 4, 5, 6]);
});
test('excluded tolerates missing fields', () => assert.equal(excluded(null).size, 0));
test('pickPartner never returns excluded, stays in 1..1025', () => {
  const ex = excluded({ dex: [{ chainOrder: [1, 2, 3] }], active: { plannedPathIDs: [4, 5, 6] } });
  for (let i = 0; i < 10000; i++) {
    const id = pickPartner(Math.random, ex);
    assert.ok(id >= 1 && id <= 1025 && !ex.has(id), String(id));
  }
});
test('pickPartner edges', () => {
  assert.equal(pickPartner(() => 0, new Set()), 1);
  assert.equal(pickPartner(() => 0.9999999, new Set()), 1025);
});
