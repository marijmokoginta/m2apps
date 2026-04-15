# M2Apps Phase 13 Summary

Date: 2026-04-15

## Scope Implemented

Phase 13 improves app process runtime behavior and interactive CLI UX.

Delivered scope:

- dynamic runtime port resolution
- persisted process port metadata
- runtime command placeholder injection for port
- better URL visibility in process start/status output
- improved clear-screen strategy for interactive menu transitions

## Process Runtime Enhancements

### Dynamic Port Resolution

Added package:

- `internal/network/port.go`
  - `IsPortAvailable(port int) bool`
  - `ResolvePort(base int) int`

Behavior:

- start flow attempts to reuse saved port when available
- if not available, next free port is selected from runtime default base

### Runtime Preset Port Placeholder

Updated:

- `internal/runtime/preset.go`

Changes:

- Laravel web command now uses `--port={PORT}`
- added runtime default port mapping:
  - Laravel / Laravel Inertia: `8000`
  - Node / Next.js: `3000`
  - Flutter: `5000`

### Process Registry Port Persistence

Updated:

- `internal/process/types.go`
  - added `Process.Port int`
- `internal/process/registry.go`
  - normalize and persist `Port` field

## Process Manager Flow Updates

Updated:

- `internal/process/manager.go`

Changes:

- resolves runtime port before starting commands
- injects resolved port into `{PORT}` placeholders
- stores port in process registry state
- injects APP_URL using resolved port

## Env Injection Update

Updated:

- `internal/env/injector.go`

Changes:

- added `InjectAppURL(installPath string, port int) error`
- supports placeholder replacement for dynamic APP_URL value

## CLI Output and UX Improvements

Updated:

- `cmd/app.go`
- `cmd/root.go`
- `internal/ui/menu.go`

Changes:

- app start output now includes PID and URL details per process
- app status table includes URL column with port-aware resolution
- screen is cleared when navigating between menus, but final info screens remain visible
- user confirms return with `Press Enter to back to Main Menu...`
- interactive flow errors are displayed and paused before menu redraw

## Validation Snapshot

Completed checks:

- `go test ./...`
- `go build ./...`
