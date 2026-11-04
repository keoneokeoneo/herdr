import { test } from 'node:test';
import assert from 'node:assert/strict';
import { usageState } from '../../../mods/poke/lib/usage.js';

const now = new Date(2026, 10, 4, 0, 5).getTime();
const yesterday = { date: '2026-11-03', claude: 99e6, codex: 0 };
const today = { date: '2026-11-04', claude: 0, codex: 2e6 };

test('Today uses the local calendar date, not the last recorded day', () => {
  assert.equal(usageState({ days: [yesterday], updated: now / 1000 - 600 }, now).today, null);
  assert.equal(usageState({ days: [today, yesterday], updated: now / 1000 }, now).today, today);
});

test('fresh zero usage is data; a missing collection is unknown', () => {
  const zero = { ...today, codex: 0 };
  assert.equal(usageState({ days: [zero], updated: now / 1000 }, now).today, zero);
  assert.equal(usageState(null, now).today, null);
  assert.equal(usageState(null, now).updatedText, 'No usage data yet');
});

test('last-update text and stale status expose a stopped collector', () => {
  const fresh = usageState({ days: [today], updated: now / 1000 - 120 }, now);
  assert.equal(fresh.updatedText, 'Updated 2m ago');
  assert.equal(fresh.stale, false);
  const old = usageState({ days: [today], updated: now / 1000 - 25 * 60 }, now);
  assert.equal(old.updatedText, 'Updated 25m ago');
  assert.equal(old.stale, true);
  assert.equal(usageState({ days: [yesterday], updated: now / 1000 - 600 }, now).stale, true);
});

test('unknown and future update times do not pretend to be fresh', () => {
  for (const updated of [undefined, 0, 'bad', now / 1000 + 600]) {
    const s = usageState({ days: [today], updated }, now);
    assert.equal(s.stale, true);
    assert.equal(s.updatedText, 'Update time unavailable');
  }
});

test('panel and status do not relabel yesterday as today; historical chart remains', async () => {
  const { readFile } = await import('node:fs/promises');
  const { tokensM } = await import('../../../mods/poke/lib/format.js');
  const text = [];
  function element() {
    const el = new Proxy({}, { get: (_target, method) => (...args) => {
      if (method === 'child' && typeof args[0] === 'string') text.push(args[0]);
      return el;
    } });
    return el;
  }
  globalThis.__usageRender = {
    div: element, image: element, h_flex: element, v_flex: element,
    theme: () => ({}), props: () => ({ tab: 'usage' }), rows: () => [], waiting: () => [],
    win: () => null, live: () => false, pctOrDash: () => '–', WIN_LABEL: String,
    tokensM, usageState,
  };
  const originalNow = Date.now;
  Date.now = () => now;
  try {
    for (const name of ['panel', 'status']) {
      const source = (await readFile(new URL(`../../../mods/poke/views/${name}.js`, import.meta.url), 'utf8'))
        .replace(/^import .*;\n/gm, '');
      const prelude = `const {${Object.keys(globalThis.__usageRender).join(',')}}=globalThis.__usageRender;\n`;
      const view = await import('data:text/javascript;base64,' + Buffer.from(prelude + source).toString('base64'));
      text.length = 0;
      view.render({ usage: { days: [yesterday], updated: now / 1000 - 600 } });
      assert.ok(text.includes('Today'), name);
      assert.ok(!text.includes('99.0M'), `${name} must not show old total as today's total`);
      assert.ok(text.includes('Updated 10m ago'), name);
      if (name === 'panel') {
        assert.ok(text.includes('No usage recorded for today yet.'));
        assert.ok(text.includes('14 days'));
        assert.ok(text.includes('avg 99.0M/day'));
      }
    }
  } finally {
    Date.now = originalNow;
    delete globalThis.__usageRender;
  }
});
