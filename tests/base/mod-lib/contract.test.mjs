import { test } from 'node:test';
import assert from 'node:assert/strict';
import { snapshot, resolveLimits, resolveDaily } from '../../../mods/base/lib/contract.js';

const fresh = { windows: [{ minutes: 300, used: 20, resets_at: 200 }], updated: 200 };

test('snapshot accepts only objects with a numeric updated', () => {
  assert.equal(snapshot(fresh), fresh);
  for (const bad of [null, undefined, '', 42, [], {}, { plan: 'p' }, { updated: 'x' }]) assert.equal(snapshot(bad), null);
});

test('resolveLimits reads the neutral provider files only', () => {
  const r = resolveLimits({ claude: fresh, codex: { updated: 'bad' } });
  assert.equal(r.claude, fresh);
  assert.equal(r.codex, null);
});

test('legacy files are ignored (the old herdr-poke state is no longer read)', () => {
  assert.equal(resolveLimits({}, { claude: fresh, codex: fresh }).claude, null);
  assert.equal(resolveDaily(null, { days: [], updated: 9 }), null);
});

test('resolveDaily returns the neutral snapshot when valid', () => {
  const daily = { days: [{ date: 'a' }], updated: 5 };
  assert.equal(resolveDaily(daily), daily);
});

test('corrupt input returns null and never throws', () => {
  for (const bad of [null, undefined, '', 42, [], {}]) {
    assert.doesNotThrow(() => resolveLimits(bad));
    assert.doesNotThrow(() => resolveDaily(bad));
  }
  assert.deepEqual(resolveLimits(null), { claude: null, codex: null });
  assert.equal(resolveDaily('x'), null);
});
