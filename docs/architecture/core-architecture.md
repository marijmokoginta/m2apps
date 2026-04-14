# M2Apps — Full Architecture Contract (M2Code Updater Engine)

## 1. Overview
M2Apps adalah CLI-based cross-platform deployment agent yang berfungsi sebagai installer, updater, dan background service untuk aplikasi berbasis web (Laravel, Node, dll).

Tujuan utama:
- Mempermudah client non-teknis dalam install & update aplikasi
- Menghilangkan kebutuhan git & konfigurasi manual
- Menyediakan sistem distribusi berbasis artifact (GitHub Release)

---

## 2. Core Principles
- Client hanya input minimal (install.json)
- Distribution berbasis release (bukan repository)
- Secure by default (encrypted storage + token isolation)
- Multi-app support (isolated metadata)
- Framework agnostic (preset-based system)

---

## 3. System Components

### 3.1 CLI (m2apps)
Command utama:
- m2apps install
- m2apps update
- m2apps list
- m2apps remove
- m2apps status
- m2apps daemon start/stop

---

### 3.2 Background Daemon
- Berjalan sebagai service OS
- Menangani update, monitoring, dan API lokal

---

### 3.3 Local API (localhost)
Endpoint:
- /check-update
- /start-update
- /status

Autentikasi:
- Bearer Token (per app)

---

### 3.4 Encrypted Storage
Struktur:
~/.m2apps/
  apps/{app_id}/
    config.enc
    state.json
    logs/

---

## 4. Installation Architecture

### 4.1 Input
Client hanya menjalankan:
```
m2apps install
```

Dengan file:
```
install.json (di CWD)
```

---

### 4.2 install.json Structure
```
{
  "app_id": "pengaduan-desa",
  "name": "Pengaduan Desa",

  "source": {
    "type": "github-release",
    "repo": "username/repo",
    "version": "latest",
    "asset": "app.zip"
  },

  "auth": {
    "type": "token",
    "value": "ghp_xxx"
  },

  "preset": "laravel-inertia",

  "requirements": [
    { "type": "php", "version": ">=8.1" },
    { "type": "node", "version": ">=18" }
  ]
}
```

---

### 4.3 Installation Flow
1. Read install.json
2. Validate schema
3. Check requirements
4. Fetch GitHub release
5. Download asset
6. Extract to CWD
7. Run preset steps
8. Generate local API token
9. Inject .env config
10. Save encrypted metadata
11. Register app to daemon

---

## 5. Update Architecture

### 5.1 Flow
1. Load encrypted metadata
2. Decrypt token
3. Check latest release
4. Download new version
5. Atomic replace
6. Run update steps
7. Update metadata

---

## 6. GitHub Release Integration

### 6.1 API Flow
GET /repos/{owner}/{repo}/releases/latest

Header:
Authorization: Bearer <TOKEN>

---

### 6.2 Asset Handling
- Cari asset sesuai nama (app.zip)
- Ambil download URL
- Download via HTTP

---

### 6.3 Benefits
- Tidak perlu git
- Aman (read-only access)
- Versioned deployment

---

## 7. Security Model

### 7.1 Token Handling
- Token hanya digunakan saat install
- Disimpan encrypted
- Tidak disimpan plaintext

---

### 7.2 Local API Security
- Token per aplikasi
- Disimpan di:
  - encrypted storage (updater)
  - .env (app)

Header:
Authorization: Bearer TOKEN

---

### 7.3 Encryption
- AES-GCM
- Optional OS keychain

---

## 8. Preset System

### 8.1 Concept
Preset = workflow, bukan teknologi

---

### 8.2 Example: Laravel + Inertia
```
laravel-inertia:
- composer install
- php artisan key:generate
- php artisan migrate --force
- npm install
- npm run build
- php artisan config:clear
```

---

### 8.3 Execution Rules
- Sequential
- Stop on failure
- Logging per step

---

## 9. Requirement Check

### 9.1 Config
```
"requirements": [
  { "type": "php", "version": ">=8.1" }
]
```

---

### 9.2 Flow
- Execute check command
- Parse version
- Compare
- Display CLI result

---

## 10. Multi-App Management

- Setiap app punya:
  - config sendiri
  - token sendiri
  - path sendiri

---

## 11. Background Service

### OS Integration
- Windows → Service
- Linux → systemd
- macOS → launchd

---

## 12. Communication Flow

App → Local API → Updater → GitHub

---

## 13. Future Enhancements
- Rollback system
- GUI interface
- Plugin system
- Remote dashboard
- Auto update scheduler

---

## 14. Summary
M2Apps adalah:
- Installer
- Updater
- Deployment agent
- Distribution system

Semua dalam satu CLI tool.
