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

import { agentSummary, spaceMeta, stackAgents } from '../../../mods/base/lib/model.js';
test('pace stays quiet in the last 10% of a window', () => {
  const now = 1_000_000;
  assert.equal(pace({ used: 65, resets_at: now + 0.4 * H }, 300, now * 1000), null);
});

test('agentSummary counts what needs a look, in order of urgency', () => {
  const r = (status) => ({ status });
  assert.equal(agentSummary([r('working'), r('working'), r('blocked'), r('done'), r('idle')]), '1 waiting · 2 working · 1 new');
  assert.equal(agentSummary([r('idle')]), '');
  assert.equal(agentSummary([]), '');
});

test('spaceMeta shows the branch, else how many agents, never a bare path', () => {
  assert.equal(spaceMeta({ branch: 'main', folder: '/Users/me' }, [{}, {}]), '⎇ main');
  assert.equal(spaceMeta({ branch: null, folder: '/Users/me' }, [{}, {}]), '2 agents');
  assert.equal(spaceMeta({ branch: null, folder: '/Users/me' }, [{}]), '1 agent');
  assert.equal(spaceMeta({ branch: null, folder: '/Users/me' }, []), '');
});

test('stackAgents leaves idle agents out of the space row', () => {
  const list = [{ status: 'idle', address: 'a' }, { status: 'working', address: 'b' }, { status: 'blocked', address: 'c' }];
  assert.deepEqual(stackAgents(list).map((x) => x.address), ['c', 'b']);
});

import { limitAlerts, level } from '../../../mods/base/lib/usage.js';
import { nextDone } from '../../../mods/base/lib/model.js';

test('level marks 60% and 80% without touching the provider colour', () => {
  assert.equal(level(59), null);
  assert.equal(level(60), 'high');
  assert.equal(level(80), 'limit');
});

test('limitAlerts fires once per window for 80% and for a forecast hit', () => {
  const now = 1_000_000, ms = now * 1000;
  const limits = {
    claude: { windows: [{ minutes: 300, used: 85, resets_at: now + 2 * H }, { minutes: 10080, used: 10, resets_at: now + 100 * H }] },
    codex: { windows: [{ minutes: 300, used: 50, resets_at: now + 4 * H }] }, // 1h in, 50%: hits before reset
  };
  const first = limitAlerts(limits, ms, []);
  assert.deepEqual(first.map((a) => a.key), [`claude:300:${now + 2 * H}:80`, `codex:300:${now + 4 * H}:pace`]);
  assert.match(first[0].title, /Claude session at 85%/);
  assert.equal(first[0].provider, 'claude');
  assert.match(first[1].title, /Codex session may hit its limit/);
  assert.deepEqual(limitAlerts(limits, ms, first.map((a) => a.key)), []);
  assert.deepEqual(limitAlerts({}, ms, []), []);
});

test('nextDone picks the oldest finished agent to review', () => {
  assert.equal(nextDone([{ status: 'done', since: 5, address: 'a' }, { status: 'done', since: 2, address: 'b' }, { status: 'idle', since: 1, address: 'c' }]).address, 'b');
  assert.equal(nextDone([{ status: 'idle' }]), null);
});

import { limitNote } from '../../../mods/base/lib/usage.js';
import { sinceText } from '../../../mods/base/lib/model.js';

test('limitNote explains a missing window or a missing writer', () => {
  assert.equal(limitNote('codex', { windows: [{ minutes: 10080 }] }), 'No 5h limit on this plan');
  assert.equal(limitNote('codex', { windows: [{ minutes: 300 }, { minutes: 10080 }] }), null);
  assert.equal(limitNote('codex', null), 'Codex limits need the herdr-poke helper');
  assert.equal(limitNote('claude', null), 'Claude limits come from the Claude statusline');
});

test('sinceText says what the row time measures', () => {
  assert.equal(sinceText('working', 2 * 3600e3 + 28 * 60e3), 'Working for 2h 28m');
  assert.equal(sinceText('blocked', 5 * 60e3), 'Waiting for you for 5m');
  assert.equal(sinceText('done', 3 * 60e3), 'Finished 3m ago');
  assert.equal(sinceText('idle', 60 * 60e3), 'Idle for 1h 0m');
});
