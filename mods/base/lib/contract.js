// Pure rules for reading the usage data contract (no Herdr imports, so Node can test them).
// A provider file or daily snapshot is valid only when it is an object with a numeric `updated`.
export const snapshot = (v) => (v && typeof v === 'object' && !Array.isArray(v) && Number.isFinite(v.updated) ? v : null);

// neutral = { claude?, codex? } read from claude.json / codex.json.
export function resolveLimits(neutral) {
  const n = neutral && typeof neutral === 'object' ? neutral : {};
  return { claude: snapshot(n.claude), codex: snapshot(n.codex) };
}

export const resolveDaily = (neutral) => snapshot(neutral);
