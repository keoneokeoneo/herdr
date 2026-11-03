#!/usr/bin/env bash
# Undo scripts/install.sh: put back only the values install recorded as changed
# (other edits made since are kept), remove the launchd job and the binary, and
# switch Herdr back to main. The repo and the backups stay. HERDR_POKE_DRY=1 skips launchd and Herdr.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dry="${HERDR_POKE_DRY:-0}"

python3 "$root/scripts/settings_edit.py" uninstall

if [ "$dry" != 1 ]; then
  launchctl bootout "gui/$(id -u)/dev.keno.herdr-poke.usage" 2>/dev/null || true
  rm -f "$HOME/Library/LaunchAgents/dev.keno.herdr-poke.usage.plist"
  herdr-app do layout.switch --name main >/dev/null 2>&1 || true
fi
rm -f "$HOME/.local/bin/herdr-poke"
for n in node npm npx; do
  [ "$(readlink "$HOME/.local/bin/$n" 2>/dev/null)" = "../libexec/nvm-shim" ] && rm -f "$HOME/.local/bin/$n"
done
rm -f "$HOME/.local/libexec/nvm-shim"
echo "done. Backups (*.herdr-poke-backup-*) are kept next to each file."
