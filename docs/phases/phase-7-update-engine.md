# M2Apps — Phase 7 Implementation Guide (Update Engine + Channel Strategy)

## Context

Phase 6 has completed secure encrypted metadata storage and installer integration fileciteturn1file6.

Phase 7 introduces the **Update Engine**, including:

- release channel support (stable, beta, alpha)
- version comparison
- GitHub release filtering
- update execution pipeline
- metadata reuse

---

## Objectives

Implement a complete update system that:

- reads encrypted metadata
- determines update eligibility
- filters release based on channel
- downloads and installs new version safely
- updates stored metadata

---

## Key Concepts

### 1. Channel System

Each application is bound to a single update channel:

- stable → production
- beta → testing
- alpha → experimental

Channel must be stored in metadata.

---

## Required Changes

### Update AppConfig (internal/storage/model.go)

Add:

```go
Channel string // "stable" | "beta" | "alpha"
```

---

### Update Install Flow

When saving metadata during install:

- read `channel` from install.json
- default to "stable" if not provided
- persist into encrypted config

---

## Module Structure

Create:

```
internal/updater/
  updater.go
  channel.go
  version.go
```

---

## Channel Filtering Logic

### channel.go

Implement:

```go
func MatchChannel(r Release, channel string) bool {
    switch channel {
    case "stable":
        return !r.Prerelease

    case "beta":
        return r.Prerelease && strings.Contains(r.TagName, "beta")

    case "alpha":
        return r.Prerelease && strings.Contains(r.TagName, "alpha")

    default:
        return false
    }
}
```

---

## Version Comparison

### version.go

Use semantic version comparison.

Rules:

- 1.0.0-alpha < 1.0.0-beta < 1.0.0
- ignore non-matching channel versions

You may:
- implement simple parser
- or use semver library

---

## Updater Core

### updater.go

Main flow:

```go
func Update(appID string) error {
    // 1. Load metadata
    config, err := storage.Load(appID)

    // 2. Fetch releases
    releases := github.FetchAllReleases(config.Repo, config.Token)

    // 3. Filter by channel
    var target Release
    for _, r := range releases {
        if MatchChannel(r, config.Channel) {
            target = r
            break
        }
    }

    // 4. Compare version
    if !IsNewer(target.TagName, config.Version) {
        fmt.Println("Already up to date")
        return nil
    }

    // 5. Download
    file := downloader.Download(target.AssetURL)

    // 6. Reuse installer pipeline
    installer.Run(file, config.InstallPath, config.Preset)

    // 7. Update metadata
    config.Version = target.TagName
    storage.Save(appID, config)

    return nil
}
```

---

## CLI Integration

### cmd/update.go

Update behavior:

```bash
m2apps update <app_id>
```

Flow:

1. Validate app exists
2. Call updater.Update(app_id)
3. Print progress output

---

## CLI Output Example

```
Checking update (channel: stable)...

Current: v1.0.0
Latest:  v1.1.0

[INFO] Downloading update...
[OK] Update completed
```

---

## Safety Rules

- Do NOT mix channels
- Do NOT downgrade version
- Stop on any failure
- Do NOT overwrite existing app unless install succeeds

---

## Error Handling

Must handle:

- metadata load failure
- GitHub API failure
- no matching release
- version parsing failure
- download failure
- install failure

All errors must be clear and actionable.

---

## Optional Enhancement (Not Required)

- channel switch command:
  m2apps channel set <app_id> beta

---

## Done Criteria

- update command works end-to-end
- channel filtering works correctly
- version comparison prevents downgrade
- metadata updates after success
- no breaking existing install system

---

## Notes

- reuse Phase 4 GitHub client fileciteturn1file4
- reuse Phase 5 installer pipeline fileciteturn1file5
- reuse Phase 6 storage system fileciteturn1file6

Do NOT duplicate logic.
Integrate cleanly.
