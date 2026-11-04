// Reads the usage data contract: ~/.local/state/herdr-app/usage/{claude,codex,daily}.json.
// Until the cutover (plan Task 10) the old ~/.local/state/herdr-poke files are a fallback.
import { home } from "herdr";
import { readJSON } from "./store.js";
import { resolveLimits, resolveDaily } from "../lib/contract.js";

export const USAGE_DIR = () => `${home()}/.local/state/herdr-app/usage`;
const LEGACY_DIR = () => `${home()}/.local/state/herdr-poke`;

export async function loadUsage(s) {
  const [claude, codex, daily, legacyLimits, legacyUsage] = await Promise.all([
    readJSON(`${USAGE_DIR()}/claude.json`), readJSON(`${USAGE_DIR()}/codex.json`), readJSON(`${USAGE_DIR()}/daily.json`),
    readJSON(`${LEGACY_DIR()}/limits.json`), readJSON(`${LEGACY_DIR()}/usage.json`),
  ]);
  s.limits = resolveLimits({ claude, codex }, legacyLimits);
  s.usage = resolveDaily(daily, legacyUsage);
}
