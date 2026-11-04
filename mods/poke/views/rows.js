// What the persona host needs from Herdr's live data: each agent's status and partner label.
import { agents, terminals } from "herdr";

export function personaRows() {
  const byAddr = new Map(terminals().map((t) => [t.address, t]));
  return agents().map((a) => {
    const raw = (byAddr.get(a.address)?.labels ?? {})["poke.partner"];
    const n = raw == null || raw === "" ? null : Number(raw);
    return { address: a.address, status: a.status, partner: Number.isFinite(n) ? n : null, partnerSet: raw != null };
  });
}
