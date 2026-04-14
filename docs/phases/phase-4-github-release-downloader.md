# M2Apps Phase 4 — GitHub Release Downloader (Codex Instruction)

## Overview
Phase 4 introduces the GitHub Release integration and downloader system.

Goal:
- Fetch release data from GitHub API
- Resolve asset (e.g., app.zip)
- Download file using streaming (no full memory load)
- Show progress in CLI

---

## Scope

### Included
- GitHub API client
- Token authentication
- Release parsing
- Asset selection
- File downloader (stream-based)
- CLI progress output

### Excluded
- Extraction (Phase 5)
- Install execution
- Preset execution
- Storage/encryption

---

## Project Structure

Create new directories:

internal/github/
internal/downloader/

---

## Files to Create

### GitHub Module
- internal/github/client.go
- internal/github/types.go
- internal/github/release.go

### Downloader Module
- internal/downloader/downloader.go
- internal/downloader/progress.go

---

## GitHub Client

### Interface

type Client interface {
    GetLatestRelease(owner, repo string) (*Release, error)
    GetReleaseByTag(owner, repo, tag string) (*Release, error)
}

---

### Release Struct

type Release struct {
    TagName string
    Assets  []Asset
}

type Asset struct {
    Name string
    BrowserDownloadURL string
}

---

## API Endpoints

Latest:
GET https://api.github.com/repos/{owner}/{repo}/releases/latest

By tag:
GET https://api.github.com/repos/{owner}/{repo}/releases/tags/{tag}

---

## Headers

Authorization: Bearer <TOKEN>
Accept: application/vnd.github+json

---

## Repo Parsing

Input:
"username/repo"

Implementation:
- Split string by "/"
- Validate length == 2

---

## Asset Selection

Loop through release.Assets and match by:
asset.Name == config.Source.Asset

Return error if not found.

---

## Downloader

### Requirements
- Use streaming (io.Copy)
- Do not load full file into memory
- Create destination file (e.g., ./app.zip)

---

### Basic Flow

resp, err := http.Get(url)
defer resp.Body.Close()

file, err := os.Create(dest)
defer file.Close()

io.Copy(file, resp.Body)

---

## Progress Tracking

Implement custom reader:

type ProgressReader struct {
    Reader io.Reader
    Total  int64
    Read   int64
}

- Wrap resp.Body
- Track bytes read
- Calculate percentage

---

### CLI Output Example

Downloading app.zip...
[██████░░░░] 60% (45MB / 70MB)

---

## Error Handling

Must handle:

- 401 Unauthorized (invalid token)
- 404 Not Found (repo or release missing)
- 403 Forbidden (rate limit)
- Asset not found
- Download interruption

Error messages must be human-readable.

---

## Integration (cmd/install.go)

After requirement check:

1. Parse repo
2. Fetch release (latest or tag)
3. Find asset
4. Download asset

---

### Example Flow

fmt.Println("Fetching release...")

release := client.GetLatestRelease(...)

fmt.Println("Found version:", release.TagName)

asset := findAsset(...)

fmt.Println("Downloading", asset.Name)

downloader.Download(asset.URL, "./app.zip")

fmt.Println("Download completed.")

---

## CLI Expected Output

Checking requirements...
[✓] PHP >= 8.1

Fetching latest release...
Found version: v1.2.0

Downloading app.zip...
[██████████] 100%

Download completed.

---

## Implementation Order

1. GitHub client (no downloader)
2. Release parsing
3. Asset resolver
4. Basic downloader
5. Add progress
6. Integrate into install command

---

## Rules

- Do not implement extraction
- Do not modify Phase 3 logic
- Keep modules isolated
- Use clear error messages
- Follow Go best practices

---

## Done Criteria

- Can fetch release from GitHub
- Can resolve correct asset
- Can download file successfully
- CLI shows progress
- Errors handled cleanly
