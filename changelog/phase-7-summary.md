# M2Apps Phase 7 Summary

Date: 2026-04-14

## Scope Implemented

Phase 7 delivers the update engine and channel strategy end-to-end, including:

- encrypted metadata reuse for updates
- channel-based release selection (`stable`, `beta`, `alpha`)
- semantic version comparison to prevent downgrade
- safe update installation pipeline
- optional channel switch command
- install/update alignment using shared release selection logic

## Core Update Engine

### Modules Added

- `internal/updater/updater.go`
- `internal/updater/version.go`

### GitHub Client Enhancements

- Added full release listing support:
  - `GetAllReleases(owner, repo)`
- Extended release model with prerelease metadata:
  - `Release.Prerelease`

### Update Command Integration

- `cmd/update.go` now supports:
  - `m2apps update <app_id>`
- Flow:
  1. Load encrypted metadata
  2. Resolve latest release for app channel
  3. Compare current vs target version
  4. Download selected asset
  5. Reuse installer pipeline in staging
  6. Safely replace installed app
  7. Persist updated version/channel metadata

## Channel Strategy

- Added channel persistence in metadata (`AppConfig.Channel`).
- Install defaults channel to `stable` when omitted from `install.json`.
- Update strictly follows app channel (no channel mixing).

Channel rules:

- `stable` → non-prerelease releases only
- `beta` → prerelease with `beta` in tag
- `alpha` → prerelease with `alpha` in tag

## Optional Enhancement: Channel Switch Command

### Command Added

- `m2apps channel set <app_id> <stable|beta|alpha>`

### Behavior

- Loads encrypted metadata for app
- Validates channel input
- Updates channel and saves encrypted metadata
- Returns clear `[OK]/[ERROR]` status

## Install Adjustment (Phase 7 Alignment)

To align install behavior with update behavior:

- Removed install dependency on GitHub `/releases/latest`
- Install now resolves version by:
  1. fetch all releases
  2. filter by channel
  3. pick highest valid semantic version

Added shared release selector module:

- `internal/github/release_selector.go`

Shared functions include:

- channel normalization and matching
- release filtering by channel
- latest release selection by semantic version
- semantic version comparator utilities

This shared logic is now reused by both install and update flows.

## Safety and Failure Handling

- No downgrade: update proceeds only if selected release is newer.
- Stops on any failure:
  - metadata load/save
  - release fetch/filter/selection
  - version parsing/comparison
  - asset resolution
  - download/install pipeline
- Existing app install path is updated via staged replacement with backup/rollback handling.

## Validation Snapshot

Completed checks during Phase 7 implementation:

- `go build ./...` passes
- `m2apps update --help` and command argument validation work
- `m2apps update <app_id>` fails cleanly when metadata is missing
- install flow no longer depends on `/latest` path

## Reference Commits

- `fbd6061` — Implement Phase 7 update engine with channel strategy, semantic version checks, release filtering, and update command integration
- `0e31f03` — Add optional channel switch command to set app update channel in encrypted metadata
- `42934fc` — Adjust Phase 7 install flow to use channel-based release selection from all releases and shared selector logic
