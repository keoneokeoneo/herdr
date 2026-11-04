import { test } from 'node:test';
import assert from 'node:assert/strict';
import { cryFor, retryDue } from '../../../mods/poke/lib/model.js';

test('cryFor plays the partner cry when its mp3 is on disk', () => {
  assert.equal(cryFor(36, ['36.gif', '36.mp3']), 'poke-assets/36.mp3');
});

test('cryFor falls back to Herdr blocked sound when the cry is missing', () => {
  assert.equal(cryFor(36, ['36.gif']), 'blocked');
  assert.equal(cryFor(null, []), 'blocked');
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

test('asset completeness requires a sprite, cry and name', async () => {
  const { assetsMissing } = await import('../../../mods/poke/lib/model.js');
  assert.equal(assetsMissing(7, ['7.gif'], {'7': 'Squirtle'}), true);
  assert.equal(assetsMissing(7, ['7.gif', '7.mp3'], {}), true);
  assert.equal(assetsMissing(7, ['7.png', '7.mp3'], {'7': 'Squirtle'}), false);
});
