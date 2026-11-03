#!/usr/bin/env bash
# Install herdr-poke into Keno's setup. Every file it edits is backed up first;
# scripts/uninstall.sh puts the newest backup back. Safe to run twice.
# HERDR_POKE_DRY=1 skips launchd and the running Herdr app (for testing with a fake HOME).
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
app="${HERDR_APP_HOME:-$HOME/.config/herdr-app}"
bin="$HOME/.local/bin/herdr-poke"
stamp="$(date +%Y%m%d%H%M%S)"
dry="${HERDR_POKE_DRY:-0}"

backup() { cp -p "$1" "$1.herdr-poke-backup-$stamp"; echo "backup  $1.herdr-poke-backup-$stamp"; }

"$root/scripts/build.sh"

# Claude Code statusline → herdr-poke statusline (other keys untouched).
cs="$HOME/.claude/settings.json"
if [ -f "$cs" ] && ! grep -q "herdr-poke statusline" "$cs"; then
  backup "$cs"
  python3 - "$cs" "$bin" <<'PY'
import json, sys
p, b = sys.argv[1], sys.argv[2]
s = json.load(open(p))
s["statusLine"] = {**s.get("statusLine", {}), "type": "command", "command": f"{b} statusline"}
open(p, "w").write(json.dumps(s, indent=2, ensure_ascii=False) + "\n")
PY
  echo "edited  $cs (statusLine)"
fi

# zsh: run Codex through the wrapper.
zr="$HOME/.zshrc"
if [ -f "$zr" ] && grep -q "^alias codex='codex --yolo'$" "$zr"; then
  backup "$zr"
  sed -i '' "s|^alias codex='codex --yolo'\$|alias codex='herdr-poke codex -- codex --yolo --no-daemon'|" "$zr"
  echo "edited  $zr (codex alias)"
fi

# Herdr: our cries replace its attention sounds.
hs="$app/settings.json"
if [ -f "$hs" ] && python3 -c 'import json,sys; sys.exit(0 if json.load(open(sys.argv[1])).get("sounds",{}).get("enabled") is not False else 1)' "$hs"; then
  backup "$hs"
  python3 - "$hs" <<'PY'
import json, sys
p = sys.argv[1]
s = json.load(open(p))
s.setdefault("sounds", {})["enabled"] = False
open(p, "w").write(json.dumps(s, indent=2) + "\n")
PY
  echo "edited  $hs (sounds off)"
fi

HERDR_APP_HOME="$app" "$bin" assets --companion || echo "warn    companion sprite not copied"
"$bin" usage || echo "warn    usage not collected yet"

if [ "$dry" != 1 ]; then
  plist="$HOME/Library/LaunchAgents/dev.keno.herdr-poke.usage.plist"
  mkdir -p "$(dirname "$plist")"
  cat > "$plist" <<PL
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>Label</key><string>dev.keno.herdr-poke.usage</string>
  <key>ProgramArguments</key><array><string>$bin</string><string>usage</string></array>
  <key>StartInterval</key><integer>600</integer>
  <key>RunAtLoad</key><true/>
  <key>StandardErrorPath</key><string>$HOME/.local/state/herdr-poke/usage.log</string>
</dict></plist>
PL
  launchctl bootout "gui/$(id -u)/dev.keno.herdr-poke.usage" 2>/dev/null || true
  launchctl bootstrap "gui/$(id -u)" "$plist"
  echo "launchd dev.keno.herdr-poke.usage (every 10 min)"
  herdr-app do layout.switch --name poke >/dev/null 2>&1 && echo "herdr   layout → poke" || echo "warn    Herdr not running; pick the poke layout later (cmd-shift-l)"
fi

echo "done. Open a new zsh so the codex alias applies."
