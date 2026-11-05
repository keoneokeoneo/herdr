// Poke's behaviours: partner assignment, Garden walking art, pokeTokenBar sync, evening flag, level/evolution toasts.
// Things that happen, not things drawn. Runs once, in the persona host's copy.
import { hc, exec, home, appHome, shared, setShared, notify, watchFolder } from "herdr";
import { personaRows } from "./views/rows.js";
import { readJSON, PTB_DIR } from "./views/store.js";
import { typeModeInit } from "./lib/defaults.js";
import { excluded, pickPartner, activePartners } from "./lib/partner.js";
import { retryDue, assetsMissing } from "./lib/assets.js";
import { walkMissing } from "./lib/walk.js";
import { say } from "./lib/voice.js";
import { ptbChanged, coalesce } from "./lib/sync.js";

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
  await refreshNames(); // a partner fetch also adds to names.json
}

function syncSeen(all) {
  const seen = activePartners(all);
  if (JSON.stringify(seen) !== JSON.stringify(shared("poke.seen") ?? [])) setShared("poke.seen", seen);
}

async function speciesName(id) {
  const d = await readJSON(`${PTB_DIR()}/pokemon-details-v1/${id}.json`);
  return cap(d?.details?.name) || `#${id}`;
}

export function start(s, cx) {
  s.pending = new Set();   // addresses being labelled
  s.tried = new Map();     // partner id → asset retry backoff
  s.walkTried = new Map(); // partner id → walking-art retry backoff

  // Companion sprite, Dex copies and evolution/level toasts follow pokeTokenBar's
  // folder as it changes. The slow timer only covers a missed or refused watch.
  const ptbSync = coalesce(async () => {
    await exec(HP(), ["assets", "--companion"]).catch(() => null);
    await exec(HP(), ["assets", "--dex"]).catch(() => null);
    await refreshAssets();
    await companionDiff(s);
  });
  ptbSync();
  cx.timer.every(300000, () => { ptbSync(); });
  cx.spawn(async () => {
    for (;;) {
      const paths = await watchFolder(PTB_DIR()).catch(() => null);
      if (!paths) return; // watch refused: the timer carries on alone
      if (ptbChanged(paths, PTB_DIR())) ptbSync();
    }
  });

  const initial = typeModeInit(shared("poke.typeMode"));
  if (initial) setShared("poke.typeMode", initial);
  const tick = (cx) => cx.spawn(async () => { await step(s); });
  tick(cx);
  cx.timer.every(2000, tick);
  const evening = () => { const h = new Date().getHours(); const on = h >= 18 || h < 6; if (shared("poke.evening") !== on) setShared("poke.evening", on); };
  evening();
  cx.timer.every(60000, evening);
}

async function step(s) {
  const all = personaRows();
  syncSeen(all);
  const sprites = shared("poke.sprites") ?? [];
  for (const r of all) {
    if (!r.partnerSet && !s.pending.has(r.address)) assign(s, r);
    // A partner whose sprite or name never arrived (offline, API hiccup) is fetched again, with backoff.
    if (r.partnerSet && assetsMissing(r.partner, sprites, shared("poke.names")) && retryDue(s.tried, r.partner, Date.now())) {
      exec(HP(), ["assets", String(r.partner)]).then(refreshAssets, () => null);
    }
    // The Garden's walking art (PMD sheets cut by the helper); a species without any is marked once.
    if (r.partnerSet && walkMissing(r.partner, sprites) && retryDue(s.walkTried, r.partner, Date.now())) {
      exec(HP(), ["assets", "--walk", String(r.partner)]).then(refreshAssets, () => null);
    }
  }
  const live = new Set(all.map((r) => r.address));
  for (const k of s.pending) if (!live.has(k)) s.pending.delete(k);
}

async function assign(s, r) {
  s.pending.add(r.address);
  try {
    const st = await readJSON(`${PTB_DIR()}/companion-state.json`);
    const id = pickPartner(Math.random, excluded(st));
    const res = await hc(["label", r.address, `poke.partner=${id}`]);
    if (res.code !== 0) { s.pending.delete(r.address); return; }
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
  } else if (now.level > was.level) {
    notify({ title: say(now.nature, "level", await speciesName(now.id), now.level), level: "info", timeout: 5000 });
  }
}

export function command(s, c) {
  if (c === "reroll") {
    setShared("poke.typeSeed", ((Number(shared("poke.typeSeed") ?? 7)) * 7 + 13) % 9973);
  }
}

// Partner names for the Dex grid, from the helper's names.json.
async function refreshNames() {
  const n = await readJSON(`${appHome()}/poke-assets/names.json`);
  if (n && JSON.stringify(n) !== JSON.stringify(shared("poke.names") ?? {})) setShared("poke.names", n);
}
