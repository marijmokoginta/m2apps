#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist"
RELEASE_DIR="${ROOT_DIR}/release"

rm -rf "${DIST_DIR}" "${RELEASE_DIR}"
mkdir -p "${DIST_DIR}/windows" "${DIST_DIR}/linux" "${DIST_DIR}/macos"
mkdir -p "${RELEASE_DIR}/windows" "${RELEASE_DIR}/linux" "${RELEASE_DIR}/macos"

if [[ ! -f "${ROOT_DIR}/install.sh" ]]; then
	echo "Missing installer script: install.sh"
	exit 1
fi

if [[ ! -f "${ROOT_DIR}/install.ps1" ]]; then
	echo "Missing installer script: install.ps1"
	exit 1
fi

echo "Building Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -o "${DIST_DIR}/m2apps-windows-amd64.exe" "${ROOT_DIR}/main.go"
cp "${DIST_DIR}/m2apps-windows-amd64.exe" "${DIST_DIR}/windows/m2apps.exe"
cp "${DIST_DIR}/windows/m2apps.exe" "${RELEASE_DIR}/windows/m2apps.exe"
cp "${ROOT_DIR}/install.ps1" "${DIST_DIR}/windows/install.ps1"
cp "${ROOT_DIR}/install.ps1" "${RELEASE_DIR}/windows/install.ps1"

echo "Building Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -o "${DIST_DIR}/m2apps-linux-amd64" "${ROOT_DIR}/main.go"
cp "${DIST_DIR}/m2apps-linux-amd64" "${DIST_DIR}/linux/m2apps"
chmod +x "${DIST_DIR}/linux/m2apps"
cp "${DIST_DIR}/linux/m2apps" "${RELEASE_DIR}/linux/m2apps"
cp "${ROOT_DIR}/install.sh" "${DIST_DIR}/linux/install.sh"
cp "${ROOT_DIR}/install.sh" "${RELEASE_DIR}/linux/install.sh"
chmod +x "${DIST_DIR}/linux/install.sh" "${RELEASE_DIR}/linux/install.sh"

echo "Building macOS (amd64)..."
GOOS=darwin GOARCH=amd64 go build -o "${DIST_DIR}/m2apps-darwin-amd64" "${ROOT_DIR}/main.go"
cp "${DIST_DIR}/m2apps-darwin-amd64" "${DIST_DIR}/macos/m2apps"
chmod +x "${DIST_DIR}/macos/m2apps"
cp "${DIST_DIR}/macos/m2apps" "${RELEASE_DIR}/macos/m2apps"
cp "${ROOT_DIR}/install.sh" "${DIST_DIR}/macos/install.sh"
cp "${ROOT_DIR}/install.sh" "${RELEASE_DIR}/macos/install.sh"
chmod +x "${DIST_DIR}/macos/install.sh" "${RELEASE_DIR}/macos/install.sh"

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

echo "Distribution directory generated in ${RELEASE_DIR}:"
find "${RELEASE_DIR}" -maxdepth 2 -type f | sort
