# M2Apps — Phase 10 Codex Instructions (Build & Release Pipeline + Install Validation Enhancement)

## Overview

Phase 10 focuses on:

1. Cross-platform build automation
2. GitHub Release pipeline using GitHub Actions
3. Asset packaging per OS
4. Enhancement: Unique App ID validation during install

This phase transitions M2Apps into a distributable product.

---

## 1. Unique App ID Validation (Install Enhancement)

### Objective
Prevent duplicate installations of the same `app_id`.

---

### Implementation

#### File: internal/storage/storage.go

Add method:

```
func (s *Storage) Exists(appID string) (bool, error)
```

Logic:
- Check if directory exists:
  ~/.m2apps/apps/{app_id}
- If exists → return true

---

#### Update: cmd/install.go

Before installation starts:

1. Load config
2. Call storage.Exists(appID)

If exists:

```
[ERROR] Application with app_id '<app_id>' is already installed.
Installation aborted.
```

Exit with non-zero status.

---

### Behavior

| Condition | Result |
|----------|--------|
| app_id not found | proceed |
| app_id exists | abort |

---

## 2. Build Strategy (Cross Platform)

### Targets

- Windows (amd64)
- Linux (amd64)
- macOS (amd64)

---

### Output Naming

```
m2apps-windows-amd64.exe
m2apps-linux-amd64
m2apps-darwin-amd64
```

---

### Local Build Commands

```
GOOS=windows GOARCH=amd64 go build -o dist/m2apps-windows-amd64.exe
GOOS=linux GOARCH=amd64 go build -o dist/m2apps-linux-amd64
GOOS=darwin GOARCH=amd64 go build -o dist/m2apps-darwin-amd64
```

---

## 3. Packaging

### Directory Structure

```
dist/
  windows/
    m2apps.exe
  linux/
    m2apps
  macos/
    m2apps
```

---

### Optional (Recommended)

Compress:

- Windows → zip
- Linux/macOS → tar.gz

---

## 4. GitHub Actions Workflow

### File

```
.github/workflows/release.yml
```

---

### Trigger

```
on:
  push:
    tags:
      - 'v*'
```

---

### Jobs

#### 1. Build Matrix

```
strategy:
  matrix:
    os: [ubuntu-latest]
    target:
      - windows-amd64
      - linux-amd64
      - darwin-amd64
```

---

#### 2. Steps

- Checkout repo
- Setup Go
- Build binaries per target
- Archive outputs
- Upload artifacts

---

#### 3. Release Step

Use GitHub Action:

- actions/create-release
- actions/upload-release-asset

---

### Release Assets

- m2apps-windows-amd64.zip
- m2apps-linux-amd64.tar.gz
- m2apps-darwin-amd64.tar.gz

---

## 5. Versioning Strategy

- Follow semantic versioning: v1.0.0
- Tag required to trigger release

---

## 6. Validation Checklist

Before tagging:

- go build ./... passes
- install command works
- update command works
- no hardcoded paths
- no debug logs

---

## 7. Done Criteria

Phase 10 is complete when:

- App ID uniqueness validation works
- Multi-OS binaries generated
- GitHub Actions builds successfully
- Release created with assets
- Assets downloadable and runnable

---

## Notes

- Do NOT mix build logic into runtime code
- Keep CI/CD separate from core logic
- Ensure reproducible builds
