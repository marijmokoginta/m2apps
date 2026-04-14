# M2Apps — Extended Installer Instructions (Phase 10 Enhancement)

## Overview
This document extends the installer system to support full distribution:
Installer will download binary from GitHub Release dynamically.

---

## Goals
- Installer downloads correct binary per OS/ARCH
- No manual binary handling by user
- Fully automated install experience

---

## GitHub Release Assumptions
Assets:
- m2apps-windows-amd64.exe
- m2apps-linux-amd64
- m2apps-darwin-amd64

---

## 1. OS & ARCH Detection

### Linux / macOS (bash)
```
OS=$(uname -s)
ARCH=$(uname -m)
```

Mapping:
- x86_64 → amd64
- arm64 → arm64

---

## 2. Resolve Download URL

Pattern:
```
https://api.github.com/repos/<owner>/<repo>/releases/latest
```

Extract asset by name:
```
m2apps-${os}-${arch}
```

---

## 3. Download Binary

### Linux/macOS
```
curl -L -o m2apps <download_url>
chmod +x m2apps
```

---

### Windows (PowerShell)
```
Invoke-WebRequest -Uri <download_url> -OutFile m2apps.exe
```

---

## 4. Install Location

### Linux/macOS
```
/usr/local/bin/m2apps
```

### Windows
```
C:\Program Files\M2Code\m2apps.exe
```

---

## 5. Move Binary

### Linux/macOS
```
sudo mv m2apps /usr/local/bin/m2apps
```

---

### Windows
```
Copy-Item m2apps.exe "C:\Program Files\M2Code\m2apps.exe"
```

---

## 6. PATH Setup

### Linux/macOS
Already included in /usr/local/bin

---

### Windows
Add to PATH via PowerShell:
```
[Environment]::SetEnvironmentVariable(
 "Path",
 $env:Path + ";C:\Program Files\M2Code",
 [EnvironmentVariableTarget]::Machine
)
```

---

## 7. Validation

After install:
```
m2apps --version
```

---

## 8. Error Handling

- network failure → show clear message
- unsupported OS → exit
- permission error → suggest sudo/admin

---

## 9. Idempotency

If already installed:
- overwrite existing binary
- no failure

---

## Summary

Installer flow:
1. Detect OS/ARCH
2. Fetch release metadata
3. Download correct binary
4. Move to system path
5. Set permissions
6. Validate installation
