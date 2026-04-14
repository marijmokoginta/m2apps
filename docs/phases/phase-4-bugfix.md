# M2Apps Phase 4 — Bug Fix: GitHub Release Download (Private Repo)

## Context

- Request 1 (fetch release) berhasil
- Request 2 (download asset via browser_download_url) gagal (404 / timeout)

## Root Cause

GitHub menggunakan redirect (302) ke CDN:

GitHub → redirect → objects.githubusercontent.com

Masalah:
- Go HTTP client mengikuti redirect otomatis
- Authorization header dihapus saat redirect
- CDN tidak menerima request tanpa auth → 404

---

## Solution

Tangani redirect secara manual.

---

## Implementation Steps

### 1. Disable auto redirect

```go
client := &http.Client{
    CheckRedirect: func(req *http.Request, via []*http.Request) error {
        return http.ErrUseLastResponse
    },
}
```

---

### 2. Request pertama (pakai token)

```go
req, err := http.NewRequest("GET", downloadURL, nil)
if err != nil {
    return err
}

req.Header.Set("Authorization", "Bearer "+token)
req.Header.Set("Accept", "application/octet-stream")

resp, err := client.Do(req)
if err != nil {
    return err
}
defer resp.Body.Close()
```

---

### 3. Handle redirect (302)

```go
if resp.StatusCode != http.StatusFound {
    return fmt.Errorf("expected redirect (302), got %d", resp.StatusCode)
}

redirectURL := resp.Header.Get("Location")
if redirectURL == "" {
    return fmt.Errorf("missing redirect location")
}
```

---

### 4. Request kedua ke CDN (tanpa token)

```go
req2, err := http.NewRequest("GET", redirectURL, nil)
if err != nil {
    return err
}

resp2, err := http.DefaultClient.Do(req2)
if err != nil {
    return err
}
defer resp2.Body.Close()
```

---

### 5. Validasi response

```go
if resp2.StatusCode != http.StatusOK {
    return fmt.Errorf("download failed: status %d", resp2.StatusCode)
}
```

---

### 6. Stream ke file

```go
out, err := os.Create(destPath)
if err != nil {
    return err
}
defer out.Close()

_, err = io.Copy(out, resp2.Body)
if err != nil {
    return err
}
```

---

## Rules

- Jangan gunakan http.Get() langsung
- Jangan biarkan redirect otomatis
- Jangan kirim token ke CDN
- Jangan asumsi status selalu 200

---

## Debug

Tambahkan sementara:

```go
fmt.Println("Status:", resp.Status)
fmt.Println("Redirect:", redirectURL)
```

Expected:
```
Status: 302 Found
Redirect: https://objects.githubusercontent.com/...
```

---

## Expected Result

```bash
Fetching latest release...
Found version: v1.2.0

Downloading app.zip...
[██████████] 100%

Download completed.
```

---

## Notes

Ini bukan bug random, tapi behavior normal:
- GitHub redirect
- HTTP client drop header

Harus ditangani manual agar stabil.
