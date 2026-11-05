import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFile, readdir } from 'node:fs/promises';
import { existsSync } from 'node:fs';

const root = new URL('../../../', import.meta.url);
const read = async (p) => readFile(new URL(p, root), 'utf8');
const json = async (p) => JSON.parse(await read(p));
const PERSONA = ['poke-companion', 'poke-dex', 'poke-host', 'poke-strip'];
const SHIPPED = ['workspace-chrome', 'changes', 'files'];

function walk(node, fn) {
  if (!node || typeof node !== 'object') return;
  fn(node);
  for (const v of Object.values(node)) {
    if (Array.isArray(v)) v.forEach((c) => walk(c, fn));
    else if (v && typeof v === 'object') walk(v, fn);
  }
}
const ids = (layout) => { const s = new Set(); walk(layout, (n) => { if (typeof n.id === 'string') s.add(n.id); }); return s; };

test('layouts share every non-persona block id', async () => {
  const base = ids(await json('layouts/base.json'));
  const poke = ids(await json('layouts/poke.json'));
  for (const id of base) assert.ok(poke.has(id), `poke layout lacks ${id}`);
  assert.deepEqual([...poke].filter((id) => !base.has(id)).sort(), PERSONA);
});

test('base layout names no poke mod, id or theme', async () => {
  assert.equal(/poke/i.test(await read('layouts/base.json')), false);
  assert.equal((await json('layouts/base.json')).theme, 'base');
});

test('obsidian layout uses the neutral theme', async () => {
  assert.equal((await json('layouts/obsidian.json')).theme, 'base');
});

test('layouts only reference mods that exist', async () => {
  for (const f of ['base', 'poke', 'obsidian']) {
    walk(await json(`layouts/${f}.json`), (n) => {
      if (typeof n.mod === 'string') assert.ok(SHIPPED.includes(n.mod) || existsSync(new URL(`mods/${n.mod}/main.js`, root)), `${f}: mod ${n.mod}`);
    });
  }
});

test('panel tabs declared by a layout exist as children of its panel body stack', async () => {
  for (const f of ['base', 'poke']) {
    const layout = await json(`layouts/${f}.json`);
    let tabs = null, stack = null;
    walk(layout, (n) => {
      if (n.content?.props?.view === 'paneltabs') tabs = n.content.props.tabs;
      if (n.id === 'panel-body') stack = n;
    });
    assert.ok(tabs?.length, `${f}: no tabs`);
    const children = new Set(stack.children.map((c) => c.id));
    for (const [id] of tabs) assert.ok(children.has(id), `${f}: tab ${id} has no block`);
    assert.equal(children.size, tabs.length, `${f}: every panel block has a tab`);
  }
});

test('keys: setup bindings live in layout sections, not global', async () => {
  const keys = await json('keys.json');
  for (const action of Object.values(keys.global)) {
    assert.equal(/^(base|poke)\./.test(action) || action === 'layout.toggle:panel', false, action);
  }
  for (const name of ['base', 'poke']) {
    const section = keys[`layout:${name}`];
    assert.equal(section['cmd-j'], 'base.jump');
    assert.equal(section['cmd-k'], 'layout.toggle:panel');
    assert.equal(section['cmd-shift-o'], 'base.notes');
  }
});

test('keys point to ids and commands that exist', async () => {
  const keys = await json('keys.json');
  const status = await read('mods/base/views/status.js');
  for (const name of ['base', 'poke']) {
    const have = ids(await json(`layouts/${name}.json`));
    for (const action of Object.values(keys[`layout:${name}`])) {
      const [, id] = action.match(/^layout\.toggle:(.+)$/) ?? [];
      if (id) assert.ok(have.has(id), `${name}: ${action}`);
      const [, cmd] = action.match(/^base\.(.+)$/) ?? [];
      if (cmd) assert.ok(status.includes(`"${cmd}"`), `status.js does not handle ${cmd}`);
    }
  }
});

test('shared keys outside the poke section never name Poke', async () => {
  const keys = await json('keys.json');
  for (const [section, map] of Object.entries(keys)) {
    if (section === 'layout:poke') continue;
    assert.equal(/poke/i.test(JSON.stringify(map)), false, section);
  }
});

test('base files reference Poke only through the documented seam', async () => {
  const ALLOWED = /poke\.(sprites|evening|typeMode|typeSeed|partner|review_of|findings|owner|provider|model|effort|ctx)|herdr-poke|poke-assets/g;
  const files = [];
  const dig = async (dir) => {
    for (const e of await readdir(new URL(dir, root), { withFileTypes: true })) {
      if (e.isDirectory()) { if (e.name !== 'fonts') await dig(`${dir}${e.name}/`); }
      else if (/\.(js|json)$/.test(e.name) && !/^(jsconfig|.*\.d)\./.test(e.name)) files.push(`${dir}${e.name}`);
    }
  };
  await dig('mods/base/');
  files.push('layouts/base.json', 'themes/base.json');
  for (const f of files) {
    const bad = (await read(f)).split('\n').filter((l) => !/^\s*(\/\/|\*)/.test(l))
      .map((l) => l.replace(ALLOWED, '')).filter((l) => /poke/i.test(l));
    assert.deepEqual(bad, [], f);
  }
});
