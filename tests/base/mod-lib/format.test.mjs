import { test } from 'node:test';
import assert from 'node:assert/strict';
import { ago, resetsIn, pctOrDash, tokensM } from '../../../mods/base/lib/format.js';

test('ago', () => { assert.equal(ago(43e3), '43s'); assert.equal(ago(6 * 60e3), '6m'); assert.equal(ago(65 * 60e3), '1h 5m'); });
test('resetsIn', () => {
  const now = 1_000_000_000_000;
  assert.equal(resetsIn(now / 1000 + (4 * 60 + 28) * 60, now), '4h 28m');
  assert.equal(resetsIn(now / 1000 + (5 * 24 + 2) * 3600, now), '5d 2h');
  assert.equal(resetsIn(now / 1000 - 1, now), '–');
});
test('pctOrDash', () => {
  const now = 1_000_000_000_000;
  assert.equal(pctOrDash({ used: 7.4, resets_at: now / 1000 + 60 }, now), '7%');
  assert.equal(pctOrDash(undefined, now), '–');
  assert.equal(pctOrDash({ used: 50, resets_at: now / 1000 - 1 }, now), '–');
});
test('tokensM', () => { assert.equal(tokensM(100_400_000), '100.4M'); assert.equal(tokensM(1_500_000_000), '1.5B'); });
