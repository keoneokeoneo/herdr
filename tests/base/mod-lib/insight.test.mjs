import { test } from 'node:test';
import assert from 'node:assert/strict';
import { dailySummary, pace } from '../../../mods/base/lib/usage.js';
import { placeLabel, agentSubline } from '../../../mods/base/lib/model.js';

const H = 3600;
test('dailySummary averages only days with usage and names the range', () => {
  const days = [{ date: '2026-10-22', claude: 0, codex: 0 }, { date: '2026-10-23', claude: 2e6, codex: 0 }, { date: '2026-11-04', claude: 8e6, codex: 6e6 }];
  assert.deepEqual(dailySummary(days), { avg: 8e6, active: 2, from: '10/22', to: '11/4' });
  assert.deepEqual(dailySummary([]), { avg: 0, active: 0, from: null, to: null });
});

test('pace projects the share used by reset and when the limit would be hit', () => {
  const now = 1_000_000;
  // 5h window, 2.5h elapsed, 40% used -> 80% by reset, no hit
  assert.deepEqual(pace({ used: 40, resets_at: now + 2.5 * H }, 300, now * 1000), { atReset: 80, hitsInSec: null });
  // 1h elapsed, 50% used -> 250% by reset, hits 100% after another 1h
  assert.deepEqual(pace({ used: 50, resets_at: now + 4 * H }, 300, now * 1000), { atReset: 250, hitsInSec: H });
  // too early (under 10% of the window) or dead window -> null
  assert.equal(pace({ used: 5, resets_at: now + 4.9 * H }, 300, now * 1000), null);
  assert.equal(pace({ used: 5, resets_at: now - 1 }, 300, now * 1000), null);
  assert.equal(pace(undefined, 300, now * 1000), null);
});

test('placeLabel names a default numbered tab by its session title, else a folder unlike the space', () => {
  assert.equal(placeLabel('1', { title: '✳ Fix the usage bar', cwd: '/k/herdr', space: 'herdr' }), 'Fix the usage bar');
  assert.equal(placeLabel('2', { title: '', cwd: '/k/herdr/helpers/poke/', space: 'herdr' }), 'poke');
  assert.equal(placeLabel('2', { title: '', cwd: '/k/herdr', space: 'herdr' }), '2');
  assert.equal(placeLabel('review', { title: 'x', cwd: '/k/a', space: 'a' }), 'review');
  assert.equal(placeLabel('3', {}), '3');
});

test('agentSubline falls back to folder and branch, never an internal waiting note', () => {
  assert.equal(agentSubline({ status: 'working', cwd: '/k/herdr', branch: 'main', telemetryNote: null }, []), 'herdr · main');
  assert.equal(agentSubline({ status: 'idle', telemetryNote: 'Waiting for session data' }, []), '');
  assert.equal(agentSubline({ status: 'working', model: 'gpt', effort: 'medium', ctx: 29, cwd: '/k/x' }, []), 'gpt · medium · ctx 29%');
});

import { usageShown } from '../../../mods/base/lib/panel.js';
const tree = (hidden, active) => ({ id: 'root', children: [{ id: 'panel', hidden, children: [{ id: 'panel-body', type: 'stack', active, children: [
  { id: 'usage', content: { mod: 'base', props: { view: 'panel' } } }, { id: 'files', content: { mod: 'files' } }] }] }] });
test('usageShown is true only when the panel is open on the usage view', () => {
  assert.equal(usageShown(tree(false, 'usage')), true);
  assert.equal(usageShown(tree(undefined, undefined)), true); // stack shows its first child by default
  assert.equal(usageShown(tree(true, 'usage')), false);
  assert.equal(usageShown(tree(false, 'files')), false);
  assert.equal(usageShown({ id: 'root', children: [] }), false);
  assert.equal(usageShown(null), false);
});
