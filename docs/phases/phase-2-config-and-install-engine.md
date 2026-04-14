# M2Apps — Phase 2 Implementation Guide (Config & install.json Engine)

## Objective
Implement config loader for `install.json` including:
- File loading from current working directory
- JSON parsing into Go struct
- Basic validation
- CLI output summary

---

## Scope Rules

### MUST IMPLEMENT
- Load `install.json`
- Parse JSON → struct
- Validate required fields
- Print readable CLI output

### MUST NOT IMPLEMENT
- Requirement checking
- GitHub API / downloading
- Command execution
- Storage / encryption
- Token handling
- Daemon / background service

---

## Project Structure Update

```
internal/
  config/
    loader.go
    types.go
    validate.go
```

---

## Step 1 — Define Config Types

### File: internal/config/types.go

```go
package config

type InstallConfig struct {
    AppID   string       `json:"app_id"`
    Name    string       `json:"name"`
    Source  SourceConfig `json:"source"`
    Preset  string       `json:"preset"`
}

type SourceConfig struct {
    Type    string `json:"type"`
    Repo    string `json:"repo"`
    Version string `json:"version"`
    Asset   string `json:"asset"`
}
```

---

## Step 2 — Implement Loader

### File: internal/config/loader.go

```go
package config

import (
    "encoding/json"
    "fmt"
    "os"
)

func LoadFromFile(path string) (*InstallConfig, error) {
    file, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read %s: %w", path, err)
    }

    var cfg InstallConfig
    if err := json.Unmarshal(file, &cfg); err != nil {
        return nil, fmt.Errorf("invalid JSON format: %w", err)
    }

    return &cfg, nil
}
```

---

## Step 3 — Implement Validation

### File: internal/config/validate.go

```go
package config

import "fmt"

func (c *InstallConfig) Validate() error {
    var errors []string

    if c.AppID == "" {
        errors = append(errors, "app_id is required")
    }

    if c.Source.Type == "" {
        errors = append(errors, "source.type is required")
    }

    if c.Source.Repo == "" {
        errors = append(errors, "source.repo is required")
    }

    if c.Preset == "" {
        errors = append(errors, "preset is required")
    }

    if len(errors) > 0 {
        return fmt.Errorf("config validation failed:\n- %s", 
            joinErrors(errors))
    }

    return nil
}

func joinErrors(errs []string) string {
    result := ""
    for i, e := range errs {
        if i == 0 {
            result += e
        } else {
            result += "\n- " + e
        }
    }
    return result
}
```

---

## Step 4 — Integrate with CLI (install command)

### Update: cmd/install.go

```go
package cmd

import (
    "fmt"
    "m2apps/internal/config"

    "github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
    Use:   "install",
    Short: "Install application from install.json",
    Run: func(cmd *cobra.Command, args []string) {

        fmt.Println("Reading install.json...")

        cfg, err := config.LoadFromFile("install.json")
        if err != nil {
            fmt.Println("Error:", err)
            return
        }

        if err := cfg.Validate(); err != nil {
            fmt.Println("Error in install.json:")
            fmt.Println(err)
            return
        }

        fmt.Println("✔ Config loaded")
        fmt.Printf("✔ App: %s\n", cfg.Name)
        fmt.Printf("✔ Preset: %s\n", cfg.Preset)
    },
}
```

---

## Step 5 — Sample install.json (for testing)

Create file in project root:

```json
{
  "app_id": "m2code-project",
  "name": "M2Code Project App",
  "source": {
    "type": "github-release",
    "repo": "username/repo",
    "version": "latest",
    "asset": "app.zip"
  },
  "preset": "laravel-inertia"
}
```

---

## Expected CLI Output

```
m2apps install

Reading install.json...
✔ Config loaded
✔ App: Pengaduan Desa
✔ Preset: laravel-inertia
```

---

## Error Example

```
Error in install.json:
- app_id is required
- source.repo is required
```

---

## Done Criteria

- install.json successfully loaded
- JSON parsed into struct
- validation working
- readable CLI output
- no panic or crash
- no business logic added beyond config

---

## Notes

- Keep implementation minimal and clean
- This phase is foundation for all next phases
- Do NOT over-engineer
