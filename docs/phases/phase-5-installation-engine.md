# M2Apps Phase 5 — Codex Implementation Instructions

## Context

Phase 4 has successfully implemented GitHub release downloading, including private repository support and streaming downloads.

Reference:
- Phase 4 Summary → fileciteturn1file4

This phase focuses on transforming the downloaded artifact into a working application.

---

# 🎯 Objective

Implement the **Installation Engine**, including:

1. Zip extraction
2. Temporary directory handling
3. Preset execution (command runner)
4. Installation pipeline orchestration
5. Basic failure handling

---

# 📁 Required Structure

Create the following modules:

```
internal/
  extractor/
    zip.go

  preset/
    runner.go
    registry.go

  installer/
    installer.go
```

---

# 1. ZIP EXTRACTION

## File:
`internal/extractor/zip.go`

## Function:
```go
func ExtractZip(src string, dest string) error
```

## Requirements:
- Extract all files from zip
- Preserve directory structure
- Create destination folder if not exists

## Security:
Prevent Zip Slip:

```go
if !strings.HasPrefix(destPath, filepath.Clean(dest)+string(os.PathSeparator)) {
    return fmt.Errorf("invalid file path")
}
```

---

# 2. TEMP DIRECTORY STRATEGY

## Rule:
DO NOT extract directly to target directory.

## Use:
```
.m2apps_tmp/{app_id}/
```

## Flow:
- Extract to temp directory
- Run installation steps there
- Move to final destination only if success

---

# 3. PRESET SYSTEM

## File:
`internal/preset/registry.go`

## Define:
```go
type Step struct {
    Type string
    Run  string
}
```

## Preset Example:
```go
var Presets = map[string][]Step{
    "laravel-inertia": {
        {Type: "command", Run: "composer install"},
        {Type: "command", Run: "php artisan key:generate"},
        {Type: "command", Run: "php artisan migrate --force"},
        {Type: "command", Run: "npm install"},
        {Type: "command", Run: "npm run build"},
    },
}
```

## Function:
```go
func GetPreset(name string) ([]Step, error)
```

---

# 4. STEP RUNNER

## File:
`internal/preset/runner.go`

## Function:
```go
func RunSteps(steps []Step, workDir string) error
```

## Behavior:
- Execute steps sequentially
- Use OS command execution
- Print progress

## Example:
```
[1/5] composer install
[2/5] php artisan migrate
```

## Rules:
- Stop on first error
- Return clear error message

---

# 5. INSTALL PIPELINE

## File:
`internal/installer/installer.go`

## Struct:
```go
type InstallContext struct {
    ZipPath   string
    TargetDir string
    Preset    string
    AppID     string
}
```

## Function:
```go
func Install(ctx InstallContext) error
```

## Flow:

1. Create temp directory
2. Extract zip into temp
3. Load preset
4. Run preset steps inside temp
5. Move files to target directory
6. Cleanup temp

---

# 6. FAILURE HANDLING

## Requirements:
- Stop execution on failure
- Print error with step context
- Do NOT move files if failed
- Cleanup temp directory

## Example:
```
Installation failed at step: npm run build
```

---

# 7. CLI INTEGRATION

Update `cmd/install.go`:

After Phase 4 download:

```go
ctx := installer.InstallContext{
    ZipPath: downloadedFile,
    TargetDir: cwd,
    Preset: config.Preset,
    AppID: config.AppID,
}

err := installer.Install(ctx)
```

---

# ✅ DONE CRITERIA

- Zip successfully extracted
- Preset commands executed
- App installed into working directory
- Failure handled cleanly
- No partial install corruption

---

# 🚫 OUT OF SCOPE

Do NOT implement:
- encryption/storage
- daemon
- update system
- rollback system

---

# 🧠 NOTES

- Keep logic modular
- Avoid hardcoding paths
- Ensure deterministic behavior
- Prepare for reuse in update phase

---

# FINAL RESULT

Command:

```
m2apps install
```

Will:

1. Download release (Phase 4)
2. Extract zip
3. Execute preset
4. Install application

---

This phase marks the transition from downloader to real installer.
