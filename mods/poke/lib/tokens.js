// Same rounding as the base mod's usage figures (mods cannot import each other).
export function tokensM(n) {
  if (n >= 1e9) return `${(n / 1e9).toFixed(1)}B`;
  return `${(n / 1e6).toFixed(1)}M`;
}
