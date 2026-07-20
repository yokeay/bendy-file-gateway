#!/usr/bin/env bash
set -euo pipefail

# Build React admin dashboard
# Output: web/dist/

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
WEB_DIR="$PROJECT_ROOT/web"

echo "Building React admin dashboard..."
echo "  Source: $WEB_DIR"

cd "$WEB_DIR"

# Install dependencies if needed
if [ ! -d "node_modules" ]; then
  echo "Installing dependencies..."
  npm install
fi

# Build for production
npm run build

echo "Build complete: $WEB_DIR/dist/"

# Copy to platform directories
if [ -d "$PROJECT_ROOT/platforms/cloudflare" ]; then
  cp -r "$WEB_DIR/dist/"* "$PROJECT_ROOT/platforms/cloudflare/public/" 2>/dev/null || \
    (mkdir -p "$PROJECT_ROOT/platforms/cloudflare/public" && cp -r "$WEB_DIR/dist/"* "$PROJECT_ROOT/platforms/cloudflare/public/")
  echo "Copied to platforms/cloudflare/public/"
fi

if [ -d "$PROJECT_ROOT/platforms/vercel" ]; then
  cp -r "$WEB_DIR/dist/"* "$PROJECT_ROOT/platforms/vercel/public/" 2>/dev/null || \
    (mkdir -p "$PROJECT_ROOT/platforms/vercel/public" && cp -r "$WEB_DIR/dist/"* "$PROJECT_ROOT/platforms/vercel/public/")
  echo "Copied to platforms/vercel/public/"
fi
