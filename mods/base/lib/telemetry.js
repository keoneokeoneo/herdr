// Terminal labels outlive the agent process. Only display a snapshot belonging
// to the current foreground process and provider; legacy labels are unverified.
export function telemetry(labels, pid, provider) {
  const empty = { model: null, effort: null, ctx: null, note: 'Telemetry unavailable' };
  if (!(Number(pid) > 0) || labels['poke.owner'] !== String(pid) || labels['poke.provider'] !== provider) return empty;
  const model = labels['poke.model'] || null;
  const effort = labels['poke.effort'] || null;
  const raw = labels['poke.ctx'];
  const n = raw == null || raw === '' ? NaN : Number(raw);
  const ctx = Number.isFinite(n) && n >= 0 && n <= 100 ? n : null;
  return { model, effort, ctx, note: !model ? 'Waiting for session data' : ctx == null ? 'Waiting for context' : null };
}

// Claude no longer publishes telemetry labels (its statusline only records limits), so its
// rows show no model/effort/context and no "unavailable" note. Other providers (Codex) do.
export function telemetryFor(labels, pid, provider) {
  if (provider === 'claude') return { model: null, effort: null, ctx: null, note: null };
  return telemetry(labels, pid, provider);
}
