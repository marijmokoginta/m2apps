# M2Apps — Phase 11 Codex Instructions (OS Service Integration)

## Overview
Phase 11 upgrades daemon from a manual background process into a native OS service.

This includes:
- Windows Service integration
- Linux systemd service
- macOS launchd service
- Permission handling (admin/sudo)
- Robust CLI control

---

## Goals

- Daemon runs automatically on system startup
- OS manages lifecycle (start/stop/restart)
- CLI interacts with OS service instead of raw process
- Safe handling of permission errors

---

## Existing Context

From previous phases:
- Daemon runtime already implemented (Phase 8)
- Service abstraction exists but is stubbed (Phase 9)
- Command abstraction and path helpers are available

---

## 1. ServiceManager Interface (Already Exists)

Ensure interface is used as contract:

```go
type ServiceManager interface {
    Install() error
    Uninstall() error
    Start() error
    Stop() error
    Status() (string, error)
}
```

---

## 2. OS Detection

Use runtime:

```go
switch runtime.GOOS {
case "windows":
    return &WindowsService{}
case "linux":
    return &LinuxService{}
case "darwin":
    return &MacOSService{}
}
```

---

## 3. WINDOWS IMPLEMENTATION

### Install

Command:
```
sc create M2Apps binPath= "C:\Program Files\M2Code\m2apps.exe daemon run"
```

### Start / Stop

```
sc start M2Apps
sc stop M2Apps
```

---

### Permission Handling (IMPORTANT)

Windows requires Administrator.

Detect permission:

```go
func isAdmin() bool {
    cmd := exec.Command("net", "session")
    return cmd.Run() == nil
}
```

If not admin:
- Return error:
```
[ERROR] Administrator privileges required. Please run terminal as Administrator.
```

---

## 4. LINUX IMPLEMENTATION (systemd)

### Service File

Path:
```
/etc/systemd/system/m2apps.service
```

Content:
```
[Unit]
Description=M2Apps Daemon
After=network.target

[Service]
ExecStart=/usr/local/bin/m2apps daemon run
Restart=always

[Install]
WantedBy=multi-user.target
```

---

### Commands

```
sudo systemctl daemon-reload
sudo systemctl enable m2apps
sudo systemctl start m2apps
```

---

### Permission Handling

Detect root:

```go
if os.Geteuid() != 0 {
    return fmt.Errorf("sudo required")
}
```

Error message:

```
[ERROR] Root privileges required. Please run with sudo.
```

---

## 5. MACOS IMPLEMENTATION (launchd)

### File

```
~/Library/LaunchAgents/com.m2apps.daemon.plist
```

### Content

```
<plist>
<dict>
    <key>Label</key>
    <string>com.m2apps.daemon</string>

    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/m2apps</string>
        <string>daemon</string>
        <string>run</string>
    </array>

    <key>RunAtLoad</key>
    <true/>
</dict>
</plist>
```

---

### Command

```
launchctl load ~/Library/LaunchAgents/com.m2apps.daemon.plist
```

---

## 6. CLI COMMANDS

Add new commands:

```
m2apps daemon install
m2apps daemon uninstall
m2apps daemon enable
m2apps daemon disable
m2apps daemon start
m2apps daemon stop
m2apps daemon status
```

---

## 7. ERROR HANDLING STRATEGY

### Cases

1. Permission denied
2. Service already exists
3. Service not found
4. Command failure

---

### Standard Output Format

```
[INFO] Installing service...
[OK] Service installed
[ERROR] Failed to install service: <reason>
```

---

## 8. SAFETY RULES

- Do not overwrite existing service without warning
- Validate binary path before install
- Ensure daemon run command works standalone
- Use command abstraction layer (Phase 9)

---

## 9. TESTING CHECKLIST

- Install service (with admin/sudo)
- Restart system → daemon auto runs
- Start/stop via CLI works
- Permission error shows correctly
- Status reflects actual state

---

## 10. IMPLEMENTATION ORDER

1. Linux (systemd)
2. Windows (sc)
3. macOS (launchd)

---

## FINAL RESULT

M2Apps daemon becomes:
- persistent
- OS-managed
- auto-starting
- production-ready

