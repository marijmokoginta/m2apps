# M2Apps Requirement Installation Strategy (Design Guide)

## Overview
This document defines the architecture and strategy for handling system requirements (PHP, Node, MySQL, etc.) in M2Apps.

Goal:
- Simplify installation for non-technical users
- Avoid full environment bundling (e.g. XAMPP)
- Provide guided, semi-automatic installation

---

## Core Principles

1. Do NOT auto-install silently
2. Always ask user confirmation
3. Install per-tool (not bundled stacks)
4. Be OS-aware (Windows/Linux/macOS)
5. Always re-check after installation
6. Provide fallback manual instructions

---

## High-Level Flow

1. Read requirements from install.json
2. Run requirement checks
3. Identify missing tools
4. Build install plan
5. Ask user confirmation
6. Execute installation per tool
7. Re-check installation
8. Continue install process

---

## Requirement Source

Example:
```json
{
  "requirements": [
    { "type": "php", "version": ">=8.1" },
    { "type": "node", "version": ">=18" }
  ]
}
```

---

## Install Plan Concept

Instead of installing immediately, build a plan:

Example:
```
Missing:
- PHP >=8.1
- MySQL >=8.0

Install Plan:
1. PHP 8.2
2. MySQL 8.0
```

---

## Version Resolution

Convert constraints into actual install version:

- >=8.1 → latest compatible (e.g. 8.2)
- Avoid hardcoding versions
- Always prefer stable versions

---

## Installation Types

### 1. CLI-based
- Linux: apt/yum
- macOS: brew

### 2. Portable
- Download ZIP
- Extract
- Optional PATH setup

### 3. GUI Installer
- Download .exe/.dmg
- Launch installer
- Wait for user completion

---

## OS Strategy

### Windows
- Prefer portable (ZIP) if available
- Otherwise use official installer (.exe)
- Optionally update PATH

### Linux
- Use curl or device package manager (apt, yum)

### macOS
- Use Homebrew

---

## Execution Flow

```
Check → Missing → Plan → Confirm → Install → Re-check
```

---

## UX Example

```
Missing tools detected:

[X] PHP >= 8.1
[X] MySQL >= 8.0

Install plan:

> Install PHP 8.2
  Install MySQL 8.0

Proceed? (y/n)
```

---

## Post Install Validation

After each install:

```
Re-checking PHP...

[✓] PHP 8.2 installed
```

If failed:
```
Installation failed. Please install manually.
```

---

## Edge Cases

- Installer closed early
- PATH not updated
- Requires restart
- Permission denied

Handle with clear messages.

---

## Safety Boundaries

Do NOT:
- Configure services (MySQL root password, etc.)
- Modify system configs aggressively
- Run silent installers

Only:
- Ensure tool is installed and callable

---

## Optional Config

```json
{
  "install_mode": "assisted"
}
```

Modes:
- assisted (default)
- manual
- future: auto

---

## Future Enhancements

- m2apps doctor command
- install profiles (quick vs full)
- portable runtime mode (SQLite, etc.)

---

## Summary

M2Apps acts as:
- Requirement checker
- Guided installer
- Environment assistant

NOT:
- Full package manager
- Full container system

---

## Final Insight

The goal is not to control the system,
but to guide the user safely and effectively.
