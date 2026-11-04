#!/usr/bin/env bash
# Install herdr-poke into Keno's setup. Each edited file is backed up first, and the value
# each edit replaced is recorded in ~/.local/state/herdr-poke/install.json so
# scripts/uninstall.sh can undo exactly that. Safe to run twice.
# Close Claude Code sessions first: Claude rewrites ~/.claude/settings.json itself.
# HERDR_POKE_DRY=1 skips launchd and the running Herdr app (for testing with a fake HOME).
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
app="${HERDR_APP_HOME:-$HOME/.config/herdr-app}"
state="${HERDR_POKE_STATE:-$HOME/.local/state/herdr-poke}"
bin="$HOME/.local/bin/herdr-poke"
stamp="$(date +%Y%m%d%H%M%S)-$$"
dry="${HERDR_POKE_DRY:-0}"
export HERDR_APP_HOME="$app" HERDR_POKE_STATE="$state"

# Prerequisites must pass before journals, builds, backups or settings are written.
command -v python3 >/dev/null 2>&1 || { echo "abort: install Python 3 and expose python3 on PATH" >&2; exit 1; }
python3 "$root/scripts/preflight.py"

python3 "$root/scripts/install_files.py" prepare "$bin"
"$root/scripts/build.sh"
python3 "$root/scripts/install_files.py" record "$bin"

# Smoke-test the built helper before editing configuration (not a Codex CLI check).
for probe in "codex -- true" "usage"; do
  # shellcheck disable=SC2086
  if ! "$bin" $probe >/dev/null 2>&1; then
    echo "abort: '$bin $probe' failed; configuration edits not started (built helper is journaled)" >&2
    exit 1
  fi
done

for f in "$HOME/.claude/settings.json" "$HOME/.zshrc" "$app/settings.json"; do
  [ -f "$f" ] && cp -p "$f" "$f.herdr-poke-backup-$stamp"
done
python3 "$root/scripts/settings_edit.py" install "$bin"

"$bin" assets --companion || echo "warn    companion sprite not copied"

# node/npm/npx for programs the hc server starts: it has no login shell, so no nvm PATH
# (Claude hooks would fail with "node: command not found"). Only links that are missing
# or already ours are touched.
if [ -d "${NVM_DIR:-$HOME/.nvm}/versions/node" ]; then
  shim="$HOME/.local/libexec/nvm-shim"
  mkdir -p "$(dirname "$shim")" "$HOME/.local/bin"
  python3 "$root/scripts/install_files.py" prepare "$shim"
  install -m 755 "$root/scripts/nvm-shim" "$shim"
  python3 "$root/scripts/install_files.py" record "$shim"
  for n in node npm npx; do
    l="$HOME/.local/bin/$n"
    if [ ! -e "$l" ] && [ ! -L "$l" ] || [ "$(readlink "$l")" = "../libexec/nvm-shim" ]; then
      python3 "$root/scripts/install_files.py" prepare "$l"
      ln -sfn ../libexec/nvm-shim "$l"
      python3 "$root/scripts/install_files.py" record "$l"
    else
      echo "warn    $l exists and is not ours; left as is"
    fi
  done
  echo "shims   ~/.local/bin/{node,npm,npx} → nvm default"
fi

if [ "$dry" != 1 ]; then
  mkdir -p "$state"
  plist="$HOME/Library/LaunchAgents/dev.keno.herdr-poke.usage.plist"
  mkdir -p "$(dirname "$plist")"
  python3 "$root/scripts/install_files.py" prepare "$plist"
  python3 - "$plist" "$bin" "$state" <<'PYPLIST'
import os, plistlib, sys, tempfile
path, binary, state = sys.argv[1:]
job = {
    "Label": "dev.keno.herdr-poke.usage",
    "ProgramArguments": [binary, "usage"],
    "EnvironmentVariables": {
        "PATH": os.path.expanduser("~/.local/bin") + ":/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin",
        "HERDR_POKE_STATE": state,
    },
    "StartInterval": 600, "RunAtLoad": True,
    "StandardErrorPath": os.path.join(state, "usage.log"),
}
fd, tmp = tempfile.mkstemp(dir=os.path.dirname(path), prefix=".herdr-poke-plist-")
with os.fdopen(fd, "wb") as f:
    plistlib.dump(job, f)
os.replace(tmp, path)
PYPLIST
  plutil -lint "$plist" >/dev/null
  python3 "$root/scripts/install_files.py" record "$plist"
  launchctl bootout "gui/$(id -u)/dev.keno.herdr-poke.usage" 2>/dev/null || true
  launchctl bootstrap "gui/$(id -u)" "$plist" 2>/dev/null || { sleep 1; launchctl bootstrap "gui/$(id -u)" "$plist"; }
  echo "launchd dev.keno.herdr-poke.usage (every 10 min)"
  herdr-app do layout.switch --name poke >/dev/null 2>&1 && echo "herdr   layout → poke" || echo "warn    Herdr not running; pick the poke layout later (cmd-shift-l)"
fi

echo "done. Open a new zsh so the codex alias applies."
