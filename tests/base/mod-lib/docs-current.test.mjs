import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFile, readdir } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import path from 'node:path';

const root = new URL('../../../', import.meta.url);
const read = (p) => readFile(new URL(p, root), 'utf8');
// Dated specs and plans are history and keep their original wording; these are the living docs.
const LIVING = async () => ['README.md', 'docs/guide.md', ...(await readdir(new URL('docs/modes/', root))).map((f) => `docs/modes/${f}`)];
const manifest = async () => JSON.parse(await read('scripts/setups.json'));

test('README lists every setup with its layouts, mods and themes', async () => {
  const readme = await read('README.md');
  for (const [name, s] of Object.entries(await manifest())) {
    assert.ok(readme.includes(`**${name}**`) || new RegExp(`\\|\\s*${name}\\s*\\|`, 'i').test(readme), `setup ${name}`);
    for (const l of s.layouts) assert.ok(readme.includes(`layouts/${l}.json`), `layouts/${l}.json`);
    for (const m of s.mods) assert.ok(readme.includes(`mods/${m}/`), `mods/${m}/`);
    for (const t of s.themes) assert.ok(readme.includes(`themes/${t}.json`), `themes/${t}.json`);
  }
});

test('every setup has a mode guide and the README links to it', async () => {
  const readme = await read('README.md');
  for (const name of Object.keys(await manifest())) {
    assert.ok(existsSync(new URL(`docs/modes/${name}.md`, root)), `docs/modes/${name}.md`);
    assert.ok(readme.includes(`docs/modes/${name}.md`), `README links docs/modes/${name}.md`);
  }
});

test('relative markdown links in living docs resolve', async () => {
  for (const f of await LIVING()) {
    const text = await read(f);
    for (const [, target] of text.matchAll(/\]\(([^)#\s]+)(?:#[^)]*)?\)/g)) {
      if (/^[a-z]+:/i.test(target)) continue;
      const resolved = new URL(target, new URL(f, root));
      assert.ok(existsSync(resolved), `${f}: ${target}`);
    }
  }
});

test('repo paths named in code spans exist', async () => {
  for (const f of await LIVING()) {
    const text = await read(f);
    for (const [, p] of text.matchAll(/`((?:mods|layouts|themes|scripts|helpers|tests|docs)\/[^`\s]*)`/g)) {
      if (/[<*{]/.test(p) || p.includes('...')) continue;
      assert.ok(existsSync(new URL(p, root)), `${f}: ${p}`);
    }
  }
});

test('living docs do not mention removed names', async () => {
  const banned = /poke-panel|poke-usage|poke-obsidian|poke\.jump|poke\.notes|herdr-poke statusline|docs\/superpowers|limits\.json|usage\.json|Back to Poke/;
  for (const f of await LIVING()) assert.equal(banned.test(await read(f)), false, `${f}: ${banned.exec(await read(f))?.[0]}`);
});

test('docs state the setups that need Base and the commands that install them', async () => {
  const readme = await read('README.md');
  assert.ok(/setup:install/.test(readme) && /setup:list/.test(readme));
  assert.ok(/requires/i.test(readme));
});
