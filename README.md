# M2Apps

M2Apps is a cross-platform CLI installer, updater, and deployment agent for application packages published through GitHub Releases.

## Features

- Cross-platform CLI (Linux, macOS, Windows)
- Install flow from `install.json`
- Channel-based updater (`stable`, `beta`, `alpha`)
- GitHub Release asset download integration
- Encrypted local app metadata storage
- Background daemon process
- Localhost API for update orchestration and status polling

## Installation

1. Download the `m2apps` binary for your OS.
2. Put it in a directory that is available in your `PATH`.
3. Make sure the binary is executable (Linux/macOS).
4. Run:

```bash
m2apps
```

## Usage

```bash
m2apps install
m2apps update <app_id>
m2apps list
m2apps daemon install
m2apps daemon start
m2apps daemon status
m2apps daemon stop
```

## install.json Example

```json
{
  "app_id": "my_app",
  "name": "My Application",
  "source": {
    "type": "github",
    "repo": "owner/repository",
    "version": "latest",
    "asset": "my_app_windows.zip"
  },
  "auth": {
    "type": "token",
    "value": "ghp_your_github_token"
  },
  "channel": "stable",
  "preset": "nodejs",
  "requirements": [
    {
      "type": "node",
      "version": ">=18.0.0"
    },
    {
      "type": "npm",
      "version": ">=9.0.0"
    }
  ]
}
```

## Local API Documentation

Base URL:

```text
http://127.0.0.1:{PORT}
```

All endpoints require:

```http
Authorization: Bearer <M2APPS_API_TOKEN>
Content-Type: application/json
```

### GET /apps/{app_id}/update/check

Response:

```json
{
  "has_update": true,
  "current_version": "1.0.0",
  "latest_version": "1.1.0"
}
```

### POST /apps/{app_id}/update

Request body:

```json
{}
```

Response:

```json
{
  "started": true,
  "app_id": "my_app"
}
```

### GET /apps/{app_id}/update/status

Response:

```json
{
  "app_id": "my_app",
  "phase": "download",
  "step": "downloading update package",
  "percent": 42,
  "logs": [
    "Loading app metadata",
    "Resolving release for channel stable"
  ],
  "status": "running"
}
```

### POST /apps/{app_id}/channel

Request body:

```json
{
  "channel": "beta"
}
```

Response:

```json
{
  "updated": true,
  "app_id": "my_app",
  "channel": "beta"
}
```

### POST /apps/{app_id}/auth/update

Request body:

```json
{
  "token": "new_api_token"
}
```

Response:

```json
{
  "updated": true,
  "app_id": "my_app"
}
```

## Architecture Overview

- CLI layer (`cmd/`): user command entrypoint and command routing.
- Core modules (`internal/`): installer, updater, downloader, requirements, and configuration validation.
- Daemon/API layer: localhost service to accept app-triggered update requests and expose progress polling.
- Storage layer: encrypted metadata persistence in the local M2Apps directory.
- GitHub integration layer: release discovery and asset selection by update channel.

## Security

- GitHub and API tokens are stored in encrypted app metadata.
- Local API is bound to `127.0.0.1` only.
- API endpoints enforce bearer token validation per app.
- Environment injection does not overwrite existing variables.

## Troubleshooting

- Requirement failures:
  - Check dependency binaries (`node`, `npm`, `php`, etc.) are installed and available in `PATH`.
- Token issues:
  - Verify GitHub token can access repository releases.
  - Verify `Authorization: Bearer <token>` matches stored app API token.
- Permission issues:
  - Ensure M2Apps can write to local storage directory and installation path.
  - On Linux/macOS, verify executable permission for the `m2apps` binary.
