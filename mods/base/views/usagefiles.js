// Reads the usage data contract: ~/.local/state/herdr-app/usage/{claude,codex,daily}.json.
import { home } from "herdr";
import { readJSON } from "./store.js";
import { resolveLimits, resolveDaily } from "../lib/contract.js";

export const USAGE_DIR = () => `${home()}/.local/state/herdr-app/usage`;

export async function loadUsage(s) {
  const [claude, codex, daily] = await Promise.all([
    readJSON(`${USAGE_DIR()}/claude.json`), readJSON(`${USAGE_DIR()}/codex.json`), readJSON(`${USAGE_DIR()}/daily.json`),
  ]);
  s.limits = resolveLimits({ claude, codex });
  s.usage = resolveDaily(daily);
}
