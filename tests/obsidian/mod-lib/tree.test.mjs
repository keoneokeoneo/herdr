import test from 'node:test';
import assert from 'node:assert/strict';
import { treeRows, parentFolders, displayName } from '../../../mods/obsidian/tree.js';
test('folders precede notes and remain collapsed until expanded', () => {
 const paths = ['z.md', 'Projects/A/a.md', 'Projects/B/b.md', 'Archive/old.md'];
 assert.deepEqual(treeRows(paths, new Set()).map(r => r.path), ['Archive', 'Projects', 'z.md']);
 assert.deepEqual(treeRows(paths, new Set(['Projects'])).map(r => r.path), ['Archive', 'Projects', 'Projects/A', 'Projects/B', 'z.md']);
});
test('expanded descendants preserve hierarchy and duplicate basenames', () => {
 const rows = treeRows(['A/same.md', 'B/same.md'], new Set(['A', 'B']));
 assert.equal(rows.filter(r => !r.folder).length, 2);
 assert.equal(rows[1].depth, 1);
 assert.deepEqual(parentFolders('A/B/note.md'), ['A', 'A/B']);
});
test('display names omit parent paths and markdown extensions', () => {
 assert.equal(displayName('/Users/keno/personal-vault/'), 'personal-vault');
 assert.equal(displayName('Folder/한글 노트.md', true), '한글 노트');
});
