import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { typeModeInit } from '../../../mods/poke/lib/defaults.js';

test('typeMode defaults to random only when unset', () => {
  assert.equal(typeModeInit(undefined), 'random');
  assert.equal(typeModeInit(null), 'random');
  assert.equal(typeModeInit('off'), null);
  assert.equal(typeModeInit('random'), null);
});

test('persona behaviours do not carry jump, notes or Codex attach (those live in base)', async () => {
  const src = await readFile(new URL('../../../mods/poke/behaviours.js', import.meta.url), 'utf8');
  for (const word of ['jump', 'attach', 'agent.focus', 'layout.show']) assert.equal(src.includes(word), false, word);
});

test('the poke mod does not read usage or limits', async () => {
  for (const f of ['behaviours.js', 'main.js', 'views/host.js', 'views/dex.js', 'views/companion.js']) {
    const src = await readFile(new URL(`../../../mods/poke/${f}`, import.meta.url), 'utf8');
    assert.equal(/limits\.json|usage\.json|daily\.json|herdr-app\/usage/.test(src), false, f);
  }
});
