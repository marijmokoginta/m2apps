# M2Apps — Phase 14 Codex Instructions (v1.1.0)

## Overview
Phase 14 focuses on:
1. Fixing daemon start issue (CRITICAL)
2. Service mode (auto-start apps)
3. Self-update system (advanced)

IMPORTANT:
Follow steps strictly in order. Do NOT skip.

---

# PART 1 — DAEMON FIX (BLOCKER)

## Goal
Ensure daemon can start reliably across OS.

## Tasks

### 1. Fix process spawn

Linux/macOS:
Use detached process with Setsid.

Example:
cmd := exec.Command(os.Args[0], "daemon", "run")
cmd.Stdout = nil
cmd.Stderr = nil
cmd.Stdin = nil
cmd.SysProcAttr = &syscall.SysProcAttr{
    Setsid: true,
}
cmd.Start()

---

Windows:
cmd := exec.Command(os.Args[0], "daemon", "run")
cmd.SysProcAttr = &syscall.SysProcAttr{
    CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
}
cmd.Start()

---

### 2. Add logging

Path:
~/.m2apps/logs/daemon.log

All daemon runtime errors MUST be logged.

---

### 3. Validate port binding

- Fail with clear error if port is already used
- Do NOT silently exit

---

### 4. Validation

- m2apps daemon start works
- m2apps daemon status shows running
- daemon survives terminal close

STOP if not working.

---

# PART 2 — SERVICE MODE (AUTO START APPS)

## Goal
Daemon automatically starts apps on boot.

---

### 1. Update metadata

Add:
AutoStart bool

Default: true

---

### 2. Daemon startup logic

On daemon run:

- load all apps
- for each app:
  if AutoStart == true:
    start app

---

### 3. Prevent duplicate start

- check existing PID
- if running → skip

---

### 4. Multi-process support

Each app may have multiple processes:

Example Laravel:
- web
- queue
- scheduler

Structure:

type AppProcess struct {
    AppID string
    Name  string
    PID   int
}

---

### 5. Integration

Reuse process manager (Phase 12)

---

### 6. Validation

- restart daemon → apps auto start
- no duplicate processes

---

# PART 3 — SELF UPDATE SYSTEM

## Goal
M2Apps can update itself safely.

---

### 1. Version check

API:
https://api.github.com/repos/marijmokoginta/m2apps/releases/latest

No auth needed.

---

### 2. Compare version

Reuse semantic version logic.

---

### 3. UI Prompt

Before main menu:

New version available: vX.X.X

Options:
- Update now
- Skip for now
- Skip until next version

---

### 4. Skip logic

File:
~/.m2apps/self_update.json

Example:
{
  "skipped_version": "v1.1.0"
}

---

### 5. Download binary

- detect OS
- detect ARCH
- download correct asset

---

### 6. SAFE REPLACEMENT (CRITICAL)

DO NOT overwrite running binary.

Steps:

1. download to temp:
   /tmp/m2apps_new

2. rename current:
   m2apps → m2apps_old

3. move new:
   m2apps_new → m2apps

---

### 7. Windows special handling

Cannot replace running binary.

Solution:

- create updater mode:
  m2apps internal self-update

Flow:
- main process exits
- updater replaces binary
- restart m2apps

---

### 8. Restart after update

After update:
- restart automatically

---

### 9. Validation

- update works on Linux/macOS
- update works on Windows (no file lock issue)
- binary replaced successfully
- app restarts

---

# RULES

- DO NOT skip order
- DO NOT overwrite binary directly
- DO NOT mix installer logic
- ALWAYS log errors

---

# DONE CRITERIA

- daemon start fixed
- service mode working
- self-update working across OS

---

# END
