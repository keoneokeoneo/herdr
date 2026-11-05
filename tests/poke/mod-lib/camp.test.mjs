import { test } from 'node:test';
import assert from 'node:assert/strict';
import { campGroups } from '../../../mods/poke/lib/camp.js';

const agent = (address, status, extra = {}) => ({ address, status, title: 'Session', space_name: 'Work', ...extra });
const terminal = (address, partner) => ({ address, labels: { 'poke.partner': partner } });

test('same species remains one visitor per session across spaces', () => {
  const g = campGroups([agent('t1', 'working'), agent('t2', 'idle', { space_name: 'Personal' })], [terminal('t1', '25'), terminal('t2', '25')]);
  assert.equal(g.working[0].address, 't1');
  assert.equal(g.idle[0].address, 't2');
  assert.equal(g.working[0].partner, 25);
  assert.equal(g.idle[0].partner, 25);
  assert.equal(g.idle[0].space, 'Personal');
});
test('missing and invalid partners preserve sessions as placeholders', () => {
  for (const value of [undefined, '', 'no', '0', '1026', '1.5']) {
    const g = campGroups([agent('t1', 'blocked')], [terminal('t1', value)]);
    assert.equal(g.blocked.length, 1);
    assert.equal(g.blocked[0].partner, null);
  }
  assert.equal(campGroups([agent('t1', 'idle')], []).idle[0].partner, null);
});
test('fresh snapshots move changed status and remove closed sessions', () => {
  const ts = [terminal('t1', '25'), terminal('t2', '4')];
  assert.equal(campGroups([agent('t1', 'working')], ts).working.length, 1);
  const next = campGroups([agent('t1', 'done')], ts);
  assert.equal(next.working.length, 0);
  assert.equal(next.done.length, 1);
  assert.equal(Object.values(campGroups([], ts)).flat().length, 0);
});
test('unrecognized status is explicit and names fall back to address', () => {
  const row = campGroups([agent('t3', 'other', { title: '', space_name: '' })], []).unknown[0];
  assert.equal(row.name, 't3');
  assert.equal(row.space, '');
  assert.equal(row.status, 'other');
  assert.equal(campGroups([agent('t1', 'idle', { tab_name: 'Build' })], []).idle[0].name, 'Build');
});

test('garden positions stay put across status changes and stay separated in crowds', async () => {
  const { gardenVisitors } = await import('../../../mods/poke/lib/camp.js');
  assert.equal(typeof gardenVisitors, 'function');
  const rows = Array.from({ length: 40 }, (_, i) => agent(`t${i}`, 'idle'));
  const before = gardenVisitors(campGroups(rows, []));
  const after = gardenVisitors(campGroups(rows.map((r) => ({ ...r, status: 'working' })), []));
  assert.deepEqual(before.map(({ address, x, y }) => ({ address, x, y })), after.map(({ address, x, y }) => ({ address, x, y })));
  assert.equal(new Set(before.map((r) => `${r.x},${r.y}`)).size, 40);
  assert.ok(before.every((r) => r.x >= 5 && r.x <= 80 && r.y >= 100));
});

test('default numbered tabs use readable session titles, custom names take precedence', () => {
  const rows = campGroups([
    agent('t1', 'working', { tab_name: '2', title: 'Fix login redirect' }),
    agent('t2', 'idle', { tab_name: 'Frontend', title: 'Claude Code' }),
    agent('t3', 'idle', { tab_name: '3', title: '' }),
  ], []);
  assert.equal(rows.working[0].name, 'Fix login redirect');
  assert.equal(rows.idle[0].name, 'Frontend');
  assert.equal(rows.idle[1].name, 't3');
});

test('status descriptions distinguish finished unread sessions from idle and unknown', async () => {
  const { campState } = await import('../../../mods/poke/lib/camp.js');
  assert.equal(typeof campState, 'function');
  assert.equal(campState('done').detail, 'Finished while you were away');
  assert.equal(campState('idle').detail, 'Waiting for the next task');
  assert.equal(campState('blocked').detail, 'Waiting for your input');
  assert.equal(campState('constructor').label, 'Unknown');
});

test('expanded garden stays compact on large displays and fits small windows', async () => {
  const { gardenWindowSize } = await import('../../../mods/poke/lib/camp.js');
  assert.equal(typeof gardenWindowSize, 'function');
  assert.deepEqual(gardenWindowSize({ width: 1920, height: 2130 }), { width: 1000, height: 700 });
  assert.deepEqual(gardenWindowSize({ width: 800, height: 600 }), { width: 704, height: 480 });
});

test('each visitor carries its tab change time from spaces()', () => {
  const spaces = [{ tabs: [{ id: 'tabA', changed_at: 123 }, { id: 'tabB', changed_at: null }] }];
  const g = campGroups([agent('t1', 'idle', { tab: 'tabA' }), agent('t2', 'idle', { tab: 'tabB' }), agent('t3', 'idle', { tab: 'gone' })], [], spaces);
  assert.deepEqual(g.idle.map((r) => r.changedAt), [123, null, null]);
});

test('a long-idle partner naps, a fresh idle one waits', async () => {
  const { campState } = await import('../../../mods/poke/lib/camp.js');
  assert.equal(campState('idle', true).label, 'Napping');
  assert.equal(campState('idle', true).mark, 'zZ');
  assert.equal(campState('idle', false).label, 'Idle');
  assert.equal(campState('working', true).label, 'Working', 'only idle partners nap');
});
