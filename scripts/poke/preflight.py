#!/usr/bin/env python3
"""Check local prerequisites without changing project or installation files.

Only version/help commands are executed: no server startup, login or agent session.
Go uses the local toolchain so this check cannot trigger a toolchain download.
"""
import os
from pathlib import Path
import platform
import re
import shutil
import subprocess
import sys


def output(args):
    result = subprocess.run(args, capture_output=True, text=True, timeout=10,
                            env={**os.environ, 'GOTOOLCHAIN': 'local'})
    if result.returncode:
        raise RuntimeError('exited with status ' + str(result.returncode))
    return result.stdout + result.stderr


def check(root, home):
    errors, warnings = [], []
    if platform.system() != 'Darwin':
        errors.append('This installer requires macOS (launchd and macOS asset paths).')
    required = {
        'herdr-app': 'Install Herdr and expose its CLI on PATH.',
        'hc': 'Install the Herdr core CLI and expose it on PATH.',
        'go': 'Install Go matching helpers/poke/go.mod or newer.',
        'claude': 'Install Claude Code and expose its executable on PATH.',
        'codex': 'Install a Codex CLI with --no-daemon support.',
        'git': 'Install Git / Command Line Tools.',
        'zsh': 'Install zsh for the menu launcher.',
        'lsof': 'Restore lsof on PATH for Codex rollout discovery.',
        'launchctl': 'Restore the macOS launchctl command on PATH.',
        'plutil': 'Restore the macOS plutil command on PATH.',
    }
    found = {name: shutil.which(name) for name in required}
    for name, hint in required.items():
        if not found[name]:
            errors.append(f'{name} is missing. {hint}')
    versions = {}
    for name in ('herdr-app', 'hc', 'go', 'claude', 'codex', 'git', 'zsh'):
        if not found[name]:
            continue
        try:
            versions[name] = output([name, 'version' if name == 'go' else '--version'])
        except (OSError, subprocess.TimeoutExpired, RuntimeError) as exc:
            errors.append(f'{name} cannot run ({exc}). {required[name]}')
    if 'go' in versions:
        needed = re.search(r'^go\s+(\d+\.\d+(?:\.\d+)?)$', (root/'helpers/poke/go.mod').read_text(), re.M).group(1)
        actual = re.search(r'\bgo(\d+\.\d+(?:\.\d+)?)\b', versions['go'])
        def version(v):
            parts = tuple(map(int, v.split('.')))
            return parts + (0,) * (3 - len(parts))
        if not actual or version(actual.group(1)) < version(needed):
            errors.append(f'Go {needed} or newer must be installed locally; see helpers/poke/go.mod.')
    if found['codex']:
        try:
            if not re.search(r'(?<![\w-])--no-daemon(?![\w-])', output(['codex', '--help'])):
                errors.append('Codex does not advertise --no-daemon. Install a compatible CLI before using the Poke launcher.')
        except (OSError, subprocess.TimeoutExpired, RuntimeError) as exc:
            errors.append(f'Cannot check Codex --no-daemon support ({exc}).')
    node_ok = True
    for name in ('node', 'npm', 'npx'):
        try:
            if not shutil.which(name):
                raise RuntimeError('not on PATH')
            output([name, '--version'])
        except (OSError, subprocess.TimeoutExpired, RuntimeError):
            node_ok = False
    if not node_ok:
        warnings.append('Node/npm/npx are missing or cannot run. Install them for Node-based hooks and MCP tools; Poke itself can install without them.')
    nvm = Path(os.environ.get('NVM_DIR') or home/'.nvm')
    if not any((nvm/'versions/node').glob('*/bin/node')):
        warnings.append('No installed nvm Node found. Non-nvm Node is supported, but verify node/npm/npx from a fresh Herdr terminal; managed shims need nvm.')
    if not (home/'Library/Application Support/PokeTokenBar/companion-state.json').exists():
        warnings.append('pokeTokenBar companion state is missing. Install/initialize or restore it for the companion and Dex.')
    font_dirs = [home/'Library/Fonts', Path('/Library/Fonts')]
    if not any('jetendard' in p.name.lower() for d in font_dirs if d.exists() for p in d.iterdir()):
        warnings.append('Jetendard font files were not found in the usual font folders. Install the terminal font or verify it in Font Book.')
    return errors, warnings


if __name__ == '__main__':
    errors, warnings = check(Path(__file__).resolve().parents[2], Path.home())
    for message in warnings:
        print('warn    ' + message)
    for message in errors:
        print('error   ' + message, file=sys.stderr)
    if errors:
        print('abort: prerequisites failed; installation writes have not started.', file=sys.stderr)
        sys.exit(1)
    print('preflight passed (login, fonts and live UI still need manual verification)')
