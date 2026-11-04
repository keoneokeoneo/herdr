#!/usr/bin/env bash
set -euo pipefail
source_root="$(cd "$(dirname "$0")/../.." && pwd)"
app_home="${HERDR_APP_HOME:-$HOME/.config/herdr-app}"
mkdir -p "$app_home/mods/obsidian" "$app_home/layouts" "$app_home/themes"
for relative in mods/obsidian/tree.js mods/obsidian/main.js mods/obsidian/mod.json mods/obsidian/vault.py layouts/obsidian.json themes/poke.json; do
  if [ "$source_root" != "$(cd "$app_home" && pwd)" ]; then
    if [ -f "$app_home/$relative" ] && ! cmp -s "$source_root/$relative" "$app_home/$relative"; then
      cp "$app_home/$relative" "$app_home/$relative.obsidian-backup-$(date +%Y%m%d%H%M%S)"
    fi
    cp "$source_root/$relative" "$app_home/$relative"
  fi
done
if [ "${1:-}" = "--poke" ]; then
  python3 - "$source_root" "$app_home" <<'PYINSTALL'
import json, shutil, sys, time
from pathlib import Path
source, home = map(Path, sys.argv[1:])
stamp = str(time.time_ns())
backup = home / 'data/obsidian/backups' / stamp
for relative in ['layouts/poke.json', 'mods/poke/views/tabs.js', 'mods/poke/views/paneltabs.js', 'mods/poke/views/status.js', 'keys.json']:
    destination = home / relative
    if destination.exists():
        target = backup / relative
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(destination, target)
    if relative == 'keys.json':
        data = json.loads(destination.read_text()) if destination.exists() else {'global': {}}
        keys = data.setdefault('global', {})
        binding = keys.get('cmd-shift-o')
        if binding and binding not in ('layout.toggle:poke-notes', 'poke.notes'):
            print('Existing Cmd-Shift-O preserved; use the Obsidian tab.')
        else:
            keys['cmd-shift-o'] = 'poke.notes'
        destination.write_text(json.dumps(data, indent=2) + '\n')
    elif source.resolve() != home.resolve():
        destination.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(source / relative, destination)
PYINSTALL
  herdr-app check "$app_home/mods/poke"
fi
herdr-app check "$app_home/mods/obsidian"
