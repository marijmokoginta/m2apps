# M2Apps Phase 4 — Final Bug Fix (GitHub Private Release Download)

## Context

Download asset dari GitHub release gagal (404) saat menggunakan:
- browser_download_url
- direct github.com/releases/download URL

Padahal:
- API release berhasil
- token valid
- repo private

---

## Root Cause

URL berikut TIDAK reliable untuk private repo:

https://github.com/{owner}/{repo}/releases/download/{tag}/{asset}

Karena:
- Endpoint ini bukan API
- Tidak selalu menerima Bearer token
- Auth berbasis cookie (browser), bukan token

---

## Solution (FINAL)

Gunakan field `assets.url` dari GitHub API:

Contoh response:
```
"assets": [
  {
    "id": 123456,
    "name": "app.zip",
    "url": "https://api.github.com/repos/{owner}/{repo}/releases/assets/123456"
  }
]
```

---

## Implementation Steps

### 1. Ambil asset URL dari response

```go
assetURL := asset.URL
```

---

### 2. Request download via API (WAJIB pakai header ini)

```go
req, err := http.NewRequest("GET", assetURL, nil)
if err != nil {
    return err
}

req.Header.Set("Authorization", "Bearer "+token)
req.Header.Set("Accept", "application/octet-stream")
```

---

### 3. Gunakan HTTP client dengan redirect handling

```go
client := &http.Client{
    CheckRedirect: func(req *http.Request, via []*http.Request) error {
        return http.ErrUseLastResponse
    },
}
```

---

### 4. Execute request

```go
resp, err := client.Do(req)
if err != nil {
    return err
}
defer resp.Body.Close()
```

---

### 5. Handle response

#### Case A: Direct file (200 OK)

```go
if resp.StatusCode == http.StatusOK {
    // langsung stream ke file
}
```

---

#### Case B: Redirect (302)

```go
if resp.StatusCode == http.StatusFound {
    redirectURL := resp.Header.Get("Location")

    req2, _ := http.NewRequest("GET", redirectURL, nil)
    resp2, _ := http.DefaultClient.Do(req2)

    defer resp2.Body.Close()

    if resp2.StatusCode != http.StatusOK {
        return fmt.Errorf("download failed: %d", resp2.StatusCode)
    }

    // stream resp2.Body ke file
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

_, err = io.Copy(out, resp.Body)
if err != nil {
    return err
}
```

---

## Rules

- Jangan gunakan browser_download_url untuk private repo
- Jangan gunakan github.com/releases/download
- WAJIB gunakan:
  - assets.url
  - header Authorization
  - header Accept: application/octet-stream
- Tetap handle redirect

---

## Debug Output (Optional)

```go
fmt.Println("Status:", resp.Status)
fmt.Println("Asset URL:", assetURL)
```

---

## Expected Result

```
Fetching latest release...
Found version: v1.0.0

Downloading app.zip...
[██████████] 100%

Download completed.
```

---

## Summary

Gunakan:
- API endpoint asset (assets.url)

Bukan:
- browser_download_url

Ini adalah cara resmi GitHub untuk download asset dari private repository.
