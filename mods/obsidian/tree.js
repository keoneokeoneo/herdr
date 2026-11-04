export function displayName(path, note = false) {
  const name = path.replace(/\/+$/, '').split('/').pop() || '';
  return note ? name.replace(/\.md$/i, '') : name;
}
export function parentFolders(path) {
  const parts = path.split('/');
  return parts.slice(0, -1).map((_p, i) => parts.slice(0, i + 1).join('/'));
}
export function treeRows(paths, expanded) {
  const root = new Map();
  for (const path of paths) {
    let children = root;
    const parts = path.split('/');
    parts.forEach((name, i) => {
      const folder = i < parts.length - 1;
      const key = (folder ? 'd:' : 'f:') + name;
      if (!children.has(key)) children.set(key, { name, folder, path: parts.slice(0, i + 1).join('/'), children: new Map() });
      children = children.get(key).children;
    });
  }
  const rows = [];
  function visit(children, depth) {
    for (const node of [...children.values()].sort((a, b) => Number(b.folder) - Number(a.folder) || a.name.localeCompare(b.name))) {
      rows.push({ path: node.path, name: displayName(node.name, !node.folder), folder: node.folder, depth });
      if (node.folder && expanded.has(node.path)) visit(node.children, depth + 1);
    }
  }
  visit(root, 0);
  return rows;
}
