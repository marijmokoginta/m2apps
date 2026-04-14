# M2Apps — Phase 3 Implementation Guide (Requirement Check System)

## Context

Previous phases:
- Phase 1: CLI foundation (Cobra) ✔
- Phase 2: install.json config engine ✔

Ref:
- Phase 1 Guide → fileciteturn1file0
- Phase 1 Summary → fileciteturn1file1
- Phase 2 Summary → fileciteturn1file2

Phase 3 introduces runtime environment validation.

---

## Objective

Implement requirement checking system that:
- Reads requirements from config
- Executes system checks
- Validates version constraints
- Outputs CLI results
- Stops installation if failed

---

## Scope

### Included
- Requirement registry
- Multiple checkers:
  - php
  - node
  - mysql
  - flutter
  - dart
- Version parsing
- Version comparison
- CLI output formatting

### Excluded
- Auto install dependencies
- OS-specific install scripts
- Network/download logic

---

## Folder Structure

Create:

internal/requirements/

Files:

- types.go
- result.go
- checker.go
- registry.go
- runner.go
- version.go
- checkers/
  - php.go
  - node.go
  - mysql.go
  - flutter.go
  - dart.go

---

## Core Types

### types.go

type Requirement struct {
    Type    string
    Version string
}

---

### result.go

type Result struct {
    Name     string
    Required string
    Found    string
    Success  bool
    Message  string
}

---

### checker.go

type Checker interface {
    Check(versionConstraint string) (Result, error)
}

---

## Registry

### registry.go

var registry = map[string]Checker{
    "php":     PHPChecker{},
    "node":    NodeChecker{},
    "mysql":   MySQLChecker{},
    "flutter": FlutterChecker{},
    "dart":    DartChecker{},
}

---

## Runner

### runner.go

func Run(reqs []Requirement) []Result {
    var results []Result

    for _, r := range reqs {
        checker, ok := registry[r.Type]
        if !ok {
            results = append(results, Result{
                Name:    r.Type,
                Success: false,
                Message: "unknown requirement type",
            })
            continue
        }

        res, err := checker.Check(r.Version)
        if err != nil {
            res.Success = false
            res.Message = err.Error()
        }

        results = append(results, res)
    }

    return results
}

---

## Version System

### version.go

type Version struct {
    Major int
    Minor int
    Patch int
}

Implement:
- ParseVersion(string)
- Compare(v1, v2)

Support operator:
- >=

---

## Checkers Implementation

### PHP

Command:
php -v

---

### Node

Command:
node -v

---

### MySQL

Command:
mysql --version

---

### Flutter

Command:
flutter --version

---

### Dart

Command:
dart --version

---

## Parsing Rules

- Remove prefix:
  - v (node)
- Extract first version match
- Normalize to x.y.z

---

## CLI Output Format

Success:

Checking requirements...

[✓] PHP >= 8.1 (found 8.2.3)
[✓] Node >= 18 (found 18.17.0)

---

Failure:

Checking requirements...

[✓] PHP >= 8.1 (found 8.2.3)
[✗] Node >= 18 (not found)

Installation aborted.

---

## Integration (cmd/install.go)

After config loaded:

results := requirements.Run(config.Requirements)

Print results

If any result.Success == false:
    print "Installation aborted."
    exit(1)

---

## Edge Cases

- Command not found
- Invalid version output
- Unknown requirement type
- Partial version (8.1)

---

## Done Criteria

- Requirements executed from install.json
- CLI output readable
- Failure stops installation
- Supports multiple tools (php, node, mysql, flutter, dart)
- Build passes: go build ./...

---

## Notes

Keep implementation simple.
Do not over-engineer.
Focus on correctness and clarity.
