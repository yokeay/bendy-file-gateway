#!/usr/bin/env bash
set -euo pipefail

# Build Go backend to WASM using TinyGo
# Output: platforms/wasm/gateway.wasm

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
WASM_OUT_DIR="$PROJECT_ROOT/platforms/wasm"
GO_SRC="$PROJECT_ROOT/cmd/gateway"

echo "Building Go WASM module..."
echo "  Source: $GO_SRC"
echo "  Output: $WASM_OUT_DIR/gateway.wasm"

mkdir -p "$WASM_OUT_DIR"

# TinyGo build for WASM
# -target wasm produces a standard WASI module
# -no-debug strips DWARF sections for smaller binary
# -panic=trap converts panics to traps
tinygo build \
  -target wasm \
  -no-debug \
  -panic=trap \
  -o "$WASM_OUT_DIR/gateway.wasm" \
  "$GO_SRC"

WASM_SIZE=$(wc -c < "$WASM_OUT_DIR/gateway.wasm" | tr -d ' ')
echo "Build complete: $WASM_OUT_DIR/gateway.wasm ($WASM_SIZE bytes)"

# Copy to platform directories
if [ -d "$PROJECT_ROOT/platforms/cloudflare" ]; then
  mkdir -p "$PROJECT_ROOT/platforms/cloudflare/wasm"
  cp "$WASM_OUT_DIR/gateway.wasm" "$PROJECT_ROOT/platforms/cloudflare/wasm/gateway.wasm"
  echo "Copied to platforms/cloudflare/wasm/"
fi

if [ -d "$PROJECT_ROOT/platforms/vercel" ]; then
  mkdir -p "$PROJECT_ROOT/platforms/vercel/wasm"
  cp "$WASM_OUT_DIR/gateway.wasm" "$PROJECT_ROOT/platforms/vercel/wasm/gateway.wasm"
  echo "Copied to platforms/vercel/wasm/"
fi
