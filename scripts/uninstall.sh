#!/usr/bin/env bash
# Undo scripts/install.sh: put back only the values install recorded as changed
# (other edits made since are kept), and restore journaled installed files.
# The repo, layout and backups stay. HERDR_POKE_DRY=1 skips launchd.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dry="${HERDR_POKE_DRY:-0}"

python3 "$root/scripts/settings_edit.py" uninstall

plist="$HOME/Library/LaunchAgents/dev.keno.herdr-poke.usage.plist"
restore_job=0
if [ "$dry" != 1 ] && python3 "$root/scripts/install_files.py" owned "$plist"; then
  restore_job=1
  launchctl bootout "gui/$(id -u)/dev.keno.herdr-poke.usage" 2>/dev/null || true
fi
python3 "$root/scripts/install_files.py" restore
if [ "$restore_job" = 1 ] && [ -f "$plist" ]; then
  launchctl bootstrap "gui/$(id -u)" "$plist" || echo "warn    restored usage job could not be loaded" >&2
fi
# The layout belongs to the user; the repo remains usable, so do not switch it.
echo "done. Backups, runtime state and assets are kept. Unjournaled files are left alone."
