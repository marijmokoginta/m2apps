# M2Apps Phase 7 Adjustment — Install Command Channel Alignment

## Objective
Align `install` command behavior with `update` logic by:
- Removing dependency on GitHub `/latest` endpoint
- Using channel-based filtering (stable/beta/alpha)
- Selecting highest version within selected channel

---

## Background

Previously:
- Install uses: `/releases/latest`
- Update uses: `/releases` + filtering

Problem:
- GitHub `/latest` ignores prerelease
- Cannot install beta/alpha directly
- Inconsistent behavior between install and update

---

## Required Changes

### 1. Remove Latest Release Usage

❌ REMOVE:
- Any usage of:
  GET /repos/{owner}/{repo}/releases/latest

---

### 2. Fetch All Releases

✅ USE:
GET /repos/{owner}/{repo}/releases

---

### 3. Add Channel Support to Install Flow

#### Update config model:
Add field:
```
Channel string
```

#### Default:
- If empty → use `"stable"`

---

### 4. Implement Channel Filtering Logic

Create function:
```
func FilterReleasesByChannel(releases []Release, channel string) []Release
```

Rules:

- stable:
  - prerelease == false

- beta:
  - prerelease == true
  - tag contains "beta"

- alpha:
  - prerelease == true
  - tag contains "alpha"

---

### 5. Select Highest Version

Implement:
```
func GetLatestVersion(releases []Release) Release
```

Requirements:
- Use semantic version comparison
- Ignore invalid tags
- Sort descending
- Pick highest

---

### 6. Update Install Flow

Replace existing logic with:

1. Load config
2. Determine channel (default stable)
3. Fetch all releases
4. Filter by channel
5. Select highest version
6. Resolve asset
7. Download + continue existing install pipeline

---

### 7. CLI Output Update

Add:
```
[INFO] Channel: <channel>
[INFO] Resolving latest version...
[OK] Selected version: <tag>
```

---

### 8. Error Handling

Add errors:

- No releases found:
  "No releases available for channel"

- No matching asset:
  "Asset not found in selected release"

- Invalid version:
  "Invalid version format"

---

### 9. Reuse Existing Update Logic

IMPORTANT:
- DO NOT duplicate logic
- Extract shared logic into:
  internal/github/release_selector.go

Shared functions:
- fetch releases
- filter by channel
- pick latest

---

## Expected Result

Install behavior becomes:

| Channel | Installed Version |
|--------|------------------|
| stable | latest stable |
| beta   | latest beta |
| alpha  | latest alpha |

---

## Done Criteria

- Install works for all channels
- No usage of `/latest` endpoint
- Version selection consistent with update
- No regression in existing install pipeline
- `go build ./...` passes

---

## Notes

This ensures:
- consistent behavior between install and update
- support for prerelease installation
- removal of GitHub API limitation
