#!/usr/bin/env python3
"""Surgical, reversible edits for install.sh / uninstall.sh (the codex alias and PATH line; older installs also muted Herdr's sounds).

install:   apply each edit only if needed; record the value it replaced in the manifest.
uninstall: put back only what the manifest says this project changed, leaving every
           other change (Claude Code rewrites its settings.json often) as it is.
Writes are atomic (temp file + os.replace) and abort if the file changed under us.
"""
import json, os, re, sys, tempfile, fcntl

HOME = os.path.expanduser("~")
APP = os.environ.get("HERDR_APP_HOME") or os.path.join(HOME, ".config", "herdr-app")
STATE = os.environ.get("HERDR_POKE_STATE") or os.path.join(HOME, ".local", "state", "herdr-poke")
MANIFEST = os.path.join(STATE, "install.json")
CLAUDE = os.path.join(HOME, ".claude", "settings.json")
ZSHRC = os.path.join(HOME, ".zshrc")
HERDR = os.path.join(APP, "settings.json")
OLD_ALIAS = "alias codex='codex --yolo'"
PATH_LINE = 'export PATH="$HOME/.local/bin:$PATH" # herdr-poke'
NEW_ALIAS = "alias codex='herdr-poke codex -- codex --yolo --no-daemon'"


def read(path):
    with open(path, encoding="utf-8") as f:
        return f.read()


def write_atomic(path, text, expect):
    if (read(path) if os.path.exists(path) else None) != expect:
        sys.exit(f"abort: {path} changed while editing; run again")
    d = os.path.dirname(path)
    os.makedirs(d, exist_ok=True)
    fd, tmp = tempfile.mkstemp(dir=d, prefix=".herdr-poke-")
    with os.fdopen(fd, "w", encoding="utf-8") as f:
        f.write(text)
    os.chmod(tmp, (os.stat(path).st_mode & 0o777) if os.path.exists(path) else 0o600)
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
    except FileNotFoundError:
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
    # Validate every input before writing any configuration.
    raw_z = read(ZSHRC) if os.path.exists(ZSHRC) else None
    z = raw_z or ""
    aliases = re.findall(r"^[ \t]*alias[ \t]+codex=.*$", z, re.M)
    if any(a not in (OLD_ALIAS, NEW_ALIAS) for a in aliases) or len(aliases) > 1:
        sys.exit("abort: custom or duplicate codex alias; resolve it before installation")
    # The global Claude statusLine is Keno's own and is never edited here (Poke no longer needs it).
    edits = []
    if NEW_ALIAS not in aliases:
        if OLD_ALIAS in aliases:
            z = z.replace(OLD_ALIAS, NEW_ALIAS, 1)
            m.setdefault("zshrc_alias", True)
        else:
            z = z + ("\n" if z and not z.endswith("\n") else "") + NEW_ALIAS + "\n"
            m.setdefault("zshrc_alias", "added")
        edits.append((ZSHRC, z, raw_z))
    if PATH_LINE not in z.splitlines():
        m.setdefault("zshrc_path", True)
        z = PATH_LINE + "\n" + z
        edits = [e for e in edits if e[0] != ZSHRC]
        edits.append((ZSHRC, z, raw_z))
    # Write-ahead recovery: a crash after any edit still retains the original values.
    save_manifest(m)
    for path, text, expected in edits:
        write_atomic(path, text, expected)
        print(f"edited  {path}")
    restore_sounds(m)
    save_manifest(m)


def restore_sounds(m):
    """Undo the mute older installs applied for the partner cries (now removed), if it is still ours."""
    if "herdr_sounds_enabled" not in m:
        return
    if os.path.exists(HERDR):
        try:
            raw, s = load_json(HERDR)
        except (ValueError, SystemExit):
            print(f"skipped {HERDR}: not valid JSON; sounds left as they are")
            return
        sounds = s.get("sounds") if isinstance(s, dict) else None
        if isinstance(sounds, dict) and sounds.get("enabled") is False:
            before = m["herdr_sounds_enabled"]
            if before is None:
                sounds.pop("enabled", None)
            else:
                sounds["enabled"] = before
            write_atomic(HERDR, dump(s), raw)
            print(f"restored {HERDR} sounds")
    del m["herdr_sounds_enabled"]


def uninstall():
    m = manifest()
    # Legacy: older installs replaced Claude's statusLine with `herdr-poke statusline`. Undo that only
    # if it is still ours; a statusLine the user has since replaced is left alone. The key is dropped either way.
    if "claude_statusLine" in m and os.path.exists(CLAUDE):
        raw, s = load_json(CLAUDE)
        if (s.get("statusLine") or {}).get("command", "").replace("\'", "").endswith("herdr-poke statusline"):
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
            write_atomic(ZSHRC, pat.sub("" if m["zshrc_alias"] == "added" else OLD_ALIAS, raw, count=1), raw)
            print(f"restored {ZSHRC} codex alias")
        del m["zshrc_alias"]
    if m.get("zshrc_path") and os.path.exists(ZSHRC):
        raw = read(ZSHRC)
        write_atomic(ZSHRC, raw.replace(PATH_LINE + "\n", "", 1), raw)
        del m["zshrc_path"]
    restore_sounds(m)
    save_manifest(m)


if __name__ == "__main__":
    os.makedirs(STATE, mode=0o700, exist_ok=True)
    with open(os.path.join(STATE, "settings-install.lock"), "a") as lock:
        fcntl.flock(lock, fcntl.LOCK_EX)
        {"install": lambda: install(sys.argv[2]), "uninstall": uninstall}[sys.argv[1]]()
