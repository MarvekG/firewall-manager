#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FRONTEND_DIR="$ROOT_DIR/frontend"
BACKEND_DIR="$ROOT_DIR/backend"
DIST_DIR="$BACKEND_DIR/internal/staticweb/dist"
OUT_DIR="$ROOT_DIR/dist"

mkdir -p "$OUT_DIR"

cd "$FRONTEND_DIR"
npm ci
npm run build

rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"
cp -R "$FRONTEND_DIR/dist/." "$DIST_DIR/"

cd "$BACKEND_DIR"
go build -o "$OUT_DIR/firewall-manager" ./cmd/firewall-manager

cp "$ROOT_DIR/scripts/install.sh" "$OUT_DIR/install.sh"
cp "$ROOT_DIR/scripts/uninstall.sh" "$OUT_DIR/uninstall.sh"
cp "$ROOT_DIR/scripts/reinstall.sh" "$OUT_DIR/reinstall.sh"
rm -rf "$OUT_DIR/deploy"
cp -R "$ROOT_DIR/deploy" "$OUT_DIR/deploy"

printf 'Release built at %s\n' "$OUT_DIR"
