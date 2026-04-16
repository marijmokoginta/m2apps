# M2Apps — Phase 15: Requirement Install Engine

## Objective
Implement guided requirement installation for `m2apps install` with flow:

`Check -> Missing -> Plan -> Confirm -> Install -> Re-check`

This phase extends existing requirement check so installation can continue for non-technical users without silent system changes.

## Source of Truth
- `docs/architecture/m2apps-requirement-strategy.md`
- Existing requirement checker engine in `internal/requirements/*`

## Scope

### In Scope
- Add `install_mode` in `install.json`:
  - `assisted` (default)
  - `manual`
- Detect and classify missing requirement details (not found/version mismatch/unknown)
- Build OS-aware install plan per tool
- Ask explicit user confirmation before any install command
- Execute install per tool (best effort, per-OS)
- Re-check requirements after execution
- Provide manual fallback instructions when install fails or user declines
- Integrate full flow into `m2apps install`

### Out of Scope
- Silent/background dependency install without user confirmation
- Aggressive system reconfiguration (service setup, database initialization, etc.)
- Bundled stack installer (XAMPP-like)
- Container/runtime orchestration

## Functional Requirements

1. `install_mode` behavior:
   - If omitted, treat as `assisted`
   - `manual`: do check and print manual instructions, then abort when requirements are missing
   - `assisted`: do guided plan+confirm+install+re-check

2. Missing requirement classification:
   - `not_found`
   - `version_mismatch`
   - `unknown`

3. Plan generation:
   - Must include tool, required version, resolved target version, method, and command list
   - Must be OS-aware (Windows/Linux/macOS)

4. Confirmation policy:
   - Always ask user confirmation globally
   - Optionally allow per-tool confirmation (y/N)

5. Execution policy:
   - Re-check tool before attempting install candidate (idempotent)
   - Stop counting as success unless post-check passes
   - Continue to next candidate on failure, but show clear status

6. Fallback policy:
   - Always print manual instructions for unresolved tools
   - Keep command examples copy-paste ready

## Technical Design

### A. Config + Validation
- `internal/config/types.go`
  - Add `InstallMode string \`json:"install_mode"\``
- `internal/config/validate.go`
  - Allowed: empty, `assisted`, `manual`

### B. Requirement Result Enrichment
- `internal/requirements/result.go`
  - Add fields:
    - `Missing bool`
    - `Reason string`

- `internal/requirements/runner.go`
  - Fill `Missing` and `Reason` for unknown checker/error/version mismatch

### C. New Package: `internal/reqinstall`

#### 1) `plan`
- Build install plan from failed requirement results
- Core structs:
  - `MissingRequirement`
  - `InstallCandidate`
  - `InstallPlan`

#### 2) `resolver`
- Resolve required constraint to target install version
- Initial stable mapping:
  - php `>=8.1` -> `8.2`
  - node `>=18` -> `20`
  - mysql `>=8.0` -> `8.0`
  - flutter/dart -> `stable`

#### 3) `executor`
- Execute command candidates per OS/tool
- Linux: apt/dnf/yum fallback sequence
- macOS: brew
- Windows: winget/choco fallback sequence
- No forced elevation; if command fails due privilege/tool absence, report clearly

#### 4) `manual`
- Manual installation instructions per OS/tool
- Output used for:
  - manual mode
  - assisted mode fallback

#### 5) Orchestrator
- One entry point for install flow handling in `cmd/install.go`
- Responsibilities:
  - detect missing
  - render plan
  - request confirmation
  - execute candidates
  - re-check
  - render fallback instructions

### D. Integration in `cmd/install.go`

Replace direct abort-on-fail with:
1. Run requirement check
2. If all pass -> continue install app
3. If missing:
   - `manual` -> print instructions and abort
   - `assisted` -> plan + confirm + execute + re-check
4. If still missing -> print fallback instructions and abort
5. If passed -> continue existing install flow

## UX Output Contract

Required sections:
- `Checking requirements...`
- `Missing tools detected`
- `Install plan`
- `Execution result` (`[OK]`, `[FAIL]`, `[SKIP]`)
- `Manual fallback instructions` (if unresolved)

## Reliability Requirements
- Idempotent re-check before each candidate execution
- Re-check full requirement list after assisted execution
- Never continue app install when final requirement check still fails

## Validation Checklist
- Missing tool + assisted mode + user approve -> tries install then re-check
- Missing tool + assisted mode + user decline -> abort with manual instructions
- Missing tool + manual mode -> no auto install attempt, manual instructions only
- Version mismatch -> plan selects upgrade target version
- Unknown requirement type -> clear error and no crash

## Deliverables
- New doc: `docs/phases/phase-15-requirement-install.md`
- New package: `internal/reqinstall/*`
- Updated:
  - `internal/config/types.go`
  - `internal/config/validate.go`
  - `internal/requirements/result.go`
  - `internal/requirements/runner.go`
  - `cmd/install.go`

## Notes
- This phase is intentionally guided/semi-automatic, not fully autonomous.
- System safety and explicit user intent remain mandatory.
