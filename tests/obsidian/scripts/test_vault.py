import importlib.util
from pathlib import Path
import tempfile
import unittest

spec = importlib.util.spec_from_file_location('vault', Path(__file__).resolve().parents[3] / 'mods/obsidian/vault.py')
vault = importlib.util.module_from_spec(spec)
spec.loader.exec_module(vault)

class VaultTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name).resolve()
        (self.root / 'folder').mkdir()
        (self.root / 'folder/한글 노트.md').write_text('# 제목\nunique body')
        (self.root / 'index.md').write_text('[[한글 노트]]')

    def test_search_body_and_names(self):
        self.assertEqual(vault.request('search', [str(self.root), 'unique'])['notes'], ['folder/한글 노트.md'])
        self.assertEqual(vault.request('search', [str(self.root), '한글'])['notes'], ['folder/한글 노트.md', 'index.md'])

    def test_wiki_relative_root_and_encoded_links(self):
        for link, current in [('한글 노트#제목', 'index.md'), ('../index.md', 'folder/한글 노트.md'), ('/index.md', 'folder/한글 노트.md'), ('%ED%95%9C%EA%B8%80%20%EB%85%B8%ED%8A%B8', 'index.md')]:
            expected = 'index.md' if 'index' in link else 'folder/한글 노트.md'
            self.assertEqual(vault.resolve(self.root, link, current), expected)

    def test_escape_and_symlink_rejected(self):
        with self.assertRaises(ValueError): vault.read(self.root, '../outside.md')
        (self.root / 'escape.md').symlink_to(self.root.parent / 'outside.md')
        self.assertNotIn('escape.md', vault.notes(self.root))
        with self.assertRaises(ValueError): vault.read(self.root, 'escape.md')

    def test_hidden_and_large_files(self):
        (self.root / '.obsidian').mkdir()
        (self.root / '.obsidian/secret.md').write_text('hidden')
        self.assertNotIn('.obsidian/secret.md', vault.notes(self.root))
        (self.root / 'large.md').write_text('x' * (vault.LIMIT + 1))
        with self.assertRaises(ValueError): vault.read(self.root, 'large.md')

    def test_empty_search_keeps_all_tree_entries(self):
        for i in range(205):
            (self.root / f'note-{i}.md').write_text('common')
        self.assertEqual(len(vault.request('search', [str(self.root), ''])['notes']), 207)
        self.assertEqual(len(vault.request('search', [str(self.root), 'common'])['notes']), 200)

    def test_vaults_are_independent(self):
        with tempfile.TemporaryDirectory() as other:
            Path(other, 'personal.md').write_text('personal only')
            self.assertEqual(vault.request('search', [other, ''])['notes'], ['personal.md'])
            self.assertNotIn('personal.md', vault.request('search', [str(self.root), ''])['notes'])
            with self.assertRaises(ValueError):
                vault.request('read', [str(self.root), 'personal', 'index.md'])

    def test_ambiguous_and_missing(self):
        (self.root / 'other').mkdir()
        (self.root / 'other/한글 노트.md').write_text('duplicate')
        with self.assertRaises(ValueError): vault.resolve(self.root, '한글 노트', 'index.md')
        with self.assertRaises(ValueError): vault.resolve(self.root, 'missing', 'index.md')

if __name__ == '__main__': unittest.main()
