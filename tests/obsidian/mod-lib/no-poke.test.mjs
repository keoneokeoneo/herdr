import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';

const read = (p) => readFile(new URL(`../../../${p}`, import.meta.url), 'utf8');

test('obsidian mod and layout have no Poke reference', async () => {
  for (const f of ['mods/obsidian/main.js', 'mods/obsidian/tree.js', 'layouts/obsidian.json']) {
    assert.equal(/poke/i.test(await read(f)), false, f);
  }
});

test('closing the embedded reader hides the neutral panel; Back leads to base', async () => {
  const src = await read('mods/obsidian/main.js');
  assert.ok(src.includes("run('layout.hide', { id: 'panel' })"));
  assert.ok(src.includes("run('layout.switch', { name: 'base' })"));
  assert.ok(src.includes("'Back'"));
});

test('the installer copies the neutral theme and does not patch Poke files', async () => {
  const sh = await read('scripts/obsidian/install.sh');
  assert.ok(sh.includes('themes/base.json'));
  assert.equal(/poke/i.test(sh), false);
});
