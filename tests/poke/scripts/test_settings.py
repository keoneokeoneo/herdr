import importlib.util
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

SCRIPT = Path(__file__).resolve().parents[3] / 'scripts/poke/settings_edit.py'

class SettingsTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.home = Path(self.tmp.name)
        with patch.dict(os.environ, {'HOME': str(self.home), 'HERDR_APP_HOME': str(self.home / 'app'), 'HERDR_POKE_STATE': str(self.home / 'state')}):
            spec = importlib.util.spec_from_file_location('settings', SCRIPT)
            self.m = importlib.util.module_from_spec(spec)
            spec.loader.exec_module(self.m)
        self.binary = str(self.home / '.local/bin/herdr-poke')

    def claude(self, data):
        p = Path(self.m.CLAUDE)
        p.parent.mkdir(parents=True, exist_ok=True)
        p.write_text(json.dumps(data, indent=2) + '\n')
        return p

    def test_fresh_repeat_and_undo(self):
        self.m.install(self.binary)
        self.m.install(self.binary)
        self.assertIn(self.m.NEW_ALIAS, Path(self.m.ZSHRC).read_text())
        self.assertEqual(Path(self.m.ZSHRC).read_text().count(self.m.NEW_ALIAS), 1)
        self.m.uninstall()
        self.m.uninstall()
        self.assertNotIn(self.m.NEW_ALIAS, Path(self.m.ZSHRC).read_text())

    def test_install_leaves_claude_statusline_untouched(self):
        p = self.claude({'statusLine': {'type': 'command', 'command': 'my-status'}, 'other': 1})
        before = p.read_bytes()
        self.m.install(self.binary)
        self.assertEqual(p.read_bytes(), before)

    def test_install_does_not_create_claude_settings(self):
        self.m.install(self.binary)
        self.assertFalse(Path(self.m.CLAUDE).exists())
        self.assertNotIn('claude_statusLine', json.loads(Path(self.m.MANIFEST).read_text()))

    def test_invalid_claude_settings_do_not_block_install(self):
        p = Path(self.m.CLAUDE)
        p.parent.mkdir(parents=True)
        p.write_text('{not json')
        self.m.install(self.binary)
        self.assertEqual(p.read_text(), '{not json')

    def test_uninstall_does_not_touch_a_custom_statusline_and_drops_the_legacy_key(self):
        p = self.claude({'statusLine': {'type': 'command', 'command': 'my-status'}})
        before = p.read_bytes()
        self.m.save_manifest({'claude_statusLine': {'type': 'command', 'command': 'old'}, 'zshrc_alias': 'added'})
        self.m.uninstall()
        self.assertEqual(p.read_bytes(), before)
        self.assertNotIn('claude_statusLine', self.m.manifest())

    def test_uninstall_restores_only_a_legacy_statusline_that_is_still_ours(self):
        original = {'type': 'command', 'command': 'my-status'}
        p = self.claude({'statusLine': {'type': 'command', 'command': f"{self.binary} statusline"}})
        self.m.save_manifest({'claude_statusLine': original})
        self.m.uninstall()
        self.assertEqual(json.loads(p.read_text())['statusLine'], original)

    def test_invalid_json_preflight(self):
        Path(self.m.HERDR).parent.mkdir(parents=True)
        Path(self.m.HERDR).write_text('{broken')
        with self.assertRaises((SystemExit, ValueError)):
            self.m.install(self.binary)
        self.assertFalse(Path(self.m.ZSHRC).exists())

    def test_failure_after_first_write_keeps_original(self):
        herdr = Path(self.m.HERDR)
        herdr.parent.mkdir(parents=True)
        herdr.write_text(json.dumps({'sounds': {'enabled': True}}))
        real = self.m.write_atomic
        def fail(path, text, expect):
            if path == self.m.HERDR:
                raise OSError('injected interruption')
            return real(path, text, expect)
        with patch.object(self.m, 'write_atomic', side_effect=fail):
            with self.assertRaises(OSError):
                self.m.install(self.binary)
        self.m.install(self.binary)
        self.m.uninstall()
        self.assertEqual(json.loads(herdr.read_text())['sounds']['enabled'], True)

    def test_custom_alias_is_not_overwritten(self):
        Path(self.m.ZSHRC).write_text("alias codex='custom-codex'\n")
        with self.assertRaises(SystemExit):
            self.m.install(self.binary)
        self.assertEqual(Path(self.m.ZSHRC).read_text(), "alias codex='custom-codex'\n")

if __name__ == '__main__':
    unittest.main()
