# M2Apps — Core Decisions

## Overview
Dokumen ini berisi keputusan arsitektur utama (core decisions) yang menjadi fondasi pengembangan M2Apps.
Keputusan ini bersifat fundamental dan akan mempengaruhi seluruh fase implementasi.

---

## 1. Distribution Strategy

### Decision
Menggunakan GitHub Release sebagai sumber distribusi aplikasi.

### Detail
- Repository bersifat private
- Distribusi dilakukan melalui release artifact (ZIP)
- Akses menggunakan read-only token

### Rationale
- Tidak memerlukan Git di sisi client
- Lebih aman dibanding memberikan akses repository penuh
- Mendukung versioning dan rollback
- Lebih cocok untuk client non-teknis

### Trade-offs
- Tidak ada incremental update (selalu download ulang)
- Bergantung pada GitHub API
- Perlu manajemen token yang aman

---

## 2. Configuration Source

### Decision
Menggunakan file lokal `install.json` di current working directory sebagai sumber konfigurasi.

### Detail
- Tidak menggunakan remote config
- Tidak memerlukan server tambahan
- File dikirim bersama installer ke client

### Rationale
- Lebih sederhana untuk distribusi awal
- Tidak bergantung pada koneksi eksternal untuk config
- Lebih mudah dikontrol oleh developer

### Trade-offs
- Tidak bisa dynamic update config
- Client tetap perlu menerima file awal
- Kurang cocok untuk skala besar

---

## 3. Preset-Based Execution

### Decision
Menggunakan sistem preset untuk menentukan workflow install/update.

### Detail
- Preset merepresentasikan workflow, bukan teknologi
- Contoh: laravel, node, laravel-inertia
- Preset dipilih melalui config, bukan auto-detect

### Rationale
- Menghindari kompleksitas auto-detection
- Lebih predictable dan controllable
- Mudah di-extend untuk berbagai stack

### Trade-offs
- Perlu maintain preset
- Tidak otomatis mengenali stack
- Butuh definisi awal dari developer

---

## 4. Artifact-Based Deployment

### Decision
Menggunakan artifact hasil build (ZIP) sebagai unit distribusi, bukan source code mentah.

### Detail
- Aplikasi sudah dalam kondisi siap pakai
- Build dilakukan sebelum release
- Client tidak perlu build ulang

### Rationale
- Mengurangi kompleksitas di sisi client
- Mempercepat proses install
- Menghindari dependency build di client

### Trade-offs
- Ukuran file lebih besar
- Perlu pipeline build di sisi developer
- Kurang fleksibel untuk debugging di client

---

## 5. Localhost API Communication

### Decision
Menggunakan localhost HTTP API untuk komunikasi antara aplikasi dan updater.

### Detail
- Endpoint berjalan di 127.0.0.1
- Menggunakan token authentication
- Digunakan untuk update dan status tracking

### Rationale
- Decoupled architecture
- Cross-platform friendly
- Tidak tergantung IPC kompleks

### Trade-offs
- Perlu manajemen port
- Perlu pengamanan token
- Potensi konflik port

---

## Summary

Keputusan inti M2Apps:
- Distribution: GitHub Release (private + token)
- Config: Local install.json
- Execution: Preset-based workflow
- Deployment: Artifact (ZIP)
- Communication: Localhost API

Semua keputusan ini bertujuan untuk:
- Menyederhanakan pengalaman client
- Mengurangi dependency teknis
- Menjaga fleksibilitas di sisi developer
