# M2Apps Phase 8 — Codex Implementation Instructions
Date: 2026-04-14

## Objective

Phase 8 introduces full application integration via:

- Background daemon/service (cross-platform)
- Localhost API for app communication
- Realtime progress tracking system
- Environment injection for app integration

This phase transforms M2Apps from CLI tool into a persistent system service.

---

## IMPORTANT RULES

- DO NOT break Phase 1–7 behavior
- DO NOT modify installer/update core logic
- ADD new modules only
- KEEP separation of concerns
- ALL new logic must be modular under internal/

---

# 1. PROJECT STRUCTURE (NEW MODULES)

Create the following:

```
internal/daemon/
  service.go
  manager.go

internal/api/
  server.go
  middleware.go
  handlers.go

internal/progress/
  progress.go
  manager.go

internal/env/
  injector.go
```

---

# 2. DAEMON / BACKGROUND SERVICE

## Goal

Run M2Apps as persistent background service.

---

## Interface

```go
type ServiceManager interface {
    Install() error
    Start() error
    Stop() error
    Status() (string, error)
}
```

---

## Implementation

Create OS-specific logic (basic version first):

- Linux/macOS:
  - Run as background process (no systemd yet)
- Windows:
  - Run detached process

NOTE:
Do NOT implement full system service yet.
Just simulate daemon process.

---

## CLI COMMAND

Add:

```
m2apps daemon start
m2apps daemon stop
m2apps daemon status
```

---

# 3. LOCALHOST API SERVER

## Goal

Expose HTTP API for apps.

---

## Server

- Port: dynamic (store in memory)
- Bind: 127.0.0.1 only

---

## Endpoints

### 1. Check Update

GET /apps/{app_id}/update/check

Response:
```
{
  "has_update": true,
  "current_version": "1.0.0",
  "latest_version": "1.1.0"
}
```

---

### 2. Start Update

POST /apps/{app_id}/update

- Calls existing updater (Phase 7)

---

### 3. Status

GET /apps/{app_id}/update/status

Returns progress object.

---

### 4. Switch Channel

POST /apps/{app_id}/channel

---

### 5. Update Token

POST /apps/{app_id}/auth/update

---

# 4. AUTH MIDDLEWARE

## Requirement

All API requests must validate token.

---

## Flow

1. Read Authorization header
2. Extract Bearer token
3. Load app config (Phase 6)
4. Compare token

Reject if invalid.

---

# 5. PROGRESS ENGINE

## Goal

Track update progress globally.

---

## Struct

```go
type Progress struct {
    AppID   string
    Phase   string
    Step    string
    Percent int
    Logs    []string
    Status  string
}
```

---

## Manager

- Map[app_id]*Progress
- Thread-safe (use mutex)

---

## Functions

- Start(app_id)
- Update(app_id, phase, step, percent)
- Log(app_id, message)
- Complete(app_id)
- Fail(app_id)

---

## Integration

Hook into:

- downloader (Phase 4)
- installer (Phase 5)
- updater (Phase 7)

---

# 6. ENV INJECTION

## Goal

Inject API config into app environment.

---

## Variables

```
M2APPS_API_URL=http://127.0.0.1:{PORT}
M2APPS_API_TOKEN=<token>
M2APPS_APP_ID=<app_id>
```

---

## Behavior

- Detect file:
  - .env
  - .env.local
- Append if not exist
- DO NOT overwrite existing values

---

# 7. INTEGRATION FLOW

## INSTALL

After Phase 7 install success:

1. Start daemon (if not running)
2. Generate API token
3. Save token to encrypted storage
4. Inject env variables
5. Register app in daemon

---

## UPDATE (API triggered)

1. App calls API
2. API triggers updater
3. Progress updated
4. App polls status endpoint

---

# 8. ERROR HANDLING

- All API errors must return JSON
- No panic allowed
- Always return clear message

---

# 9. VALIDATION

Ensure:

- go build ./... passes
- daemon runs
- API responds
- token validation works
- progress updates correctly

---

# DONE CRITERIA

- CLI daemon commands work
- API endpoints functional
- Progress visible via API
- Env successfully injected
- Integration with updater works

---

# NOTES

- Do NOT implement WebSocket yet
- Use polling-based approach
- Keep implementation simple and stable
- Avoid over-engineering

---

This phase is critical.
Focus on correctness over complexity.
