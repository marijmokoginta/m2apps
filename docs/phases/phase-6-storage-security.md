# M2Apps Phase 6 — Storage & Security (Codex Instruction)

## Objective
Implement secure storage system for application metadata using encryption.

---

## Scope
This phase includes:
- Encrypted storage (AES-GCM)
- App metadata persistence
- Token security
- Integration with installer (Phase 5)

---

## Directory Structure
~/.m2apps/
  apps/{app_id}/
    config.enc
    state.json
  logs/

---

## Data Model

type AppConfig struct {
    AppID       string
    Name        string
    InstallPath string
    Repo        string
    Asset       string
    Token       string
    Version     string
    Preset      string
}

---

## Storage Module

Create:
internal/storage/
  storage.go
  encrypt.go
  decrypt.go
  model.go

---

## Storage Interface

type Storage interface {
    Save(appID string, data AppConfig) error
    Load(appID string) (AppConfig, error)
}

---

## Encryption

Use AES-256 GCM

Steps:
1. Generate key from machine identity
2. Create cipher block
3. Use GCM
4. Generate random nonce
5. Encrypt data

---

## Key Derivation

key = SHA256(hostname + user + static_salt)

---

## Save Flow

1. Serialize struct to JSON
2. Encrypt JSON
3. Write to config.enc

---

## Load Flow

1. Read config.enc
2. Decrypt
3. Deserialize JSON

---

## Installer Integration

After successful install:

config := AppConfig{...}
storage.Save(appID, config)

---

## Token Handling

- Never store plaintext token
- Encrypt before saving
- Do not print token in CLI

---

## Error Handling

- Fail on encryption error
- Fail on write error
- Fail on decrypt error

---

## Cleanup

After install success:
- Remove install.json

---

## Validation Checklist

- Encrypted file not readable
- Decrypt returns correct data
- Token hidden
- Save/load works
