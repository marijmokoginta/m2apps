#!/usr/bin/env bash
set -euo pipefail

REPO_OWNER="${M2APPS_REPO_OWNER:-marijmokoginta}"
REPO_NAME="${M2APPS_REPO_NAME:-m2apps}"
TARGET="/usr/local/bin/m2apps"
API_URL="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest"

cleanup() {
  if [[ -n "${TMP_DIR:-}" && -d "${TMP_DIR}" ]]; then
    rm -rf "${TMP_DIR}"
  fi
}
trap cleanup EXIT

log() {
  printf '[INFO] %s\n' "$1"
}

fail() {
  printf '[ERROR] %s\n' "$1" >&2
  exit 1
}

resolve_os() {
  local uname_os
  uname_os="$(uname -s)"
  case "${uname_os}" in
    Linux) echo "linux" ;;
    Darwin) echo "darwin" ;;
    *) fail "unsupported OS: ${uname_os}" ;;
  esac
}

resolve_arch() {
  local uname_arch
  uname_arch="$(uname -m)"
  case "${uname_arch}" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *) fail "unsupported architecture: ${uname_arch}" ;;
  esac
}

require_python3() {
  if ! command -v python3 >/dev/null 2>&1; then
    fail "python3 is required to parse release metadata"
  fi
}

download_release_json() {
  if ! curl -fsSL "${API_URL}"; then
    fail "failed to fetch latest release metadata from ${API_URL}"
  fi
}

extract_download_url() {
  local asset_name="$1"
  python3 -c '
import json
import sys

asset_name = sys.argv[1]
payload = json.load(sys.stdin)
for asset in payload.get("assets", []):
    if asset.get("name") == asset_name:
        print(asset.get("browser_download_url", ""))
        sys.exit(0)
sys.exit(1)
' "${asset_name}"
}

install_binary() {
  local source_path="$1"
  if [[ -w "$(dirname "${TARGET}")" ]]; then
    mv -f "${source_path}" "${TARGET}"
  elif command -v sudo >/dev/null 2>&1; then
    sudo mv -f "${source_path}" "${TARGET}"
  else
    fail "permission denied. run as root or install sudo"
  fi

  if [[ -w "${TARGET}" ]]; then
    chmod +x "${TARGET}"
  elif command -v sudo >/dev/null 2>&1; then
    sudo chmod +x "${TARGET}"
  else
    fail "permission denied while setting executable bit"
  fi
}

validate_installation() {
  if command -v m2apps >/dev/null 2>&1; then
    m2apps --version >/dev/null 2>&1 || fail "installation validation failed: m2apps --version"
    return
  fi

  "${TARGET}" --version >/dev/null 2>&1 || fail "installation validation failed: ${TARGET} --version"
}

main() {
  local os arch asset_name release_json download_url archive_path extracted_binary

  os="$(resolve_os)"
  arch="$(resolve_arch)"
  asset_name="m2apps-${os}-${arch}.tar.gz"

  log "Repository: ${REPO_OWNER}/${REPO_NAME}"
  log "Detected target asset: ${asset_name}"

  require_python3
  release_json="$(download_release_json)"
  download_url="$(printf '%s' "${release_json}" | extract_download_url "${asset_name}")" || true

  if [[ -z "${download_url}" ]]; then
    fail "release asset not found for ${asset_name}"
  fi

  TMP_DIR="$(mktemp -d)"
  archive_path="${TMP_DIR}/${asset_name}"
  extracted_binary="${TMP_DIR}/m2apps"

  log "Downloading release archive from GitHub Release..."
  if ! curl -fL "${download_url}" -o "${archive_path}"; then
    fail "failed to download binary from ${download_url}"
  fi

  if ! command -v tar >/dev/null 2>&1; then
    fail "tar is required to extract release archive"
  fi

  log "Extracting release archive..."
  if ! tar -xzf "${archive_path}" -C "${TMP_DIR}"; then
    fail "failed to extract archive ${asset_name}"
  fi

  if [[ ! -f "${extracted_binary}" ]]; then
    fail "m2apps binary not found after extraction"
  fi

  chmod +x "${extracted_binary}"

  log "Installing to ${TARGET}..."
  install_binary "${extracted_binary}"
  validate_installation

  log "M2Apps installed successfully."
}

main "$@"
