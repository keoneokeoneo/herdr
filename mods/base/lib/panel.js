// Right-panel tabs are declared by each layout on the panel-tabs block:
// props = { stack?: "panel-body", tabs: [[blockId, label], ...] }.
export const tabsFrom = (props) =>
  (Array.isArray(props?.tabs) ? props.tabs : []).filter((t) => Array.isArray(t) && typeof t[0] === 'string' && typeof t[1] === 'string');

export const stackOf = (props) => props?.stack ?? 'panel-body';
