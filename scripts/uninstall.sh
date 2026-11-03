#!/usr/bin/env bash
# Undo scripts/install.sh: restore the newest backup of each edited file,
# remove the launchd job and the helper binary, switch Herdr back to main.
# The repo itself stays. HERDR_POKE_DRY=1 skips launchd and Herdr.
set -euo pipefail

app="${HERDR_APP_HOME:-$HOME/.config/herdr-app}"
dry="${HERDR_POKE_DRY:-0}"

restore() {
  local newest
  newest="$(ls -1 "$1".herdr-poke-backup-* 2>/dev/null | sort | tail -1 || true)"
  if [ -n "$newest" ]; then cp -p "$newest" "$1"; echo "restored $1 ← $newest"; fi
}

restore "$HOME/.claude/settings.json"
restore "$HOME/.zshrc"
restore "$app/settings.json"

if [ "$dry" != 1 ]; then
  launchctl bootout "gui/$(id -u)/dev.keno.herdr-poke.usage" 2>/dev/null || true
  rm -f "$HOME/Library/LaunchAgents/dev.keno.herdr-poke.usage.plist"
  herdr-app do layout.switch --name main >/dev/null 2>&1 || true
fi
rm -f "$HOME/.local/bin/herdr-poke"
echo "done. Backups are kept next to each file."
