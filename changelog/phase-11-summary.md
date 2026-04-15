# M2Apps Phase 11 Summary

Date: 2026-04-14

## Scope Implemented

Phase 11 upgrades daemon control from manual process management into native OS service integration with:

- Linux `systemd` implementation
- Windows Service implementation
- macOS `launchd` implementation
- permission-aware execution flow
- extended daemon CLI control commands

## Service Contract Upgrade

Updated service interface:

- `internal/service/service.go`
  - `Install() error`
  - `Uninstall() error`
  - `Enable() error`
  - `Disable() error`
  - `Start() error`
  - `Stop() error`
  - `Status() (string, error)`

## CLI Integration Changes

Updated daemon command module:

- `cmd/daemon.go`

New service commands:

- `m2apps daemon install`
- `m2apps daemon uninstall`
- `m2apps daemon enable`
- `m2apps daemon disable`
- `m2apps daemon start`
- `m2apps daemon stop`
- `m2apps daemon status`

Internal runtime command is preserved:

- hidden `m2apps daemon run`

Output strategy now follows standardized operation format:

- `[INFO] <operation>`
- `[OK] <result>`
- `[ERROR] <failure reason>`

## Linux Service Implementation (`systemd`)

Implemented in:

- `internal/service/linux.go`

Behavior implemented:

- validates root privilege before mutating service state
- validates daemon binary path at `/usr/local/bin/m2apps`
- writes service file:
  - `/etc/systemd/system/m2apps.service`
- unit content includes:
  - `ExecStart=/usr/local/bin/m2apps daemon run`
  - `Restart=always`
  - `WantedBy=multi-user.target`
- executes lifecycle commands via command abstraction:
  - `systemctl daemon-reload`
  - `systemctl enable/disable m2apps`
  - `systemctl start/stop m2apps`
  - `systemctl is-active m2apps`

Safety behavior:

- install fails if service file already exists
- uninstall removes service file and reloads systemd
- returns explicit permission error when non-root:
  - `[ERROR] Root privileges required. Please run with sudo.`

## Windows Service Implementation (`sc`)

Implemented in:

- `internal/service/windows.go`

Behavior implemented:

- detects Administrator privilege using:
  - `net session`
- validates daemon binary path:
  - `C:\Program Files\M2Code\m2apps.exe`
- installs service using `sc create` with daemon run argument
- supports:
  - uninstall (`sc delete`)
  - enable/disable auto-start (`sc config start= auto|demand`)
  - start/stop (`sc start`, `sc stop`)
  - status query (`sc query`)

Error handling:

- maps not-found state from `sc` output (`FAILED 1060`)
- handles already-running / already-stopped states gracefully
- returns explicit permission error when non-admin:
  - `[ERROR] Administrator privileges required. Please run terminal as Administrator.`

## macOS Service Implementation (`launchd`)

Implemented in:

- `internal/service/macos.go`

Behavior implemented:

- validates daemon binary path at `/usr/local/bin/m2apps`
- generates launch agent plist:
  - `~/Library/LaunchAgents/com.m2apps.daemon.plist`
- plist contains program args:
  - `/usr/local/bin/m2apps daemon run`
- supports:
  - install/load (`launchctl load`)
  - uninstall/unload + plist removal
  - enable/disable (`launchctl load/unload`)
  - start/stop (`launchctl start/stop`)
  - status (`launchctl list com.m2apps.daemon`)

Safety behavior:

- install fails if plist already exists
- uninstall fails when service plist is not found

## Privilege Helper Split (Cross-Platform Build Safety)

Added:

- `internal/service/privilege_unix.go`
  - uses `os.Geteuid()` for root detection
- `internal/service/privilege_windows.go`
  - Windows-safe stub for root check symbol compatibility

This keeps cross-compilation stable while preserving Linux root validation logic.

## Validation Snapshot

Completed checks:

- `go test ./...`
- `GOOS=linux GOARCH=amd64 go build ./main.go`
- `GOOS=windows GOARCH=amd64 go build ./main.go`
- `GOOS=darwin GOARCH=amd64 go build ./main.go`

All checks passed.
