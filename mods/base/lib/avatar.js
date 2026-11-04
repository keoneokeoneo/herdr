// Persona seam: Poke publishes its sprite file names as shared "poke.sprites". Without it
// (no Poke running) an avatar is a plain letter.
export function spriteSrc(files, id, dir) {
  const have = Array.isArray(files) ? files : [];
  if (have.includes(`${id}.gif`)) return `${dir}/${id}.gif`;
  if (have.includes(`${id}.png`)) return `${dir}/${id}.png`;
  return null;
}

export const letterFor = (provider) => (provider === 'codex' ? 'X' : 'C');
