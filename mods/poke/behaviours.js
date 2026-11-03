// Things that happen, not things drawn. Runs once, in the status bar's copy.
import { hc, exec, home, run, shared, setShared, playSound, notify } from "herdr";
import { rows } from "./views/data.js";
import { readJSON, PTB_DIR } from "./views/store.js";
import { excluded, pickPartner } from "./lib/partner.js";
import { waiting } from "./lib/model.js";
import { say } from "./lib/voice.js";

const HP = () => `${home()}/.local/bin/herdr-poke`;
const cap = (s) => (s ? s.split("-").map((w) => w[0].toUpperCase() + w.slice(1)).join(" ") : "");

async function refreshAssets() {
  const r = await exec(HP(), ["assets", "--list"]).catch(() => null);
  if (r?.code !== 0) return;
  try {
    const list = JSON.parse(r.stdout);
    // setShared re-renders every view, so only when the list really changed.
    if (Array.isArray(list) && JSON.stringify(list) !== JSON.stringify(shared("poke.sprites") ?? [])) setShared("poke.sprites", list);
  } catch {}
}

function addSeen(id) {
  const seen = shared("poke.seen") ?? [];
  if (!seen.includes(id)) setShared("poke.seen", [...seen, id]);
}

async function speciesName(id) {
  const d = await readJSON(`${PTB_DIR()}/pokemon-details-v1/${id}.json`);
  return cap(d?.details?.name) || `#${id}`;
}

export function start(s, cx) {
  s.prev = new Map();      // address → last status
  s.pending = new Set();   // addresses being labelled
  s.jumpAt = -1;
  s.onCommand = (c) => command(s, c);

  const assets = (cx) => cx.spawn(async () => { await exec(HP(), ["assets", "--companion"]).catch(() => null); await refreshAssets(); });
  assets(cx);
  cx.timer.every(30000, assets);

  const tick = (cx) => cx.spawn(async (cx) => { await step(s); cx.notify(); });
  tick(cx);
  cx.timer.every(2000, tick);
  const evening = () => { const h = new Date().getHours(); const on = h >= 18 || h < 6; if (shared("poke.evening") !== on) setShared("poke.evening", on); };
  evening();
  cx.timer.every(60000, evening);
  const ptb = (cx) => cx.spawn(async () => { await companionDiff(s); });
  ptb(cx);
  cx.timer.every(30000, ptb);
}

async function step(s) {
  const all = rows();
  const sprites = shared("poke.sprites") ?? [];
  for (const r of all) {
    if (r.partnerSet) addSeen(r.partner);
    else if (!s.pending.has(r.address)) assign(s, r);
    const was = s.prev.get(r.address);
    if (r.status === "blocked" && was && was !== "blocked" && r.partnerSet && sprites.includes(`${r.partner}.mp3`)) {
      playSound(`poke-assets/${r.partner}.mp3`);
    }
    s.prev.set(r.address, r.status);
  }
  const live = new Set(all.map((r) => r.address));
  for (const k of s.prev.keys()) if (!live.has(k)) s.prev.delete(k);
}

async function assign(s, r) {
  s.pending.add(r.address);
  try {
    const st = await readJSON(`${PTB_DIR()}/companion-state.json`);
    const id = pickPartner(Math.random, excluded(st));
    const res = await hc(["label", r.address, `poke.partner=${id}`]);
    if (res.code !== 0) { s.pending.delete(r.address); return; }
    addSeen(id);
    await exec(HP(), ["assets", String(id)]).catch(() => null);
    await refreshAssets();
  } catch { s.pending.delete(r.address); }
}

async function companionDiff(s) {
  const st = await readJSON(`${PTB_DIR()}/companion-state.json`);
  const a = st?.active;
  if (!a) return;
  const id = a.pathIDs?.at(-1) ?? a.baseID;
  const now = { key: a.profile?.instanceID, id, level: a.profile?.level ?? 0, nature: a.nature };
  const was = s.comp;
  s.comp = now;
  if (!was || was.key !== now.key) return; // first look, or a new egg
  if (now.id !== was.id) {
    const [p, q] = await Promise.all([speciesName(was.id), speciesName(now.id)]);
    notify({ title: say(now.nature, "evoE", p, q), level: "success", timeout: 8000 });
    await exec(HP(), ["assets", "--companion"]).catch(() => null);
    await refreshAssets();
  } else if (now.level > was.level) {
    notify({ title: say(now.nature, "level", await speciesName(now.id), now.level), level: "info", timeout: 5000 });
  }
}

function command(s, c) {
  if (c === "jump") {
    const list = waiting(rows());
    if (!list.length) return;
    s.jumpAt = (s.jumpAt + 1) % list.length;
    run("agent.focus", { terminal: list[s.jumpAt].address });
  } else if (c === "reroll") {
    setShared("poke.typeSeed", ((Number(shared("poke.typeSeed") ?? 7)) * 7 + 13) % 9973);
  }
}
