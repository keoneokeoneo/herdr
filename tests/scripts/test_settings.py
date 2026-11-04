import importlib.util
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

SCRIPT = Path(__file__).resolve().parents[2] / 'scripts/settings_edit.py'

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

    def test_fresh_repeat_and_undo(self):
        self.m.install(self.binary)
        self.m.install(self.binary)
        self.assertIn(self.m.NEW_ALIAS, Path(self.m.ZSHRC).read_text())
        self.assertEqual(Path(self.m.ZSHRC).read_text().count(self.m.NEW_ALIAS), 1)
        self.assertEqual(json.loads(Path(self.m.CLAUDE).read_text())['statusLine']['type'], 'command')
        self.m.uninstall()
        self.m.uninstall()
        self.assertNotIn(self.m.NEW_ALIAS, Path(self.m.ZSHRC).read_text())
        self.assertNotIn('statusLine', json.loads(Path(self.m.CLAUDE).read_text()))

    def test_invalid_json_preflight(self):
        Path(self.m.HERDR).parent.mkdir(parents=True)
        Path(self.m.HERDR).write_text('{broken')
        with self.assertRaises((SystemExit, ValueError)):
            self.m.install(self.binary)
        self.assertFalse(Path(self.m.CLAUDE).exists())

    def test_failure_after_first_write_keeps_original(self):
        p = Path(self.m.CLAUDE)
        p.parent.mkdir(parents=True)
        original = {'type': 'command', 'command': 'my-status'}
        p.write_text(json.dumps({'statusLine': original}))
        real = self.m.write_atomic
        def fail(path, text, expect):
            if path == self.m.ZSHRC:
                raise OSError('injected interruption')
            return real(path, text, expect)
        with patch.object(self.m, 'write_atomic', side_effect=fail):
            with self.assertRaises(OSError):
                self.m.install(self.binary)
        self.m.install(self.binary)
        self.m.uninstall()
        self.assertEqual(json.loads(p.read_text())['statusLine'], original)

    def test_custom_alias_is_not_overwritten(self):
        Path(self.m.ZSHRC).write_text("alias codex='custom-codex'\n")
        with self.assertRaises(SystemExit):
            self.m.install(self.binary)
        self.assertFalse(Path(self.m.CLAUDE).exists())

if __name__ == '__main__':
    unittest.main()
