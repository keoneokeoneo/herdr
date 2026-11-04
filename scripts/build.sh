#!/usr/bin/env bash
# Build the herdr-poke helper into ~/.local/bin/herdr-poke.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
mkdir -p "$HOME/.local/bin"
cd "$root/helper"
build_tmp="$(mktemp "$HOME/.local/bin/.herdr-poke-build.XXXXXX")"
trap 'rm -f "$build_tmp"' EXIT
go build -o "$build_tmp" .
mv -f "$build_tmp" "$HOME/.local/bin/herdr-poke"
echo "built $HOME/.local/bin/herdr-poke"
