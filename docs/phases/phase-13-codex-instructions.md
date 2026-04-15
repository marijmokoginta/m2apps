# M2Apps — Phase 13 Codex Instructions (Dynamic Port & URL Injection)

## Overview
Phase 13 enhances runtime presets with dynamic port allocation and environment URL injection.

This prevents port conflicts and ensures frontend assets (e.g. Laravel + Vite) work correctly.

---

## Objectives

1. Replace static ports with dynamic allocation
2. Introduce `{PORT}` placeholder in runtime presets
3. Persist assigned port in process state
4. Inject full application URL into `.env`
5. Display URL in CLI output

---

## 1. Dynamic Port Allocation

### Default Ports

| Preset | Default Port |
|--------|-------------|
| laravel | 8000 |
| node | 3000 |
| nextjs | 3000 |
| flutter | 5000 |

---

### Implement Port Checker

File:
`internal/network/port.go`

```go
package network

import (
    "fmt"
    "net"
)

func IsPortAvailable(port int) bool {
    ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
    if err != nil {
        return false
    }
    ln.Close()
    return true
}
```

---

### Implement Port Resolver

```go
func ResolvePort(base int) int {
    port := base
    for {
        if IsPortAvailable(port) {
            return port
        }
        port++
    }
}
```

---

## 2. Update Runtime Presets

File:
`internal/runtime/preset.go`

Replace static ports:

### BEFORE
```go
"php artisan serve --host=127.0.0.1 --port=8000"
```

### AFTER
```go
"php artisan serve --host=127.0.0.1 --port={PORT}"
```

---

## 3. Inject Port at Runtime

File:
`internal/process/manager.go`

Before executing command:

```go
cmdStr = strings.ReplaceAll(cmdStr, "{PORT}", strconv.Itoa(port))
```

---

## 4. Persist Port in Process State

File:
`internal/process/types.go`

Add field:

```go
Port int `json:"port"`
```

---

Ensure saved in registry:

File:
`internal/process/registry.go`

---

## 5. URL Construction

```go
url := fmt.Sprintf("http://127.0.0.1:%d", port)
```

---

## 6. Inject URL into .env

File:
`internal/env/injector.go`

Add:

```env
APP_URL=http://127.0.0.1:{PORT}
```

---

### Behavior

- Replace `{PORT}` with resolved port
- Append only if APP_URL not exists

---

## 7. Update Start Flow

File:
`internal/process/manager.go`

Flow:

1. resolve port
2. inject into commands
3. start process
4. save process with port
5. inject env
6. print URL

---

## 8. CLI Output

```
[OK] App started
URL: http://127.0.0.1:8001
```

---

## 9. Status Command Update

Display URL from saved port.

---

## 10. Edge Cases

- Port conflict → auto increment
- App restart → reuse stored port if available
- Dead process → re-resolve port

---

## Done Criteria

- App runs without port conflict
- URL displayed correctly
- APP_URL injected into env
- State includes port
- All commands build successfully

---

## Notes

Do NOT over-engineer:
- no random port allocation
- no external port scanning lib
- sequential increment only

Keep it simple and deterministic.
