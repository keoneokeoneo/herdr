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

"$root/scripts/build.sh"

# Every command the edits below depend on must work before anything is touched.
for probe in "codex -- true" "usage"; do
  # shellcheck disable=SC2086
  if ! "$bin" $probe >/dev/null 2>&1; then
    echo "abort: '$bin $probe' failed; nothing edited" >&2
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
  install -m 755 "$root/scripts/nvm-shim" "$shim"
  for n in node npm npx; do
    l="$HOME/.local/bin/$n"
    if [ ! -e "$l" ] && [ ! -L "$l" ] || [ "$(readlink "$l")" = "../libexec/nvm-shim" ]; then
      ln -sfn ../libexec/nvm-shim "$l"
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
  cat > "$plist" <<PL
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>Label</key><string>dev.keno.herdr-poke.usage</string>
  <key>ProgramArguments</key><array><string>$bin</string><string>usage</string></array>
  <key>EnvironmentVariables</key><dict>
    <key>PATH</key><string>$HOME/.local/bin:/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
  </dict>
  <key>StartInterval</key><integer>600</integer>
  <key>RunAtLoad</key><true/>
  <key>StandardErrorPath</key><string>$state/usage.log</string>
</dict></plist>
PL
  plutil -lint "$plist" >/dev/null
  launchctl bootout "gui/$(id -u)/dev.keno.herdr-poke.usage" 2>/dev/null || true
  launchctl bootstrap "gui/$(id -u)" "$plist" 2>/dev/null || { sleep 1; launchctl bootstrap "gui/$(id -u)" "$plist"; }
  echo "launchd dev.keno.herdr-poke.usage (every 10 min)"
  herdr-app do layout.switch --name poke >/dev/null 2>&1 && echo "herdr   layout → poke" || echo "warn    Herdr not running; pick the poke layout later (cmd-shift-l)"
fi

echo "done. Open a new zsh so the codex alias applies."
