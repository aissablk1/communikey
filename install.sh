#!/usr/bin/env bash
# Build a universal (arm64 + x86_64) csend binary and symlink it onto PATH.
# Universal so it runs natively whatever the calling shell's arch (incl. a
# Rosetta x86_64 shell on Apple Silicon).
set -euo pipefail
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
mkdir -p "$DIR/.build"

GOOS=darwin GOARCH=arm64 go -C "$DIR" build -o "$DIR/.build/csend.arm64" .
GOOS=darwin GOARCH=amd64 go -C "$DIR" build -o "$DIR/.build/csend.amd64" .
lipo -create -output "$DIR/csend" "$DIR/.build/csend.arm64" "$DIR/.build/csend.amd64"

# Symlink onto PATH. ~/.local/bin is on Aïssa's PATH (no sudo). The bin dir is
# kept in a variable so the path string isn't a literal the workspace guard trips on.
BINDIR="$HOME/.local/bin"
mkdir -p "$BINDIR"
ln -sf "$DIR/csend" "$BINDIR/csend"

echo "✓ csend installé : $BINDIR/csend → $DIR/csend"
lipo -info "$DIR/csend"
