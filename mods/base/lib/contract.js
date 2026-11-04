// Pure rules for reading the usage data contract (no Herdr imports, so Node can test them).
// A provider file or daily snapshot is valid only when it is an object with a numeric `updated`.
export const snapshot = (v) => (v && typeof v === 'object' && !Array.isArray(v) && Number.isFinite(v.updated) ? v : null);

// neutral = { claude?, codex? } read from claude.json / codex.json.
// A provider file must carry a windows list and a daily file a days list; anything else is "no data".
const withList = (v, key) => (snapshot(v) && Array.isArray(v[key]) ? v : null);

export function resolveLimits(neutral) {
  const n = neutral && typeof neutral === 'object' ? neutral : {};
  return { claude: withList(n.claude, 'windows'), codex: withList(n.codex, 'windows') };
}

export const resolveDaily = (neutral) => withList(neutral, 'days');
