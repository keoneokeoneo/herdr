import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';

test('store ignores unrelated files and unchanged snapshots, but publishes changes', async () => {
  const source = (await readFile(new URL('../../../mods/poke/views/store.js', import.meta.url), 'utf8'))
    .replace('import { home, readTextFile, watchFolder } from "herdr";',
      'const { home, readTextFile, watchFolder } = globalThis.__pokeStoreTest;');
  let wake;
  globalThis.__pokeStoreTest = { home: () => '/home', readTextFile: async () => null,
    watchFolder: () => new Promise((resolve) => { wake = resolve; }) };
  const { keepFresh } = await import('data:text/javascript;base64,' + Buffer.from(source).toString('base64'));
  let notices = 0, loads = 0, value = 1;
  const timers = [];
  const cx = { notify: () => notices++, spawn: (fn) => fn(cx),
    timer: { every: (_ms, fn) => timers.push(fn) } };
  const settle = async () => { for (let i = 0; i < 8; i++) await Promise.resolve(); };
  keepFresh({}, cx, () => '/state', async (s) => { loads++; s.limits = {value}; });
  await settle();
  assert.equal(notices, 1);
  await timers[0](cx);
  assert.equal(notices, 1);
  const before = loads;
  wake(['/state/labels-t1.json']);
  await settle();
  assert.equal(loads, before);
  value = 2;
  wake(['/state/limits.json']);
  await settle();
  assert.equal(notices, 2);
  delete globalThis.__pokeStoreTest;
});

test('store also watches extra folders and reacts to the neutral usage files', async () => {
  const source = (await readFile(new URL('../../../mods/poke/views/store.js', import.meta.url), 'utf8'))
    .replace('import { home, readTextFile, watchFolder } from "herdr";',
      'const { home, readTextFile, watchFolder } = globalThis.__pokeStoreTest;');
  const wakes = {};
  globalThis.__pokeStoreTest = { home: () => '/home', readTextFile: async () => null,
    watchFolder: (dir) => new Promise((resolve) => { wakes[dir] = resolve; }) };
  const { keepFresh } = await import('data:text/javascript;base64,' + Buffer.from(source + '\n// extra').toString('base64'));
  let loads = 0, value = 1;
  const cx = { notify: () => {}, spawn: (fn) => fn(cx), timer: { every: () => {} } };
  const settle = async () => { for (let i = 0; i < 8; i++) await Promise.resolve(); };
  keepFresh({}, cx, () => '/state', async (s) => { loads++; s.limits = { value }; }, [() => '/usage']);
  await settle();
  const before = loads;
  value = 2;
  wakes['/usage'](['/usage/claude.json']);
  await settle();
  assert.equal(loads, before + 1);
  delete globalThis.__pokeStoreTest;
});
