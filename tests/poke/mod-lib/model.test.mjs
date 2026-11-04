import { test } from 'node:test';
import assert from 'node:assert/strict';
import { URG, byUrgency, groups, waiting, chips, tabPill, typeOf, TYPES } from '../../../mods/poke/lib/model.js';

const M = 60e3, now = 1_000_000_000;
test('URG order', () => assert.deepEqual(URG, { blocked: 0, working: 1, done: 2, idle: 3 }));
test('byUrgency sorts blocked first', () => {
  assert.deepEqual(byUrgency([{ status: 'idle' }, { status: 'blocked' }, { status: 'working' }]).map(a => a.status), ['blocked', 'working', 'idle']);
});
test('groups', () => {
  const g = groups([{ status: 'blocked' }, { status: 'done' }, { status: 'done' }, { status: 'idle' }]);
  assert.equal(g.needs.length, 1); assert.equal(g.done.length, 2); assert.equal(g.working.length, 0); assert.equal(g.idle.length, 1);
});
test('waiting = blocked oldest first', () => {
  const w = waiting([{ id: 'a', status: 'blocked', since: 5 }, { id: 'b', status: 'working', since: 1 }, { id: 'c', status: 'blocked', since: 2 }]);
  assert.deepEqual(w.map(a => a.id), ['c', 'a']);
});
test('chips ctx', () => {
  assert.deepEqual(chips({ status: 'working', ctx: 83, since: now }, now).map(c => c.text), ['ctx 83%']);
  assert.equal(chips({ status: 'working', ctx: 79, since: now }, now).length, 0);
  assert.equal(chips({ status: 'working', ctx: null, since: now }, now).length, 0);
});
test('chips waiting', () => {
  assert.deepEqual(chips({ status: 'blocked', since: now - 6 * M }, now).map(c => c.text), ['waiting 6m']);
  assert.equal(chips({ status: 'blocked', since: now - 4 * M }, now).length, 0);
});
test('tabPill', () => {
  assert.deepEqual(tabPill([{ status: 'blocked' }, { status: 'working' }]), { text: '! needs you', tone: 'blocked' });
  assert.deepEqual(tabPill([{ status: 'working' }, { status: 'working' }]), { text: '● 2 working', tone: 'working' });
  assert.deepEqual(tabPill([{ status: 'done' }]), { text: '✓ new', tone: 'accent' });
  assert.deepEqual(tabPill([{ status: 'idle' }]), { text: 'idle', tone: 'dim' });
  assert.deepEqual(tabPill([]), { text: '', tone: 'dim' });
});
test('typeOf random is stable and valid', () => {
  const a = typeOf('feat/x', 7, 'random'), b = typeOf('feat/x', 7, 'random');
  assert.equal(a, b); assert.ok(TYPES.includes(a));
  assert.equal(typeOf('feat/x', 7, 'off'), null);
  assert.equal(typeOf('feat/x', 7, 'repo', 'water'), 'water');
});
