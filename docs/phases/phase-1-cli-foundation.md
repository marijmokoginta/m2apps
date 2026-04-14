# M2Apps — Phase 1 Implementation Guide (CLI Foundation)

Build a Go CLI application using Cobra in the CURRENT directory.

Context:

* The current working directory is already the project root named "m2apps"
* Do NOT create a new folder
* Do NOT nest another "m2apps" directory inside

Steps:

1. Initialize Go module in the current directory:
   go mod init m2apps

2. Initialize Cobra CLI in the current directory.

3. Create CLI structure:

   * cmd/root.go
   * cmd/install.go
   * cmd/update.go
   * cmd/list.go

4. Implement commands:

   * m2apps (root)
   * m2apps install
   * m2apps update
   * m2apps list

5. Each command should print:

   * install → "Starting installation..."
   * update → "Updating application..."
   * list → "Listing installed applications..."

6. Ensure:

   * No business logic
   * Only CLI wiring
   * Proper separation of structure

7. Running `m2apps` without arguments should show help.

Important:

* Do NOT create nested folders
* Use the current directory only
* Do NOT implement install/update logic yet

## Project Structure

```
m2apps/
├── cmd/
│   ├── root.go
│   ├── install.go
│   ├── update.go
│   ├── list.go
│
├── internal/
│   └── (empty for now)
│
├── main.go
├── go.mod
```

---

## Commands

### Root
- m2apps

### Subcommands
- m2apps install
- m2apps update
- m2apps list

---

## Command Behavior

### install
Print:
```
Starting installation...
```

### update
Print:
```
Updating application...
```

### list
Print:
```
Listing installed applications...
```

---

## Example Implementation

### install.go
```go
var installCmd = &cobra.Command{
    Use:   "install",
    Short: "Install application from install.json",
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Println("Starting installation...")
    },
}
```

---

## Root Wiring

```go
rootCmd.AddCommand(installCmd)
rootCmd.AddCommand(updateCmd)
rootCmd.AddCommand(listCmd)
```

---

## Entry Point

### main.go
```go
func main() {
    cmd.Execute()
}
```

---

## Expected Output

```bash
m2apps
→ show help

m2apps install
→ Starting installation...

m2apps update
→ Updating application...

m2apps list
→ Listing installed applications...
```

---

## Rules

- Keep CLI and logic separated
- No business logic
- Clear output messages
- Follow project structure

---

## Done Criteria

- CLI runs successfully
- All commands available
- Help command works
- No runtime errors

---

## Notes

This phase is the foundation.
Do not over-engineer.
Focus on clean structure.
