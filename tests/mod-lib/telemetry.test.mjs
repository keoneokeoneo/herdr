import {test} from 'node:test';
import assert from 'node:assert/strict';
import {telemetry} from '../../mods/poke/lib/telemetry.js';
const labels = {'poke.owner':'123','poke.provider':'codex','poke.session':'s1','poke.model':'gpt-test','poke.effort':'high','poke.ctx':'92'};
test('rejects previous process, provider and unverified legacy labels',()=>{
  for(const [pid,provider,l] of [[124,'codex',labels],[123,'claude',labels],[123,'codex',{'poke.model':'old','poke.ctx':'92'}],[null,'codex',labels]]) {
    const t=telemetry(l,pid,provider);assert.equal(t.model,null);assert.equal(t.ctx,null);assert.equal(t.note,'Telemetry unavailable');
  }
});
test('accepts matching process and valid data including zero context',()=>{
  assert.deepEqual(telemetry({...labels,'poke.ctx':'0'},123,'codex'),{model:'gpt-test',effort:'high',ctx:0,note:null});
});
test('new collector awaiting rollout never exposes previous fields',()=>{
  assert.deepEqual(telemetry({'poke.owner':'123','poke.provider':'codex'},123,'codex'),{model:null,effort:null,ctx:null,note:'Waiting for session data'});
});
test('invalid context is hidden and partial data has an honest note',()=>{
  for(const ctx of ['NaN','-1','101',''])assert.equal(telemetry({...labels,'poke.ctx':ctx},123,'codex').ctx,null);
  assert.equal(telemetry({...labels,'poke.ctx':''},123,'codex').note,'Waiting for context');
});
