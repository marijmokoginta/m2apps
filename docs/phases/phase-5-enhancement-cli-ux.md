# M2Apps Phase 5 Enhancement — CLI UX, Logging, and Output Control

## 🎯 Objective

Enhance Phase 5 Installation Engine with:

1. Loading indicator (spinner)
2. Full external log suppression from terminal
3. External logs written to file
4. Colored CLI output based on status

---

# 📦 FEATURE OVERVIEW

| Feature | Description |
|--------|------------|
| Spinner | Show progress without blocking UX |
| Silent Execution | Hide noisy logs (npm, composer, etc.) |
| File Logging | Save all logs into file |
| Colored Output | Improve readability |

---

# 📁 STRUCTURE

```
internal/
  ui/
    spinner.go
    color.go

  logger/
    logger.go

  preset/
    runner.go (update)

  installer/
    installer.go (update)
```

---

# 🔥 1. SPINNER (LOADING INDICATOR)

## File:
`internal/ui/spinner.go`

```go
package ui

import (
	"fmt"
	"time"
)

type Spinner struct {
	stop chan bool
}

func NewSpinner() *Spinner {
	return &Spinner{stop: make(chan bool)}
}

func (s *Spinner) Start(message string) {
	go func() {
		chars := []string{"|", "/", "-", "\"}
		i := 0

		for {
			select {
			case <-s.stop:
				return
			default:
				fmt.Printf("\r%s %s", chars[i], message)
				time.Sleep(100 * time.Millisecond)
				i = (i + 1) % len(chars)
			}
		}
	}()
}

func (s *Spinner) Stop(message string) {
	s.stop <- true
	fmt.Printf("\r✔ %s\n", message)
}
```

---

# 🎨 2. COLOR OUTPUT

## File:
`internal/ui/color.go`

```go
package ui

const (
	Reset  = "\033[0m"
	Green  = "\033[32m"
	Red    = "\033[31m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
)

func Success(msg string) string {
	return Green + msg + Reset
}

func Error(msg string) string {
	return Red + msg + Reset
}

func Warning(msg string) string {
	return Yellow + msg + Reset
}

func Info(msg string) string {
	return Blue + msg + Reset
}
```

---

# 📜 3. LOGGER

## File:
`internal/logger/logger.go`

```go
package logger

import (
	"os"
	"path/filepath"
)

var logFile *os.File

func Init() error {
	home, _ := os.UserHomeDir()
	logDir := filepath.Join(home, ".m2apps", "logs")

	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return err
	}

	filePath := filepath.Join(logDir, "install.log")

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}

	logFile = f
	return nil
}

func Writer() *os.File {
	return logFile
}
```

---

# ⚙️ 4. UPDATE RUNNER (HIDE LOG + WRITE FILE)

## File:
`internal/preset/runner.go`

### Update execution:

```go
logWriter := logger.Writer()

cmd.Stdout = logWriter
cmd.Stderr = logWriter

cmd.Env = append(os.Environ(),
	"CI=true",
	"NPM_CONFIG_LOGLEVEL=silent",
	"NO_COLOR=1",
)
```

---

## Add step logging:

```go
logWriter.WriteString("\n=== Running: " + step.Run + " ===\n")
```

---

# 🔁 5. SPINNER INTEGRATION

Inside runner loop:

```go
spinner := ui.NewSpinner()
spinner.Start("Running: " + step.Run)

err := cmd.Run()

if err != nil {
	spinner.Stop("Failed: " + step.Run)
	return fmt.Errorf("step failed: %s (see logs)", step.Run)
}

spinner.Stop("Done: " + step.Run)
```

---

# 🎨 6. COLOR USAGE

Update CLI output:

```go
fmt.Println(ui.Success("✔ Installation complete"))
fmt.Println(ui.Error("❌ Installation failed"))
fmt.Println(ui.Warning("⚠ Warning"))
fmt.Println(ui.Info("ℹ Info"))
```

---

# 🧠 7. INSTALLER INTEGRATION

In `installer.go`:

- Initialize logger at start:
```go
if err := logger.Init(); err != nil {
	return err
}
```

- Ensure log file closed:
```go
defer logger.Writer().Close()
```

---

# 🧪 8. EXPECTED CLI OUTPUT

```
✔ Checking requirements
✔ Downloading package
✔ Extracting files
✔ Running: composer install
✔ Running: npm install
✔ Installation complete
```

---

# 📄 LOG FILE OUTPUT

```
=== Running: npm install ===
added 1200 packages...

=== Running: php artisan migrate ===
Migrating...
```

---

# ⚠️ RULES

- NEVER print external logs to terminal
- ALWAYS write logs to file
- Spinner must not conflict with output
- Always stop spinner before printing new line

---

# 🚫 OUT OF SCOPE

- log rotation
- verbose mode
- structured logging

---

# ✅ DONE CRITERIA

- Spinner works smoothly
- External logs hidden completely
- Logs written to file
- Colored output visible
- No broken CLI formatting

---

# 🧠 FINAL NOTE

This enhancement upgrades UX from:
- basic CLI tool → professional installer experience

Do not overcomplicate.
Keep it clean, deterministic, and readable.
