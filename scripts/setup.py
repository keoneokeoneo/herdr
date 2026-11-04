#!/usr/bin/env python3
"""Selective installer for this repository's setups (see scripts/setups.json).

  setup.py list
  setup.py install <name>... [--select <layout>] [--no-installer]
  setup.py status

Each setup lists the mods, layouts and themes it needs, the setups it requires, and the
`layout:<name>` key sections to merge. Installing a setup copies only those resources (nothing
is copied when the repo already is the app home), merges only missing key bindings (a
different existing binding is kept and reported), and records what it did in
~/.local/state/herdr-app/setups.json. It never edits ~/.zshrc, programs.json or settings.json;
a setup that needs that (Poke) names its own installer, which is run last.

Environment: HERDR_APP_HOME (default ~/.config/herdr-app), HERDR_SETUP_STATE (default
~/.local/state/herdr-app), HERDR_SETUP_DRY=1 (skip herdr-app and installers).
"""
import copy
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import time

ROOT = Path(__file__).resolve().parents[1]
SKIP_NAMES = {"jsconfig.json", ".DS_Store", "__pycache__"}


def load_manifest(path):
    return json.loads(Path(path).read_text(encoding="utf-8"))


def resolve(names, manifest):
    """Requested setups plus what they require, dependencies first, no duplicates."""
    order = []

    def visit(name, stack):
        if name not in manifest:
            raise ValueError(f"unknown setup: {name}")
        if name in stack:
            raise ValueError("setup cycle: " + " -> ".join(stack + [name]))
        if name in order:
            return
        for required in manifest[name].get("requires", []):
            visit(required, stack + [name])
        order.append(name)

    for name in names:
        visit(name, [])
    return order


def _skip(path):
    return path.name in SKIP_NAMES or path.name.endswith(".d.ts")


def setup_files(name, manifest, source):
    """Repo-relative files one setup owns: whole mod folders, layouts and themes."""
    entry, source, files = manifest[name], Path(source), []
    for mod in entry.get("mods", []):
        base = source / "mods" / mod
        files += [p.relative_to(source).as_posix() for p in sorted(base.rglob("*")) if p.is_file() and not _skip(p)
                  and not any(part in SKIP_NAMES for part in p.parts)]
    files += [f"layouts/{n}.json" for n in entry.get("layouts", [])]
    files += [f"themes/{n}.json" for n in entry.get("themes", [])]
    return files


def digest(path):
    p = Path(path)
    return hashlib.sha256(p.read_bytes()).hexdigest() if p.is_file() else None


def merge_keys(source_keys, app_keys, layouts):
    """Add missing `layout:<name>` bindings; return (merged, kept) where kept lists conflicts."""
    merged, kept = copy.deepcopy(app_keys), []
    for name in layouts:
        section = f"layout:{name}"
        want = source_keys.get(section, {})
        if not want:
            continue
        current = merged.setdefault(section, {})
        for key, action in want.items():
            if key not in current:
                current[key] = action
            elif current[key] != action:
                kept.append(f"{section} {key} = {current[key]} (kept; this repo wants {action})")
    return merged, kept


def _atomic_write(path, text):
    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    fd, tmp = tempfile.mkstemp(dir=path.parent, prefix=".setup-")
    with os.fdopen(fd, "w", encoding="utf-8") as f:
        f.write(text)
    os.replace(tmp, path)


def _dump(obj):
    return json.dumps(obj, indent=2, ensure_ascii=False) + "\n"


def _journal(state):
    p = Path(state) / "setups.json"
    return json.loads(p.read_text()) if p.exists() else {"installed": {}}


def install(names, source, app, state, manifest, dry=False, run_installers=True, select=None):
    source, app, state = Path(source).resolve(), Path(app).resolve(), Path(state)
    order = resolve(names, manifest)
    in_place = source == app
    stamp = time.strftime("%Y%m%d%H%M%S")
    copied, kept, journal = [], [], _journal(state)
    for name in order:
        fingerprints = {}
        for rel in setup_files(name, manifest, source):
            src, dst = source / rel, app / rel
            fingerprints[rel] = digest(src)
            if in_place or digest(src) == digest(dst):
                continue
            if dst.exists():
                shutil.copy2(dst, dst.with_name(dst.name + f".setup-backup-{stamp}"))
            dst.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(src, dst)
            copied.append(rel)
        journal["installed"][name] = {"files": fingerprints}
    layouts = [n for name in order for n in manifest[name].get("keys_layouts", [])]
    if layouts and not in_place:
        keys_src = json.loads((source / "keys.json").read_text(encoding="utf-8"))
        keys_path = app / "keys.json"
        current = json.loads(keys_path.read_text(encoding="utf-8")) if keys_path.exists() else {}
        merged, kept = merge_keys(keys_src, current, layouts)
        if merged != current:
            if keys_path.exists():
                shutil.copy2(keys_path, keys_path.with_name(f"keys.json.setup-backup-{stamp}"))
            _atomic_write(keys_path, _dump(merged))
    Path(state).mkdir(parents=True, exist_ok=True)
    _atomic_write(Path(state) / "setups.json", _dump(journal))
    if not dry:
        for name in order:
            for mod in manifest[name].get("mods", []):
                subprocess.run(["herdr-app", "check", str(app / "mods" / mod)], check=False)
        if run_installers:
            for name in order:
                installer = manifest[name].get("installer")
                if installer:
                    subprocess.run(["bash", str(source / installer)], check=True)
        if select:
            subprocess.run(["herdr-app", "do", "layout.switch", json.dumps({"name": select})], check=False)
    return {"order": order, "copied": copied, "kept": kept}


def _paths():
    app = Path(os.environ.get("HERDR_APP_HOME") or Path.home() / ".config" / "herdr-app")
    state = Path(os.environ.get("HERDR_SETUP_STATE") or Path.home() / ".local" / "state" / "herdr-app")
    return app, state


def main(argv=None):
    argv = list(sys.argv[1:] if argv is None else argv)
    manifest = load_manifest(ROOT / "scripts" / "setups.json")
    app, state = _paths()
    command = argv.pop(0) if argv else "list"
    if command == "list":
        installed = _journal(state)["installed"]
        for name, entry in manifest.items():
            needs = f" (requires {', '.join(entry['requires'])})" if entry.get("requires") else ""
            print(f"{name:10} {'installed' if name in installed else '-':10} {entry['title']}{needs}")
        return 0
    if command == "status":
        for name, entry in _journal(state)["installed"].items():
            changed = [rel for rel, h in entry["files"].items() if digest(app / rel) != h]
            print(f"{name}: " + ("ok" if not changed else "modified or missing: " + ", ".join(changed)))
        return 0
    if command == "install":
        select = None
        run_installers = True
        names = []
        while argv:
            a = argv.pop(0)
            if a == "--select":
                select = argv.pop(0) if argv else None
            elif a == "--no-installer":
                run_installers = False
            else:
                names.append(a)
        if not names:
            print("usage: setup.py install <name>... [--select <layout>] [--no-installer]", file=sys.stderr)
            return 2
        try:
            summary = install(names, ROOT, app, state, manifest, dry=os.environ.get("HERDR_SETUP_DRY") == "1",
                              run_installers=run_installers, select=select)
        except ValueError as e:
            print(f"error: {e}", file=sys.stderr)
            return 2
        print("installed: " + ", ".join(summary["order"]))
        for rel in summary["copied"]:
            print(f"copied   {rel}")
        for line in summary["kept"]:
            print(f"kept     {line}")
        return 0
    print("usage: setup.py list | install <name>... | status", file=sys.stderr)
    return 2


if __name__ == "__main__":
    sys.exit(main())
