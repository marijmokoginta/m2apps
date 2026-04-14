# M2Apps Phase 6 Summary

Date: 2026-04-14

## Scope Implemented

Phase 6 introduces secure metadata storage with encryption and installer integration.

Implemented features:

- encrypted metadata persistence using AES-256 GCM
- storage interface with save/load operations
- token protection in encrypted config file
- machine-based key derivation
- post-install metadata write
- cleanup of `install.json` after successful installation

## Modules Added

- `internal/storage/model.go`
- `internal/storage/encrypt.go`
- `internal/storage/decrypt.go`
- `internal/storage/storage.go`

## Data Model

`AppConfig` now stores:

- `AppID`
- `Name`
- `InstallPath`
- `Repo`
- `Asset`
- `Token`
- `Version`
- `Preset`

## Encryption Strategy

- Algorithm: AES-256 GCM
- Nonce: cryptographically random per encryption
- Key derivation:
  - `SHA256(hostname + user + static_salt)`

Flow:

1. Serialize config to JSON
2. Encrypt JSON bytes
3. Write encrypted bytes to `config.enc`

Decryption reverses the same process for `Load(appID)`.

## Storage Location

Base directory:

- `~/.m2apps/`

Per app directory:

- `~/.m2apps/apps/{app_id}/config.enc`
- `~/.m2apps/apps/{app_id}/state.json`

## Installer Integration

Updated `cmd/install.go` after successful install pipeline:

1. Initialize file storage
2. Build `storage.AppConfig` from current install context
3. Save encrypted metadata using `Save(appID, data)`
4. Remove `install.json`

## Security Notes

- Token is never printed to CLI output.
- Token is persisted only inside encrypted `config.enc`.
- Plaintext `install.json` is removed after success.

## Error Handling

Install now fails with explicit error output when any of the following fails:

- key derivation
- encryption/decryption
- storage directory creation
- config write/read
- JSON serialization/deserialization

`install.json` removal failure is reported as warning and does not invalidate completed installation.

## Validation Snapshot

Completed checks:

- `go build ./...` passed after Phase 6 implementation
- storage module compiles and integrates with install flow
- encrypted persistence path is created under `~/.m2apps/apps/{app_id}`

## Reference

- This summary accompanies the Phase 6 storage and security implementation commit.
