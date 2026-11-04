import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { progress } from '../../mods/poke/lib/companion.js';
import * as sync from '../../mods/poke/lib/sync.js';

const D = '/home/Library/Application Support/PokeTokenBar';
const settle = async () => { for (let i = 0; i < 80; i++) await Promise.resolve(); };
let serial = 0;

async function fixture() {
  const st = { active: { baseID: 1, rarity: 'common', totalForms: 1, stageIndex: 0,
    usedAtStage: 375e6, plannedPathIDs: [1], profile: { level: 20 } }, dex: [] };
  const f = { st, notices: 0, reads: 0, defaultsReads: 0, result: { code: 0, stdout: '1' }, timers: [], stateReads: 0 };
  let wake;
  const env = {
    home: () => '/home', appHome: () => '/app', C: {}, progress, ...sync,
    readTextFile: async (p) => {
      f.reads++;
      if (p.endsWith('/companion-state.json')) {
        f.stateReads++;
        if (f.gate) { const gate = f.gate; f.gate = null; await gate; }
        return JSON.stringify(f.st);
      }
      if (p.endsWith('/last-snapshot.json')) return JSON.stringify({ todayTotalTokens: f.today ?? 100 });
      return JSON.stringify({ details: { name: 'bulbasaur', types: ['grass'] } });
    },
    exec: async () => { f.defaultsReads++; if (f.result instanceof Error) throw f.result; return f.result; },
    watchFolder: () => new Promise((resolve) => { wake = resolve; }),
  };
  globalThis.__pokeCompanionTest = env;
  const src = (await readFile(new URL('../../mods/poke/views/companion.js', import.meta.url), 'utf8'))
    .replace(/^import .*;\n/gm, '');
  const prelude = `const {home,appHome,C,progress,readTextFile,exec,watchFolder,coalesce,companionChanged}=globalThis.__pokeCompanionTest;`;
  const m = await import('data:text/javascript;base64,' + Buffer.from(prelude + src + `\n// fixture ${serial++}`).toString('base64'));
  const cx = { notify: () => f.notices++, spawn: (fn) => fn(cx), timer: { every: (_ms, fn) => f.timers.push(fn) } };
  f.s = {};
  m.init(f.s, cx);
  await settle();
  f.tick = async () => { f.timers[0](cx); await settle(); };
  f.event = async (paths) => { wake(paths); await settle(); };
  return f;
}

test('companion refreshes changed difficulty on the timer and keeps last value on failure', async () => {
  const f = await fixture();
  assert.equal(f.s.c.pct, 50);
  f.result = { code: 0, stdout: '2' };
  await f.tick();
  assert.equal(f.s.c.pct, 25);
  f.result = new Error('defaults unavailable');
  await f.tick();
  assert.equal(f.s.c.pct, 25);
  f.result = { code: 0, stdout: 'Infinity' };
  await f.tick();
  assert.equal(f.s.c.pct, 25);
});

test('deleted difficulty preference resets to the default, unrelated errors do not', async () => {
  const f = await fixture();
  f.result = { code: 0, stdout: '2' };
  await f.tick();
  f.result = { code: 1, stderr: 'Failed to read preferences: permission denied' };
  await f.tick();
  assert.equal(f.s.difficulty, 2);
  f.result = { code: 1, stderr: 'The domain/default pair of (io.github.chattymin.poketokenbar, growthDifficulty) does not exist' };
  await f.tick();
  assert.equal(f.s.difficulty, 1);
  assert.equal(f.s.c.pct, 50);
});

test('unrelated PTB events do not read or redraw, unchanged updates do not redraw', async () => {
  const f = await fixture();
  assert.equal(f.notices, 1);
  const before = f.reads;
  await f.event([`${D}/usage-cache.json`, `${D}/companion-state.backup.json`]);
  assert.equal(f.reads, before);
  assert.equal(f.notices, 1);
  await f.tick();
  assert.equal(f.notices, 1);
  f.today = 200;
  await f.event([`${D}/last-snapshot.json`]);
  assert.equal(f.notices, 2);
  assert.equal(f.s.c.fed, 200);
});

test('timer and watcher loads are serialized and a mid-load update is not lost', async () => {
  const f = await fixture();
  let release;
  f.gate = new Promise((r) => { release = r; });
  const before = f.stateReads;
  await f.tick();
  await f.event([`${D}/companion-state.json`]);
  await f.tick();
  assert.equal(f.stateReads, before + 1);
  f.st.active.profile.level = 21;
  release();
  await settle();
  assert.equal(f.stateReads, before + 2);
  assert.equal(f.s.c.level, 21);
  assert.equal(f.notices, 2);
});
