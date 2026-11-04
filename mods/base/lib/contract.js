// Pure rules for reading the usage data contract (no Herdr imports, so Node can test them).
// A provider file or daily snapshot is valid only when it is an object with a numeric `updated`.
const valid = (v) => !!v && typeof v === 'object' && !Array.isArray(v) && Number.isFinite(v.updated);

// The snapshot with the larger `updated`; on a tie the first wins. Null when neither is valid.
export function pickFresher(a, b) {
  if (valid(a) && valid(b)) return b.updated > a.updated ? b : a;
  return valid(a) ? a : valid(b) ? b : null;
}

const obj = (v) => (v && typeof v === 'object' ? v : {});

// neutral/legacy are { claude?, codex? }; legacy only wins when neutral is missing or older.
export function resolveLimits(neutral, legacy) {
  const n = obj(neutral), l = obj(legacy);
  return { claude: pickFresher(n.claude, l.claude), codex: pickFresher(n.codex, l.codex) };
}

export const resolveDaily = (neutral, legacy) => pickFresher(neutral, legacy);
