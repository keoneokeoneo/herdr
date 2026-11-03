// pokeTokenBar growth formula (PokemonBalance in chattymin/PokeTokenBar).
export const GRAD = { common: 750e6, uncommon: 1875e6, rare: 3e9, legendary: 6e9 };

export function stageCost(rarity, k, stageIndex, difficulty = 1) {
  const kk = Math.max(1, k);
  const base = Math.round((GRAD[rarity] * (stageIndex + 1)) / ((kk * (kk + 1)) / 2));
  return Math.round(Math.max(1, base) * difficulty);
}

export function progress(active, difficulty = 1) {
  const cost = stageCost(active.rarity, active.totalForms, active.stageIndex, difficulty);
  const used = Math.max(0, active.usedAtStage || 0);
  const path = active.plannedPathIDs || [];
  return {
    pct: Math.min(100, (used / cost) * 100),
    toNext: Math.max(0, cost - used),
    nextId: path[active.stageIndex + 1] ?? null,
  };
}
