# M2Apps — Phase 10 Patch (v1.0.1) Codex Instructions

## Overview
This patch focuses on UX enhancement, release automation, and installer setup:
1. Interactive CLI (root command)
2. Changelog-based GitHub release notes
3. Cross-platform installer scripts

---

## 1. INTERACTIVE CLI (ROOT COMMAND)

### Goal
Transform `m2apps` root command into an interactive menu.

---

### Dependency
Use:
- github.com/charmbracelet/bubbletea
- github.com/charmbracelet/lipgloss

Install:
```
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/lipgloss
```

---

### Behavior

Running:
```
m2apps
```

Should display:

- Banner
- Version (v1.0.1)
- Interactive menu:
  - Install Application
  - Update Application
  - Switch Channel

Static section:
- list
- daemon
- help

---

### Navigation
- ↑ / ↓ → move selection
- Enter → execute
- Highlight active item using INFO color

---

### Implementation

Create:
```
internal/ui/menu.go
```

Define model:
```go
type MenuItem struct {
    Title string
    Action string
}
```

Menu list:
```go
[]MenuItem{
    {"Install Application", "install"},
    {"Update Application", "update"},
    {"Switch Channel", "channel"},
}
```

---

### Update Flow

When "Update Application" selected:
1. Load installed apps from storage
2. Display selectable list
3. On select:
   call existing update command logic

---

### Channel Flow

1. Select app
2. Select channel:
   - stable
   - beta
   - alpha
3. Execute existing channel logic

---

### Edge Cases
- No apps → show message and return
- Escape key → exit menu

---

## 2. CHANGELOG INTEGRATION (GITHUB ACTIONS)

### Goal
Use local changelog file as release note.

---

### File Structure
```
changelog/
  v1.0.1.md
```

---

### GitHub Workflow Step

Add:

```yaml
- name: Read changelog
  id: changelog
  run: |
    FILE="changelog/${{ github.ref_name }}.md"
    if [ ! -f "$FILE" ]; then
      echo "Changelog file not found!"
      exit 1
    fi

    echo "body<<EOF" >> $GITHUB_OUTPUT
    cat $FILE >> $GITHUB_OUTPUT
    echo "EOF" >> $GITHUB_OUTPUT
```

---

### Use in Release

```yaml
body: ${{ steps.changelog.outputs.body }}
```

---

## 3. INSTALLER SCRIPT (MULTI PLATFORM)

### Goal
Allow client to install m2apps globally.

---

## WINDOWS INSTALLER

### File
```
install.ps1
```

### Script
```powershell
$target = "C:\Program Files\M2Code"

New-Item -ItemType Directory -Force -Path $target
Copy-Item ".\m2apps.exe" "$target\m2apps.exe"

$currentPath = [Environment]::GetEnvironmentVariable("Path", [EnvironmentVariableTarget]::Machine)

if ($currentPath -notlike "*$target*") {
    [Environment]::SetEnvironmentVariable(
        "Path",
        $currentPath + ";" + $target,
        [EnvironmentVariableTarget]::Machine
    )
}

Write-Host "M2Apps installed successfully."
```

---

## LINUX / MAC INSTALLER

### File
```
install.sh
```

### Script
```bash
#!/bin/bash

set -e

TARGET="/usr/local/bin/m2apps"

sudo cp m2apps $TARGET
sudo chmod +x $TARGET

echo "M2Apps installed successfully."
```

---

### Make executable
```
chmod +x install.sh
```

---

## DISTRIBUTION STRUCTURE

```
release/
  windows/
    m2apps.exe
    install.ps1
  linux/
    m2apps
    install.sh
  macos/
    m2apps
    install.sh
```

---

## VALIDATION CHECKLIST

- `m2apps` shows interactive menu
- Arrow navigation works
- Update flow selectable
- Channel switching works
- Changelog correctly used in release
- Installer works on each OS
- Binary accessible via PATH

---

## RULES

- Do not break existing commands
- Keep CLI fallback if interactive fails
- Do not over-engineer UI
- Keep installer simple

---

## DONE CRITERIA

- Interactive CLI functional
- Release notes automated
- Installer works on Windows, Linux, macOS
- No regression in existing features
