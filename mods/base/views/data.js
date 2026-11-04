// One place that turns Herdr's live data into rows the views draw.
import { agents, terminals, panes, spaces } from "herdr";

import { telemetryFor } from "../lib/telemetry.js";

const PROVIDER = { "claude-code": "claude", claude: "claude", codex: "codex" };

export function providerOf(a) {
  return PROVIDER[a.app] ?? PROVIDER[a.program?.key] ?? "agent";
}

// Until the partner label arrives, show a stable stand-in so rows don't jump.
function fallbackPartner(address) {
  let h = 7;
  for (const c of address) h = (h * 31 + c.charCodeAt(0)) >>> 0;
  return (h % 1025) + 1;
}

export function rows() {
  const byAddr = new Map(terminals().map((t) => [t.address, t]));
  const paneBy = new Map(panes().map((p) => [p.pane, p]));
  return agents().map((a) => {
    const t = byAddr.get(a.address) ?? {};
    const l = t.labels ?? {};
    const info = telemetryFor(l, t.foreground?.pid, providerOf(a));
    const p = paneBy.get(a.pane) ?? {};
    const num = (v) => (v == null || v === "" ? null : Number(v));
    return {
      address: a.address, pane: a.pane, space: a.space, spaceName: a.space_name,
      tab: a.tab, tabName: a.tab_name, status: a.status, focused: a.focused, pinned: !!a.pinned, title: a.title ?? "",
      provider: providerOf(a), program: a.program,
      since: t.status_ms ?? t.created_ms ?? Date.now(), turn: t.turn ?? 0,
      partner: num(l["poke.partner"]) ?? fallbackPartner(a.address),
      partnerSet: l["poke.partner"] != null,
      model: info.model, effort: info.effort, ctx: info.ctx, telemetryNote: info.note,
      reviewOf: l["poke.review_of"] ?? null, findings: num(l["poke.findings"]),
      cwd: p.cwd ?? t.cwd ?? null, branch: p.branch ?? null,
    };
  });
}

export const spaceList = () => spaces();
// Outside mods/: Herdr reloads a mod on any write inside its folder.
export const SPRITE_DIR = "poke-assets";
