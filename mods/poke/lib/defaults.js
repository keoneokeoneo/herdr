// Poke's tab-type pills are off in the neutral base layout; Poke turns them on once, when unset.
// Returns the value to write, or null when something is already stored.
export const typeModeInit = (current) => (current == null ? 'random' : null);
