# M2Apps Phase 14 Summary

Date: 2026-04-15

## Scope Implemented

Phase 14 implementation delivered in three parts:

- daemon reliability fix
- service mode auto-start apps
- self-update system for the M2Apps CLI binary

## Part 1: Daemon Fix

Implemented:

- detached daemon spawn attributes per OS:
  - Unix: `Setsid`
  - Windows: `CREATE_NEW_PROCESS_GROUP`
- daemon runtime logging pipeline to:
  - `~/.m2apps/logs/daemon.log`
- startup port bind validation with explicit failure messages when the port is already in use

Updated files:

- `internal/daemon/manager.go`
- `internal/daemon/procattr_windows.go`
- `internal/daemon/log.go`
- `cmd/daemon.go`

## Part 2: Service Mode (Auto Start Apps)

Implemented:

- new metadata field: `AutoStart bool`
- default compatibility behavior for existing encrypted configs:
  - if `auto_start` is missing, treated as `true`
- daemon startup loads installed apps and auto-starts only apps with `AutoStart=true`
- duplicate prevention by checking running process state before start
- multi-process app startup reused from existing process manager layer

Updated files:

- `internal/storage/model.go`
- `internal/storage/storage.go`
- `cmd/install.go`
- `internal/daemon/manager.go`

## Part 3: Self-Update System

Implemented:

- latest release check endpoint:
  - `https://api.github.com/repos/marijmokoginta/m2apps/releases/latest`
- semantic version comparison reuse for update decision
- startup update prompt with actions:
  - Update now
  - Skip for now
  - Skip until next version
- skip state persistence in:
  - `~/.m2apps/self_update.json`
- platform-aware release asset resolution and archive extraction:
  - Windows: `m2apps-windows-amd64.zip`
  - Linux: `m2apps-linux-amd64.tar.gz`
  - macOS: `m2apps-darwin-amd64.tar.gz`
- safe binary replacement flow:
  - rename current binary to `_old`
  - move new binary into current path
  - restart app automatically
- Windows locked-binary handling via internal updater command:
  - `m2apps internal self-update`

Updated files:

- `internal/github/client.go`
- `internal/selfupdate/selfupdate.go`
- `internal/selfupdate/procattr_unix.go`
- `internal/selfupdate/procattr_windows.go`
- `cmd/internal.go`
- `cmd/root.go`

## Additional UX Enhancement Included

- Added supervisor permission popup trigger for privileged daemon actions and installer flows.

Updated files:

- `internal/privilege/elevate.go`
- `internal/privilege/elevate_unix.go`
- `internal/privilege/elevate_windows.go`
- `cmd/daemon.go`
- `install.sh`
- `install.ps1`

## Validation Snapshot

Completed checks:

- `go test ./...`
- `go build ./...`
