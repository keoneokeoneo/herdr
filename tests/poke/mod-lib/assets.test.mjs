import { test } from 'node:test';
import assert from 'node:assert/strict';
import { retryDue, assetsMissing } from '../../../mods/poke/lib/assets.js';
import * as lib from '../../../mods/poke/lib/assets.js';

test('Poke has no cries', () => {
  assert.equal(lib.cryFor, undefined);
});

test('retryDue backs off per id: 1m, 2m, 4m … capped at 30m', () => {
  const tried = new Map();
  assert.equal(retryDue(tried, 7, 0), true);
  assert.equal(retryDue(tried, 7, 30_000), false);
  assert.equal(retryDue(tried, 7, 60_000), true);
  assert.equal(retryDue(tried, 7, 60_000 + 119_000), false);
  assert.equal(retryDue(tried, 7, 60_000 + 120_000), true);
  for (let i = 0; i < 20; i++) retryDue(tried, 8, 1e12 + i * 31 * 60_000);
  assert.ok(tried.get(8).wait <= 30 * 60_000);
});

test('asset completeness requires a sprite and a name, not a cry', () => {
  assert.equal(assetsMissing(7, ['7.gif'], {}), true);
  assert.equal(assetsMissing(7, [], { '7': 'Squirtle' }), true);
  assert.equal(assetsMissing(7, ['7.gif'], { '7': 'Squirtle' }), false);
  assert.equal(assetsMissing(7, ['7.png'], { '7': 'Squirtle' }), false);
});
