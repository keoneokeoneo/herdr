#!/usr/bin/env python3
"""Journal installer-owned files; never delete a later user replacement.

prepare saves the original before a write, record fingerprints the installed result,
restore restores originals only when the current file still matches our result.
"""
import hashlib
import json
import os
from pathlib import Path
import shutil
import sys
import tempfile
import fcntl

STATE = Path(os.environ.get('HERDR_POKE_STATE') or Path.home() / '.local/state/herdr-poke')
MANIFEST = STATE / 'install-files.json'


def fingerprint(path):
    p = Path(path)
    if p.is_symlink():
        return 'link:' + os.readlink(p)
    if not p.exists():
        return None
    return hashlib.sha256(p.read_bytes()).hexdigest()


def load():
    return json.loads(MANIFEST.read_text()) if MANIFEST.exists() else {}


def save(entries):
    STATE.mkdir(parents=True, exist_ok=True)
    fd, tmp = tempfile.mkstemp(dir=STATE, prefix='.install-files-')
    with os.fdopen(fd, 'w') as f:
        json.dump(entries, f, indent=2)
    os.replace(tmp, MANIFEST)


def prepare(path):
    path = str(Path(path).absolute())
    entries = load()
    if path in entries:
        expected = entries[path].get('installed')
        if expected is not None and fingerprint(path) != expected:
            raise RuntimeError(f'{path} changed since installation; refusing to overwrite')
        return
    before = fingerprint(path)
    backup = None
    if before is not None:
        backup = str(STATE / ('install-original-' + hashlib.sha256(path.encode()).hexdigest()))
        shutil.copy2(path, backup, follow_symlinks=False)
    entries[path] = {'before': before, 'backup': backup, 'installed': None}
    save(entries)


def record(path):
    path = str(Path(path).absolute())
    entries = load()
    entries[path]['installed'] = fingerprint(path)
    save(entries)


def owned(path):
    entry = load().get(str(Path(path).absolute()))
    return bool(entry and entry['installed'] is not None and fingerprint(path) == entry['installed'])


def restore():
    entries = load()
    for path, entry in list(entries.items()):
        current = fingerprint(path)
        if current == entry['before']:
            del entries[path]
            save(entries)
            continue
        if entry['installed'] is None or current != entry['installed']:
            print(f'kept    {path} (changed or interrupted install; original backup retained)')
            continue
        p = Path(path)
        if entry['backup']:
            # Prepare beside the target, then replace atomically (including symlinks).
            tmp = p.with_name('.herdr-poke-restore-' + str(os.getpid()))
            shutil.copy2(entry['backup'], tmp, follow_symlinks=False)
            os.replace(tmp, p)
        else:
            p.unlink()
        del entries[path]
        save(entries)


if __name__ == '__main__':
    STATE.mkdir(parents=True, exist_ok=True)
    with (STATE / 'install-files.lock').open('a') as lock:
        fcntl.flock(lock, fcntl.LOCK_EX)
        action = sys.argv[1]
        if action == 'restore': restore()
        elif action == 'owned': sys.exit(0 if owned(sys.argv[2]) else 1)
        elif action == 'prepare': prepare(sys.argv[2])
        elif action == 'record': record(sys.argv[2])
        else: raise SystemExit('unknown action')
