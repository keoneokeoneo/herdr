#!/usr/bin/env bash
# Installs the Obsidian setup (mod, layout and the neutral theme). Same as: python3 scripts/setup.py install obsidian
set -euo pipefail
exec python3 "$(cd "$(dirname "$0")/../.." && pwd)/scripts/setup.py" install obsidian "$@"
