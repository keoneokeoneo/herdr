import { test } from 'node:test';
import assert from 'node:assert/strict';
import { tabsFrom, stackOf } from '../../../mods/base/lib/panel.js';
import { spriteSrc, letterFor } from '../../../mods/base/lib/avatar.js';

test('tabsFrom reads the layout-declared tabs and drops malformed entries', () => {
  const p = { tabs: [['usage', 'Usage'], ['files', 'Files'], 'bad', ['x'], [1, 'n']] };
  assert.deepEqual(tabsFrom(p), [['usage', 'Usage'], ['files', 'Files']]);
  assert.deepEqual(tabsFrom(undefined), []);
  assert.equal(stackOf({}), 'panel-body');
  assert.equal(stackOf({ stack: 's' }), 's');
});

test('avatar falls back to a letter when no sprite files are published (no Poke running)', () => {
  assert.equal(spriteSrc(undefined, 5, 'poke-assets'), null);
  assert.equal(spriteSrc([], 5, 'poke-assets'), null);
  assert.equal(spriteSrc(['5.png'], 5, 'poke-assets'), 'poke-assets/5.png');
  assert.equal(spriteSrc(['5.gif', '5.png'], 5, 'poke-assets'), 'poke-assets/5.gif');
  assert.equal(letterFor('codex'), 'X');
  assert.equal(letterFor('claude'), 'C');
  assert.equal(letterFor(undefined), 'C');
});
