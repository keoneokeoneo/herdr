#!/usr/bin/env bash
# Regenerate golden-*.txt by running the real ~/.claude/statusline.mjs (Node) on
# every in-*.json, in a hermetic fake HOME. The Go port must reproduce these
# bytes exactly (see TestRenderMatchesGolden).
#
# Each fixture may have a sidecar in-NAME.env with KEY=VALUE lines:
#   CLAUDE_CODE_AUTO_COMPACT_WINDOW=<n>   exported to the script
#   ACCOUNT_FILE=<file in testdata/>      copied to $HOME/.claude.json
#                                         ("-" = no .claude.json); default claude.json
# "__HOME__" inside a fixture is replaced by the fake home. The script runs with
# cwd = $HOME/repos/nogit (what `process.cwd()` falls back to for in-empty.json).
#
# Usage: gen-golden.sh   (STATUSLINE_MJS overrides the script path)
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
script="${STATUSLINE_MJS:-$HOME/.claude/statusline.mjs}"
now_ms=1793700000000   # keep in sync with testNow in statusline_test.go

tmp="$(mktemp -d)"
tmp="$(cd "$tmp" && pwd -P)"   # physical path: process.cwd() resolves symlinks
trap 'rm -rf "$tmp"' EXIT
fake_home="$tmp/home"
mkdir -p "$fake_home/repos/nogit"

# Hermetic git: no user/system config, never discover a repo above the fake home.
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1 GIT_CEILING_DIRECTORIES="$fake_home"
export GIT_AUTHOR_NAME=t GIT_AUTHOR_EMAIL=t@example.com GIT_COMMITTER_NAME=t GIT_COMMITTER_EMAIL=t@example.com

mkrepo() { # name branch
  local d="$fake_home/repos/$1"
  mkdir -p "$d"
  git -C "$d" init -q -b "$2"
  : > "$d/file.txt"
  git -C "$d" add file.txt
  git -C "$d" commit -q -m init
}
mkrepo clean main
mkrepo dirty feat/x
echo change > "$fake_home/repos/dirty/untracked.txt"

for in in "$here"/in-*.json; do
  name="$(basename "$in" .json)"; name="${name#in-}"
  envfile="$here/in-$name.env"
  account=claude.json
  declare -a extra=()
  if [ -f "$envfile" ]; then
    while IFS='=' read -r k v; do
      [ -z "$k" ] && continue
      case "$k" in
        ACCOUNT_FILE) account="$v" ;;
        *) extra+=("$k=$v") ;;
      esac
    done < "$envfile"
  fi

  rm -f "$fake_home/.claude.json"
  [ "$account" = "-" ] || cp "$here/$account" "$fake_home/.claude.json"

  sed "s|__HOME__|$fake_home|g" "$in" > "$tmp/stdin.json"
  ( cd "$fake_home/repos/nogit" &&
    env -i PATH="$PATH" HOME="$fake_home" FAKE_NOW_MS="$now_ms" \
      GIT_CONFIG_GLOBAL="$GIT_CONFIG_GLOBAL" GIT_CONFIG_NOSYSTEM=1 GIT_CEILING_DIRECTORIES="$fake_home" \
      ${extra[@]+"${extra[@]}"} \
      node -r "$here/fake-now.cjs" "$script" < "$tmp/stdin.json" > "$here/golden-$name.txt" )
  echo "golden-$name.txt"
done
