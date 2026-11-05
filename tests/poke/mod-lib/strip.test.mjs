import { test } from 'node:test';
import assert from 'node:assert/strict';
import { stripRows, slots, partnerState, stripTicker } from '../../../mods/poke/lib/strip.js';

const agent = (address, status, extra = {}) => ({ address, status, title: 'Session', space_name: 'Work', pane: `p-${address}`, ...extra });
const terminal = (address, partner) => ({ address, labels: { 'poke.partner': partner } });

test('one row per session, with its partner, pane and tab change time, in a stable order', () => {
  const spaces = [{ tabs: [{ id: 'tabA', changed_at: 123 }] }];
  const rows = stripRows([agent('t2', 'idle', { tab: 'tabA' }), agent('t1', 'working')], [terminal('t1', '25'), terminal('t2', '25')], spaces);
  assert.deepEqual(rows.map((r) => [r.address, r.partner, r.pane, r.changedAt]), [['t1', 25, 'p-t1', null], ['t2', 25, 'p-t2', 123]]);
});

test('order is by identity, so a status change never reshuffles the strip', () => {
  const a = stripRows([agent('t1', 'idle'), agent('t2', 'working')], []);
  const b = stripRows([agent('t1', 'blocked'), agent('t2', 'done')], []);
  assert.deepEqual(a.map((r) => r.address), b.map((r) => r.address));
});

test('bad or missing partner labels become null', () => {
  for (const value of [undefined, '', 'x', '0', '1.5', '999999']) {
    assert.equal(stripRows([agent('t1', 'idle')], [terminal('t1', value)])[0].partner, null, String(value));
  }
});

test('numbered tabs use the session title; named tabs keep their name', () => {
  const rows = stripRows([
    agent('t1', 'working', { tab_name: '2', title: 'Fix login' }),
    agent('t2', 'idle', { tab_name: 'Frontend' }),
    agent('t3', 'idle', { tab_name: '3', title: '' }),
  ], []);
  assert.deepEqual(rows.map((r) => r.name), ['Fix login', 'Frontend', 't3']);
});

test('slots spread partners over the width, never wider than a cap', () => {
  assert.deepEqual(slots(2, 1000, 200), { width: 200, homes: [400, 600] });
  const crowd = slots(10, 600, 200);
  assert.equal(crowd.width, 60);
  assert.equal(crowd.homes[0], 30);
  assert.equal(crowd.homes[9], 570);
  assert.deepEqual(slots(0, 600, 200).homes, []);
});

test('partnerState marks: working, needs you, done, idle, napping', () => {
  assert.equal(partnerState('working').mark, '···');
  assert.equal(partnerState('blocked').mark, '!');
  assert.equal(partnerState('done').mark, '✓');
  assert.equal(partnerState('idle').mark, '');
  assert.equal(partnerState('idle', true).mark, 'zZ');
  assert.equal(partnerState('idle', true).label, 'Napping');
  assert.equal(partnerState('working', true).label, 'Working');
  assert.equal(partnerState('constructor').label, 'Unknown');
});

test('the ticker only reports a change when a pose, mood, spot or the cast changes', () => {
  const tick = stripTicker({ range: () => 40, speed: { walk: 60, stroll: 26 }, rand: () => 0.5 });
  const files = ['walk-25-r.gif', 'walk-25-l.gif', 'walk-25-idle.gif', 'walk-25-sleep.gif'];
  const rows = [{ address: 't1', partner: 25, status: 'blocked', changedAt: 0 }];
  assert.equal(tick.step(rows, files, 0), true, 'first frame');
  assert.equal(tick.step(rows, files, 250), false, 'standing still');
  assert.equal(tick.step([...rows, { address: 't2', partner: null, status: 'idle', changedAt: 0 }], files, 500), true, 'a newcomer');
  const walking = [{ address: 't1', partner: 25, status: 'working', changedAt: 0 }];
  assert.equal(tick.step(walking, files, 750), true, 'starts walking');
  const plan = tick.get('t1').plan;
  assert.equal(tick.step(walking, files, 760), false, 'mid-leg');
  assert.equal(tick.step(walking, files, plan.start + plan.dur), true, 'arrives and stands');
  assert.equal(tick.get('t2'), undefined, 'gone rows are forgotten');
});
