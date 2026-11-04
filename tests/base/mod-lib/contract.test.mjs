import { test } from 'node:test';
import assert from 'node:assert/strict';
import { pickFresher, resolveLimits, resolveDaily } from '../../../mods/base/lib/contract.js';

const old = { windows: [{ minutes: 300, used: 10, resets_at: 100 }], updated: 100 };
const fresh = { windows: [{ minutes: 300, used: 20, resets_at: 200 }], updated: 200 };

test('pickFresher prefers the newer updated', () => {
  assert.equal(pickFresher(old, fresh), fresh);
  assert.equal(pickFresher(fresh, old), fresh);
});

test('pickFresher ignores non-objects and snapshots without updated', () => {
  assert.equal(pickFresher(null, old), old);
  assert.equal(pickFresher(old, 'x'), old);
  assert.equal(pickFresher({ plan: 'p' }, old), old);
  assert.equal(pickFresher({ plan: 'p' }, null), null);
});

test('resolveLimits uses legacy only when neutral is missing or older', () => {
  const r = resolveLimits({ claude: fresh }, { claude: old, codex: old });
  assert.equal(r.claude, fresh);
  assert.equal(r.codex, old);
  assert.equal(resolveLimits({ claude: old }, { claude: fresh }).claude, fresh);
});

test('resolveDaily prefers neutral when not older', () => {
  const n = { days: [{ date: 'a' }], updated: 5 };
  const l = { days: [{ date: 'b' }], updated: 5 };
  assert.equal(resolveDaily(n, l), n);
  assert.equal(resolveDaily(null, l), l);
  assert.equal(resolveDaily({ days: [], updated: 4 }, l), l);
});

test('corrupt input returns null and never throws', () => {
  for (const bad of [null, undefined, '', 42, [], {}]) {
    assert.doesNotThrow(() => resolveLimits(bad, bad));
    assert.doesNotThrow(() => resolveDaily(bad, bad));
  }
  assert.deepEqual(resolveLimits(null, null), { claude: null, codex: null });
  assert.equal(resolveDaily('x', 42), null);
});
