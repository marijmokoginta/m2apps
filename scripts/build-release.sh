#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist"

rm -rf "${DIST_DIR}"
mkdir -p "${DIST_DIR}/windows" "${DIST_DIR}/linux" "${DIST_DIR}/macos"

echo "Building Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -o "${DIST_DIR}/m2apps-windows-amd64.exe" "${ROOT_DIR}/main.go"
cp "${DIST_DIR}/m2apps-windows-amd64.exe" "${DIST_DIR}/windows/m2apps.exe"

echo "Building Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -o "${DIST_DIR}/m2apps-linux-amd64" "${ROOT_DIR}/main.go"
cp "${DIST_DIR}/m2apps-linux-amd64" "${DIST_DIR}/linux/m2apps"
chmod +x "${DIST_DIR}/linux/m2apps"

echo "Building macOS (amd64)..."
GOOS=darwin GOARCH=amd64 go build -o "${DIST_DIR}/m2apps-darwin-amd64" "${ROOT_DIR}/main.go"
cp "${DIST_DIR}/m2apps-darwin-amd64" "${DIST_DIR}/macos/m2apps"
chmod +x "${DIST_DIR}/macos/m2apps"

echo "Packaging release assets..."
(
	cd "${DIST_DIR}/windows"
	zip -r "../m2apps-windows-amd64.zip" "m2apps.exe"
)
(
	cd "${DIST_DIR}/linux"
	tar -czf "../m2apps-linux-amd64.tar.gz" "m2apps"
)
(
	cd "${DIST_DIR}/macos"
	tar -czf "../m2apps-darwin-amd64.tar.gz" "m2apps"
)

echo "Release assets generated in ${DIST_DIR}:"
ls -1 "${DIST_DIR}"
