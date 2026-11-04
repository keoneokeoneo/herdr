// Things that happen, not things drawn. Started once, in the status bar's copy.
import { exec, home, run } from "herdr";
import { rows } from "./views/data.js";
import { waiting } from "./lib/model.js";

// The helper binary mirrors each Codex session's limits and labels; without it Codex rows are plain.
const HELPER = () => `${home()}/.local/bin/herdr-poke`;

export function start(s, cx) {
  s.jumpAt = -1;
  s.attachTried = new Map(); // codex terminal → next health probe
  const tick = (cx) => cx.spawn(async () => { probeCodex(s); });
  tick(cx);
  cx.timer.every(2000, tick);
}

function attach(terminal) {
  const sh = `export PATH="$HOME/.local/bin:/opt/homebrew/bin:/usr/local/bin:$PATH"; nohup "$0" codex --attach "$1" >/dev/null 2>&1 &`;
  exec("sh", ["-c", sh, HELPER(), terminal]).catch(() => null);
}

// Probe every Codex periodically, including one with stale labels after a mirror crash.
// The helper exits at once when the terminal is already mirrored, so a retry is cheap.
function probeCodex(s) {
  const all = rows();
  for (const r of all) {
    if (r.provider === "codex" && Date.now() >= (s.attachTried.get(r.address) ?? 0)) {
      s.attachTried.set(r.address, Date.now() + 60000);
      attach(r.address);
    }
  }
  const live = new Set(all.map((r) => r.address));
  for (const k of s.attachTried.keys()) if (!live.has(k)) s.attachTried.delete(k);
}

// Focus the next session waiting for you (oldest first).
export function jump(s) {
  const list = waiting(rows());
  if (!list.length) return;
  s.jumpAt = ((s.jumpAt ?? -1) + 1) % list.length;
  run("agent.focus", { terminal: list[s.jumpAt].address });
}

// Show the right panel on its Obsidian tab.
export function notes() {
  run("layout.show", { id: "panel" });
  run("layout.select", { stack: "panel-body", child: "obsidian" });
}
