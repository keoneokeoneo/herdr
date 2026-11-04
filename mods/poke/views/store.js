// Reads pokeTokenBar's files (read only; never edited).
import { home, readTextFile } from "herdr";

export const PTB_DIR = () => `${home()}/Library/Application Support/PokeTokenBar`;

export async function readJSON(path) {
  try { const t = await readTextFile(path); return t ? JSON.parse(t) : null; } catch { return null; }
}
