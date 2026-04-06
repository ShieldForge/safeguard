#!/usr/bin/env bash
# generate_ui.sh — Builds the builder web UI into cmd/builder/static/.
# Called by go:generate from cmd/builder/main.go.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
UI_DIR="$SCRIPT_DIR/ui"

if [ ! -d "$UI_DIR/node_modules" ]; then
    echo "Installing UI dependencies..."
    (cd "$UI_DIR" && npm ci)
fi

echo "Building UI..."
(cd "$UI_DIR" && npm run build)
echo "UI build complete."
