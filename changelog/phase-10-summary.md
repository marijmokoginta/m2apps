# M2Apps Phase 10 Summary

Date: 2026-04-14

## Scope Implemented

Phase 10 introduces distribution pipeline readiness and install validation enhancement by adding:

- unique `app_id` install validation to prevent duplicate installations
- cross-platform build and packaging script
- GitHub Actions release workflow for tag-based releases
- daemon process attribute adaptation for successful cross-compilation

## Install Validation Enhancement

### Storage API Update

Updated storage interface:

- `internal/storage/model.go`
  - added `Exists(appID string) (bool, error)`

Implemented method:

- `internal/storage/storage.go`
  - checks app directory existence under `~/.m2apps/apps/{app_id}`
  - returns `true` when app directory already exists

### Install Command Update

Updated:

- `cmd/install.go`

Behavior added before install flow starts:

1. initialize storage
2. check `store.Exists(cfg.AppID)`
3. if app already exists, abort install with:
   - `[ERROR] Application with app_id '<app_id>' is already installed.`
   - `[ERROR] Installation aborted.`

## Build & Packaging Pipeline

Added:

- `scripts/build-release.sh`

Script outputs:

- `dist/m2apps-windows-amd64.exe`
- `dist/m2apps-linux-amd64`
- `dist/m2apps-darwin-amd64`

Packaging outputs:

- `dist/m2apps-windows-amd64.zip`
- `dist/m2apps-linux-amd64.tar.gz`
- `dist/m2apps-darwin-amd64.tar.gz`

Also creates structured directories:

- `dist/windows/m2apps.exe`
- `dist/linux/m2apps`
- `dist/macos/m2apps`

## GitHub Release Workflow

Added:

- `.github/workflows/release.yml`

Implemented:

- trigger on pushed tags matching `v*`
- matrix build for:
  - `windows-amd64`
  - `linux-amd64`
  - `darwin-amd64`
- packaging per target
- artifact upload per build job
- release creation via:
  - `actions/create-release`
  - `actions/upload-release-asset`

Release assets configured:

- `m2apps-windows-amd64.zip`
- `m2apps-linux-amd64.tar.gz`
- `m2apps-darwin-amd64.tar.gz`

## Cross-Compile Compatibility Fix

Updated daemon process setup to compile cleanly for Windows target:

- `internal/daemon/manager.go`
- `internal/daemon/procattr_unix.go`
- `internal/daemon/procattr_windows.go`

This prevents Windows build errors related to Unix-only `SysProcAttr` fields.

## Supporting Changes

- `.gitignore`
  - added `/dist/` to avoid committing build outputs
- `README.md`
  - added release workflow instructions (how to push tags and trigger CI release)

## Validation Snapshot

Completed checks during Phase 10 implementation:

- `go build ./...` passes
- duplicate `app_id` check aborts installation correctly
- local script `./scripts/build-release.sh` builds and packages all target assets
