#!/usr/bin/env bash
# Build the herdr-poke helper into ~/.local/bin/herdr-poke.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
mkdir -p "$HOME/.local/bin"
cd "$root/helper"
go build -o "$HOME/.local/bin/herdr-poke" .
echo "built $HOME/.local/bin/herdr-poke"
