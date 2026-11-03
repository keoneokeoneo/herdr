#!/usr/bin/env python3
"""Surgical, reversible edits for install.sh / uninstall.sh.

install:   apply each edit only if needed; record the value it replaced in the manifest.
uninstall: put back only what the manifest says this project changed, leaving every
           other change (Claude Code rewrites its settings.json often) as it is.
Writes are atomic (temp file + os.replace) and abort if the file changed under us.
"""
import json, os, re, sys, tempfile

HOME = os.path.expanduser("~")
APP = os.environ.get("HERDR_APP_HOME") or os.path.join(HOME, ".config", "herdr-app")
STATE = os.environ.get("HERDR_POKE_STATE") or os.path.join(HOME, ".local", "state", "herdr-poke")
MANIFEST = os.path.join(STATE, "install.json")
CLAUDE = os.path.join(HOME, ".claude", "settings.json")
ZSHRC = os.path.join(HOME, ".zshrc")
HERDR = os.path.join(APP, "settings.json")
OLD_ALIAS = "alias codex='codex --yolo'"
NEW_ALIAS = "alias codex='herdr-poke codex -- codex --yolo --no-daemon'"


def read(path):
    with open(path, encoding="utf-8") as f:
        return f.read()


def write_atomic(path, text, expect):
    if read(path) != expect:
        sys.exit(f"abort: {path} changed while editing; run again")
    d = os.path.dirname(path)
    fd, tmp = tempfile.mkstemp(dir=d, prefix=".herdr-poke-")
    with os.fdopen(fd, "w", encoding="utf-8") as f:
        f.write(text)
    os.chmod(tmp, os.stat(path).st_mode & 0o777)
    if os.path.islink(path):
        path = os.path.realpath(path)
    os.replace(tmp, path)


def load_json(path):
    raw = read(path)
    try:
        return raw, json.loads(raw)
    except ValueError as e:
        sys.exit(f"abort: {path} is not valid JSON ({e}); nothing edited")


def dump(obj):
    return json.dumps(obj, indent=2, ensure_ascii=False) + "\n"


def manifest():
    try:
        return json.loads(read(MANIFEST))
    except (OSError, ValueError):
        return {}


def save_manifest(m):
    os.makedirs(STATE, mode=0o700, exist_ok=True)
    if m:
        with open(MANIFEST + ".tmp", "w", encoding="utf-8") as f:
            f.write(dump(m))
        os.replace(MANIFEST + ".tmp", MANIFEST)
    elif os.path.exists(MANIFEST):
        os.remove(MANIFEST)


def install(binary):
    m = manifest()
    want = f"{binary} statusline"
    if os.path.exists(CLAUDE):
        raw, s = load_json(CLAUDE)
        if (s.get("statusLine") or {}).get("command") != want:
            m.setdefault("claude_statusLine", s.get("statusLine"))  # keep the first "before"
            s["statusLine"] = {**(s.get("statusLine") or {}), "type": "command", "command": want}
            write_atomic(CLAUDE, dump(s), raw)
            print(f"edited  {CLAUDE} (statusLine)")
    if os.path.exists(ZSHRC):
        raw = read(ZSHRC)
        pat = re.compile("^" + re.escape(OLD_ALIAS) + "$", re.M)
        if pat.search(raw):
            m["zshrc_alias"] = True
            write_atomic(ZSHRC, pat.sub(NEW_ALIAS, raw, count=1), raw)
            print(f"edited  {ZSHRC} (codex alias)")
    if os.path.exists(HERDR):
        raw, s = load_json(HERDR)
        cur = (s.get("sounds") or {}).get("enabled")
        if cur is not False:
            m.setdefault("herdr_sounds_enabled", cur)
            s.setdefault("sounds", {})["enabled"] = False
            write_atomic(HERDR, dump(s), raw)
            print(f"edited  {HERDR} (sounds off)")
    save_manifest(m)


def uninstall():
    m = manifest()
    if "claude_statusLine" in m and os.path.exists(CLAUDE):
        raw, s = load_json(CLAUDE)
        if (s.get("statusLine") or {}).get("command", "").endswith("herdr-poke statusline"):
            if m["claude_statusLine"] is None:
                s.pop("statusLine", None)
            else:
                s["statusLine"] = m["claude_statusLine"]
            write_atomic(CLAUDE, dump(s), raw)
            print(f"restored {CLAUDE} statusLine")
        del m["claude_statusLine"]
    if m.get("zshrc_alias") and os.path.exists(ZSHRC):
        raw = read(ZSHRC)
        pat = re.compile("^" + re.escape(NEW_ALIAS) + "$", re.M)
        if pat.search(raw):
            write_atomic(ZSHRC, pat.sub(OLD_ALIAS, raw, count=1), raw)
            print(f"restored {ZSHRC} codex alias")
        del m["zshrc_alias"]
    if "herdr_sounds_enabled" in m and os.path.exists(HERDR):
        raw, s = load_json(HERDR)
        before = m["herdr_sounds_enabled"]
        if before is None:
            (s.get("sounds") or {}).pop("enabled", None)
        else:
            s.setdefault("sounds", {})["enabled"] = before
        write_atomic(HERDR, dump(s), raw)
        print(f"restored {HERDR} sounds")
        del m["herdr_sounds_enabled"]
    save_manifest(m)


if __name__ == "__main__":
    {"install": lambda: install(sys.argv[2]), "uninstall": uninstall}[sys.argv[1]]()
