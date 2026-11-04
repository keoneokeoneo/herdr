// Right-panel tabs are declared by each layout on the panel-tabs block:
// props = { stack?: "panel-body", tabs: [[blockId, label], ...] }.
export const tabsFrom = (props) =>
  (Array.isArray(props?.tabs) ? props.tabs : []).filter((t) => Array.isArray(t) && typeof t[0] === 'string' && typeof t[1] === 'string');

export const stackOf = (props) => props?.stack ?? 'panel-body';

const find = (node, id) => {
  if (!node || typeof node !== 'object') return null;
  if (node.id === id) return node;
  for (const c of node.children ?? []) { const f = find(c, id); if (f) return f; }
  return null;
};
const hasUsage = (node) => !!node && (node.content?.mod === 'base' && node.content?.props?.view === 'panel'
  || (node.children ?? []).some(hasUsage));

// True when the right panel is open and its stack shows the Usage view (so the status bar need not repeat it).
export function usageShown(tree) {
  const panel = find(tree, 'panel');
  if (!panel || panel.hidden) return false;
  const body = find(panel, 'panel-body');
  if (!body) return hasUsage(panel);
  const kids = body.children ?? [];
  return hasUsage(kids.find((c) => c.id === body.active) ?? kids[0]);
}
