# M2Apps — Phase 12 Codex Instructions (Process Manager)

## Overview
Implement basic process manager for running installed applications.

---

## Goals
- Start, stop, restart, status app
- Support multi-process per app
- Persist process state
- Cross-platform compatible

---

## 1. CLI Commands

Add new command group:

m2apps app start <app_id>
m2apps app stop <app_id>
m2apps app restart <app_id>
m2apps app status <app_id>

---

## 2. Project Structure

internal/process/
  manager.go
  registry.go
  types.go

cmd/app.go

---

## 3. Process Types

type Process struct {
    Name    string
    PID     int
    Command []string
    Status  string
}

type AppProcesses struct {
    AppID     string
    Processes []Process
}

---

## 4. Registry Storage

File:
~/.m2apps/processes.json

Functions:
- LoadAll()
- SaveAll()
- Get(appID)
- Set(appID, processes)

---

## 5. Runtime Preset

internal/runtime/

Example Laravel:
- php artisan serve
- php artisan queue:work
- php artisan schedule:work

Example Node:
- npm run start

---

## 6. Process Manager

Start:
- exec.Command(...).Start()
- capture PID
- redirect stdout/stderr to log file

Stop:
- kill PID

Status:
- check if PID alive

---

## 7. Command Flow

START:
- load preset
- check existing processes
- start each process
- save registry

STOP:
- load registry
- kill all processes
- update status

STATUS:
- check each PID
- print status

RESTART:
- stop → start

---

## 8. Logging

Location:
~/.m2apps/logs/{app_id}.log

---

## 9. Edge Cases

- duplicate start → prevent
- missing PID → skip
- dead process → mark stopped

---

## 10. Validation

Commands must:
- return clear output
- not crash on missing data
- handle cross-platform

---

## Done Criteria

- go build ./... passes
- all commands work
- process lifecycle stable
