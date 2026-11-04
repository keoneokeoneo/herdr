import importlib.util
from pathlib import Path
import tempfile
import unittest

SCRIPT = Path(__file__).resolve().parents[2] / 'scripts/install_files.py'

class FilesTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        spec = importlib.util.spec_from_file_location('files', SCRIPT)
        self.m = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(self.m)
        self.m.STATE = self.root / 'state'
        self.m.STATE.mkdir()
        self.m.MANIFEST = self.m.STATE / 'install-files.json'

    def test_repeat_restores_first_original(self):
        p = self.root / 'binary'
        p.write_bytes(b'original')
        self.m.prepare(p)
        p.write_bytes(b'v1')
        self.m.record(p)
        self.m.prepare(p)
        p.write_bytes(b'v2')
        self.m.record(p)
        self.m.restore()
        self.m.restore()
        self.assertEqual(p.read_bytes(), b'original')

    def test_new_link_removed_but_user_replacement_preserved(self):
        p = self.root / 'node'
        self.m.prepare(p)
        p.symlink_to('shim')
        self.m.record(p)
        self.m.restore()
        self.assertFalse(p.is_symlink())
        p.write_bytes(b'old')
        self.m.prepare(p)
        p.write_bytes(b'installed')
        self.m.record(p)
        p.write_bytes(b'user')
        self.m.restore()
        self.assertEqual(p.read_bytes(), b'user')
        with self.assertRaises(RuntimeError): self.m.prepare(p)

class InstallIntegrationTests(unittest.TestCase):
    def test_install_twice_uninstall_twice_in_isolated_home(self):
        import os
        import subprocess
        with tempfile.TemporaryDirectory() as tmp:
            home = Path(tmp)
            fakebin = home / 'tools'
            fakebin.mkdir()
            helper = home / 'fake-helper'
            helper.write_text('#!/bin/sh\nexit 0\n')
            helper.chmod(0o755)
            go = fakebin / 'go'
            go.write_text('#!/bin/sh\ncp "$FAKE_HELPER" "$3"\nchmod 755 "$3"\n')
            go.chmod(0o755)
            binary = home / '.local/bin/herdr-poke'
            binary.parent.mkdir(parents=True)
            binary.write_text('original binary')
            (home / '.nvm/versions/node').mkdir(parents=True)
            env = {**os.environ, 'HOME':tmp, 'HERDR_APP_HOME':str(home/'app'),
                   'HERDR_POKE_STATE':str(home/'state'), 'HERDR_POKE_DRY':'1',
                   'NVM_DIR':str(home/'.nvm'), 'FAKE_HELPER':str(helper),
                   'PATH':str(fakebin)+os.pathsep+os.environ['PATH']}
            root = SCRIPT.parents[1]
            for script in ['install.sh', 'install.sh', 'uninstall.sh', 'uninstall.sh']:
                result = subprocess.run(['bash', str(root/'scripts'/script)], env=env, capture_output=True, text=True)
                self.assertEqual(result.returncode, 0, result.stdout+result.stderr)
            self.assertEqual(binary.read_text(), 'original binary')
            self.assertFalse((home/'.local/bin/node').is_symlink())
            self.assertFalse((home/'.local/libexec/nvm-shim').exists())
