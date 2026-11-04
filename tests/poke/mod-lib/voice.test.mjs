import { test } from 'node:test';
import assert from 'node:assert/strict';
import { say } from '../../../mods/poke/lib/voice.js';

test('jolly level', () => assert.equal(say('jolly', 'level', 'Charmeleon', 44), 'Yay! Charmeleon grew to Lv.44!'));
test('unknown nature falls back to neutral', () => assert.equal(say('hardy', 'level', 'X', 2), 'X grew to Lv.2.'));
test('evoE', () => assert.equal(say('jolly', 'evoE', 'Charmeleon', 'Charizard'), 'Charmeleon evolved into Charizard! Woohoo!'));
