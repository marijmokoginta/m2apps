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

## Storage Paths

M2Apps stores runtime data inside the user home directory.

- Linux: `$HOME/.m2apps`
- macOS: `$HOME/.m2apps`
- Windows: `%USERPROFILE%\\.m2apps`

Main subdirectories:

- `apps/` - encrypted app metadata per `app_id`
- `daemon/` - daemon runtime data (pid, port, app registry)
- `logs/` - installer logs

## Usage

```bash
m2apps install
m2apps update <app_id>
m2apps delete <app_id>
m2apps list
m2apps daemon install
m2apps daemon start
m2apps daemon status
m2apps daemon stop
```

## Release Workflow (GitHub)

The release workflow is defined in `.github/workflows/release.yml` and runs automatically when a tag matching `v*` is pushed.

### Prerequisites

```bash
git remote -v
```

Make sure the `origin` remote points to the correct GitHub repository.

### How to Trigger Release

1. Commit the changes you want to release.
2. Push the main branch to GitHub.
3. Create a semantic version tag (`v1.0.0`, `v1.1.0`, etc.).
4. Push the tag to GitHub.

Example:

```bash
git add -A
git commit -m "Prepare release v1.0.0"
git push origin main

git tag v1.0.0
git push origin v1.0.0
```

After the tag is pushed:

- GitHub Actions will build the following targets:
  - `windows-amd64`
  - `linux-amd64`
  - `darwin-amd64`
- The workflow will create a GitHub Release with these assets:
  - `m2apps-windows-amd64.zip`
  - `m2apps-linux-amd64.tar.gz`
  - `m2apps-darwin-amd64.tar.gz`

### Local Build (Manual)

```bash
./scripts/build-release.sh
```

This script generates binaries and release archives in the `dist/` directory.

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
