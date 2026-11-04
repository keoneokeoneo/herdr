#!/usr/bin/env bash
set -euo pipefail
source_root="$(cd "$(dirname "$0")/../.." && pwd)"
app_home="${HERDR_APP_HOME:-$HOME/.config/herdr-app}"
mkdir -p "$app_home/mods/obsidian" "$app_home/layouts" "$app_home/themes"
for relative in mods/obsidian/tree.js mods/obsidian/main.js mods/obsidian/mod.json mods/obsidian/vault.py layouts/obsidian.json themes/base.json; do
  if [ "$source_root" != "$(cd "$app_home" && pwd)" ]; then
    if [ -f "$app_home/$relative" ] && ! cmp -s "$source_root/$relative" "$app_home/$relative"; then
      cp "$app_home/$relative" "$app_home/$relative.obsidian-backup-$(date +%Y%m%d%H%M%S)"
    fi
    cp "$source_root/$relative" "$app_home/$relative"
  fi
done
herdr-app check "$app_home/mods/obsidian"
