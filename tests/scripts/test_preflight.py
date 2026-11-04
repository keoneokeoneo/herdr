import importlib.util
import os
from pathlib import Path
import subprocess
import sys
import shutil
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2]


class PreflightTests(unittest.TestCase):
    def setUp(self):
        spec = importlib.util.spec_from_file_location('preflight', ROOT/'scripts/preflight.py')
        self.m = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(self.m)
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.home = Path(self.tmp.name)

    def check(self, outputs=None, missing=()):
        outputs = outputs or {}
        def run(args, **kwargs):
            if args[0] == 'go':
                self.assertEqual(kwargs['env']['GOTOOLCHAIN'], 'local')
            value = outputs.get(tuple(args), '--no-daemon' if args == ['codex', '--help'] else 'go version go1.27.3 darwin/arm64')
            if isinstance(value, Exception):
                raise value
            if isinstance(value, int):
                return subprocess.CompletedProcess(args, value, '', '')
            return subprocess.CompletedProcess(args, 0, value, '')
        with patch.dict(os.environ, {'NVM_DIR': str(self.home/'.nvm')}), patch.object(self.m.shutil, 'which', side_effect=lambda c: None if c in missing else '/tools/'+c), patch.object(self.m.subprocess, 'run', side_effect=run), patch.object(self.m.platform, 'system', return_value='Darwin'):
            return self.m.check(ROOT, self.home)

    def test_missing_tools_are_actionable(self):
        errors, _ = self.check(missing=['claude', 'hc'])
        self.assertTrue(any('claude' in e for e in errors))
        self.assertTrue(any('hc' in e for e in errors))

    def test_old_go_and_unsupported_codex_fail(self):
        errors, _ = self.check({('go', 'version'): 'go version go1.26.9 darwin/arm64', ('codex', '--help'): 'Usage: codex'})
        self.assertTrue(any('1.27.3' in e for e in errors))
        self.assertTrue(any('--no-daemon' in e for e in errors))

    def test_optional_features_do_not_block(self):
        errors, warnings = self.check(missing=['node', 'npm', 'npx'])
        self.assertEqual(errors, [])
        self.assertTrue(any('Node' in w for w in warnings))
        self.assertTrue(any('Jetendard' in w for w in warnings))
        self.assertTrue(any('pokeTokenBar' in w for w in warnings))
        self.assertEqual(list(self.home.iterdir()), [])

    def test_timeout_and_failed_command_fail(self):
        errors, _ = self.check({('claude', '--version'): subprocess.TimeoutExpired('claude', 10)})
        self.assertTrue(any('claude' in e for e in errors))
        errors, _ = self.check({('claude', '--version'): 1})
        self.assertTrue(any('claude' in e for e in errors))

    def test_nvm_absent_with_node_is_supported(self):
        errors, warnings = self.check()
        self.assertEqual(errors, [])
        self.assertTrue(any('nvm' in w for w in warnings))

    def test_install_failure_does_not_write_home(self):
        tools = self.home/'tools'
        tools.mkdir()
        python = tools/'python3'
        python.symlink_to(sys.executable)
        for name in ['dirname', 'date']:
            (tools/name).symlink_to(shutil.which(name))
        for name in ['herdr-app', 'hc', 'go', 'claude', 'git', 'zsh', 'lsof', 'launchctl', 'plutil']:
            tool = tools/name
            tool.write_text('#!/bin/sh\necho "go version go1.27.3 darwin/arm64"\n')
            tool.chmod(0o755)
        (self.home/'.zshrc').write_text('# original\n')
        before = sorted(str(p.relative_to(self.home)) for p in self.home.rglob('*'))
        result = subprocess.run(['/bin/bash', str(ROOT/'scripts/install.sh')], env={**os.environ, 'HOME': str(self.home), 'PATH': str(tools), 'HERDR_APP_HOME': str(self.home/'app'), 'HERDR_POKE_STATE': str(self.home/'state')}, capture_output=True, text=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('codex is missing', result.stderr)
        self.assertEqual((self.home/'.zshrc').read_text(), '# original\n')
        self.assertEqual(before, sorted(str(p.relative_to(self.home)) for p in self.home.rglob('*')))
        codex = tools/'codex'
        codex.write_text('#!/bin/sh\necho "Usage: codex"\n')
        codex.chmod(0o755)
        before = sorted(str(p.relative_to(self.home)) for p in self.home.rglob('*'))
        result = subprocess.run(['/bin/bash', str(ROOT/'scripts/install.sh')], env={**os.environ, 'HOME': str(self.home), 'PATH': str(tools), 'HERDR_APP_HOME': str(self.home/'app'), 'HERDR_POKE_STATE': str(self.home/'state')}, capture_output=True, text=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('does not advertise --no-daemon', result.stderr)
        self.assertEqual(before, sorted(str(p.relative_to(self.home)) for p in self.home.rglob('*')))
        self.assertEqual((self.home/'.zshrc').read_text(), '# original\n')
