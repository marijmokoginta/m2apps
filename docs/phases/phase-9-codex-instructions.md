# M2Apps — Phase 9 Codex Instructions (Platform Adaptation Layer)

## Objective
Implement full cross-platform readiness for M2Apps by introducing:
1. Service Manager per OS
2. Command Abstraction Layer
3. Path Abstraction Layer
4. Complete README.md with API documentation

IMPORTANT:
- Do NOT modify core logic from Phase 1–8
- Extend via new modules only
- Maintain clean architecture separation

---

## 1. SERVICE MANAGER (CROSS-OS)

### Goal
Provide OS-specific background service integration.

### Structure

internal/service/
  service.go
  manager.go
  windows.go
  linux.go
  macos.go

---

### service.go

Define interface:

```go
type ServiceManager interface {
    Install() error
    Start() error
    Stop() error
    Status() (string, error)
}
```

---

### manager.go

Detect OS and return correct implementation:

```go
func NewServiceManager() ServiceManager {
    switch runtime.GOOS {
    case "windows":
        return NewWindowsService()
    case "linux":
        return NewLinuxService()
    case "darwin":
        return NewMacOSService()
    default:
        panic("unsupported OS")
    }
}
```

---

### Implementation Rules

#### Windows
- Stub implementation (no full service yet)
- Print: "Windows service not implemented yet"

#### Linux
- Stub systemd integration
- Prepare for:
  - service file generation
  - systemctl commands

#### macOS
- Stub launchd integration

---

### CLI Integration

Add:

```bash
m2apps daemon install
m2apps daemon start
m2apps daemon stop
m2apps daemon status
```

---

## 2. COMMAND ABSTRACTION

### Goal
Normalize command execution across OS.

---

### Structure

internal/system/command.go

---

### Implementation

```go
func RunCommand(cmd string, args ...string) error {
    var command *exec.Cmd

    if runtime.GOOS == "windows" {
        command = exec.Command("cmd", append([]string{"/C", cmd}, args...)...)
    } else {
        command = exec.Command(cmd, args...)
    }

    command.Stdout = os.Stdout
    command.Stderr = os.Stderr

    return command.Run()
}
```

---

### Replace Usage

- Replace all direct exec.Command usage
- Use RunCommand everywhere

---

## 3. PATH ABSTRACTION

### Goal
Standardize storage paths across OS.

---

### Structure

internal/system/path.go

---

### Implementation

```go
func GetBaseDir() string {
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".m2apps")
}

func GetAppDir(appID string) string {
    return filepath.Join(GetBaseDir(), "apps", appID)
}

func GetLogDir() string {
    return filepath.Join(GetBaseDir(), "logs")
}
```

---

### Rules

- No hardcoded paths
- Always use abstraction
- Ensure directory creation

---

## 4. README.md (OFFICIAL DOCUMENTATION)

Create README.md in root.

---

### Sections Required

#### 1. Introduction
Explain M2Apps as:
- installer
- updater
- deployment agent

---

#### 2. Features
- Cross-platform CLI
- GitHub Release integration
- Encrypted storage
- Background daemon
- Local API

---

#### 3. Installation

Explain:
- Download binary
- Add to PATH
- Run `m2apps`

---

#### 4. Usage

```bash
m2apps install
m2apps update <app_id>
m2apps list
m2apps daemon start
```

---

#### 5. install.json Example

Provide full example config.

---

#### 6. Local API Documentation

Base URL:
http://127.0.0.1:{PORT}

Endpoints:

GET /apps/{app_id}/update/check
POST /apps/{app_id}/update
GET /apps/{app_id}/update/status
POST /apps/{app_id}/channel

Include request/response examples.

---

#### 7. Architecture Overview

Short explanation of:
- CLI
- Daemon
- Storage
- GitHub integration

---

#### 8. Security

- Token handling
- Encryption
- Local API auth

---

#### 9. Troubleshooting

- Requirement failures
- Token issues
- Permission issues

---

## DONE CRITERIA

- Service manager interface implemented
- OS-specific stubs exist
- Command abstraction used globally
- Path abstraction replaces hardcoded paths
- README.md created and complete
- Project builds successfully

---

## NOTES

- Do NOT implement full OS service yet (only stub)
- Focus on structure and extensibility
- Keep code clean and modular
