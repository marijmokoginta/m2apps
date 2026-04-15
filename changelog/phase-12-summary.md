# M2Apps Phase 12 Summary

Date: 2026-04-15

## Scope Implemented

Phase 12 introduces a basic process manager to run installed applications with cross-platform CLI controls and persisted process state.

Delivered scope:

- process lifecycle commands: start, stop, restart, status
- multi-process support per app
- persistent process registry
- runtime presets for Laravel and Node
- process log output per app
- interactive process management from main menu

## New CLI Commands

Added command group:

- `m2apps app start <app_id>`
- `m2apps app stop <app_id>`
- `m2apps app restart <app_id>`
- `m2apps app status <app_id>`

File:

- `cmd/app.go`

## Process Layer

Added package:

- `internal/process/types.go`
  - `Process`
  - `AppProcesses`
- `internal/process/registry.go`
  - `LoadAll()`
  - `SaveAll()`
  - `Get(appID)`
  - `Set(appID, processes)`
- `internal/process/manager.go`
  - `Start(appID)`
  - `Stop(appID)`
  - `Restart(appID)`
  - `Status(appID)`

Process registry location:

- `~/.m2apps/processes.json`

Process log location:

- `~/.m2apps/logs/{app_id}.log`

## Runtime Preset Layer

Added:

- `internal/runtime/preset.go`

Supported runtime presets:

- Laravel (`laravel`, `laravel-inertia`)
  - `php artisan serve --host=127.0.0.1 --port=8000`
  - `php artisan queue:work`
  - `php artisan schedule:work`
- Node (`node`, `nodejs`)
  - `npm run start`

## Process Lifecycle Behavior

### Start

- load app metadata
- load runtime preset commands
- prevent duplicate start when running process exists
- start each process using `exec.Command(...).Start()`
- capture PID and process command
- append stdout/stderr to app log file
- persist started processes to registry

### Stop

- load registry for app
- stop each PID (cross-platform)
- skip missing/invalid PID safely
- update each process status and persist

### Status

- load registry for app
- check each PID alive/dead
- mark dead process as `stopped`
- persist updated status when changed

### Restart

- stop then start

## Cross-Platform Process Handling

Implemented in manager:

- Unix-like:
  - liveness check via `kill -0`
  - stop via `kill -TERM`, fallback `kill -KILL`
- Windows:
  - liveness check via `tasklist`
  - stop via `taskkill /PID /T /F`

## UX Enhancements Implemented During Phase 12

### Main Menu Expansion

Root interactive menu now covers all major operations:

- install
- update
- manage application process
- delete
- switch channel
- list installed applications
- manage daemon service
- help
- exit

### Back Navigation

Added explicit back navigation behavior:

- `Back` option on submenus
- final/info-only screens show `Back to Main Menu` prompt

This avoids abrupt jumps back to root menu.

### Process Status Display

`app status` now renders a structured table with columns:

- `NAME`
- `PID`
- `STATUS` (colorized)
- `URL`
- `COMMAND`

URL inference is shown for process names `web` / `server` when host/port can be resolved.

### Start Output Detail

`app start` now prints started process details:

- process name
- PID
- URL (when resolvable for web/server process)

### Menu Readability

Interactive menu rendering now uses vertical spacing between menu items.

## Edge Cases Covered

- duplicate start prevented
- missing PID handled safely on stop
- dead process automatically marked stopped
- process command action supports app_id resolution (exact, name, prefix unique match)

## Validation Snapshot

Completed checks:

- `go test ./...`
- `go build ./...`

All checks passed.
