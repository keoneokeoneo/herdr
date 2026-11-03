export const MAX_SPECIES = 1025;

// Species Keno already owns never show up as session partners.
export function excluded(state) {
  const s = new Set();
  for (const e of state?.dex || []) for (const id of e.chainOrder || []) s.add(id);
  for (const id of state?.active?.plannedPathIDs || []) s.add(id);
  return s;
}

export function pickPartner(random, excludedSet) {
  const free = [];
  for (let id = 1; id <= MAX_SPECIES; id++) if (!excludedSet.has(id)) free.push(id);
  return free[Math.min(free.length - 1, Math.floor(random() * free.length))];
}

// Only assigned species on currently open agent sessions, in a stable order.
export function activePartners(rows) {
  return [...new Set(rows.filter((r) => r.partnerSet && Number.isInteger(r.partner)
    && r.partner >= 1 && r.partner <= MAX_SPECIES).map((r) => r.partner))].sort((a, b) => a - b);
}
