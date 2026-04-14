# M2Apps Phase 8 Summary

Date: 2026-04-14

## Scope Implemented

Phase 8 introduces daemon-based app integration and local API communication, including:

- background daemon manager and CLI commands
- localhost HTTP API for app-triggered update operations
- bearer-token authentication middleware
- global progress tracking engine
- environment variable injection into app env files
- install flow integration to register app and connect to daemon API

## New Modules Added

### Daemon

- `internal/daemon/service.go`
- `internal/daemon/manager.go`

Implemented:

- `ServiceManager` interface (`Install`, `Start`, `Stop`, `Status`)
- dynamic localhost listener management (`127.0.0.1:0`)
- daemon runtime files (`pid`, `port`) under `~/.m2apps/daemon`
- app registry file (`apps.json`)
- API token generator for app auth integration

### API

- `internal/api/server.go`
- `internal/api/middleware.go`
- `internal/api/handlers.go`

Implemented endpoints:

- `GET /apps/{app_id}/update/check`
- `POST /apps/{app_id}/update`
- `GET /apps/{app_id}/update/status`
- `POST /apps/{app_id}/channel`
- `POST /apps/{app_id}/auth/update`

All API errors return JSON and auth is enforced via bearer token.

### Progress

- `internal/progress/progress.go`
- `internal/progress/manager.go`

Implemented:

- thread-safe in-memory progress map keyed by `app_id`
- lifecycle methods: `Start`, `Update`, `Log`, `Complete`, `Fail`, `Get`
- default singleton manager for cross-module integration

### Env Injection

- `internal/env/injector.go`

Implemented behavior:

- detect `.env`, then `.env.local`
- create `.env` if no env file exists
- append only missing keys (no overwrite)
- inject:
  - `M2APPS_API_URL`
  - `M2APPS_API_TOKEN`
  - `M2APPS_APP_ID`

## CLI and Integration Changes

### New CLI Command Group

- `cmd/daemon.go`

Commands added:

- `m2apps daemon start`
- `m2apps daemon stop`
- `m2apps daemon status`
- hidden internal runner: `m2apps daemon run`

### Install Flow Integration

- `cmd/install.go`

After successful install, flow now performs:

1. ensure daemon is running
2. resolve daemon port
3. generate API token
4. save token in encrypted app metadata (`api_token`)
5. inject API env variables into installed app directory
6. register app in daemon registry

### Storage / Updater / Installer Integration

- `internal/storage/model.go`
  - added `APIToken` field in `AppConfig`
- `internal/updater/check.go`
  - added update-check result model and check function
- `internal/updater/updater.go`
  - integrated global progress updates during update lifecycle
- `internal/installer/installer.go`
  - added optional progress reporting fields in install context

## Validation Snapshot

Completed checks during Phase 8 implementation:

- `go build ./...` passes
- daemon command flow works:
  - start -> running with dynamic port
  - status -> reports current running state
  - stop -> stops cleanly
- API auth middleware responds correctly:
  - request without bearer token returns `401` JSON error
