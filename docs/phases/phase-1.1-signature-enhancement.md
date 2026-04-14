# M2Apps — Phase 1 Enhancement: CLI Signature (ASCII Banner)

## Objective
Menambahkan signature CLI berupa ASCII banner pada root command (`m2apps`) untuk meningkatkan branding dan user experience.

---

## Scope
HANYA implement:
- ASCII banner pada root command
- Banner tampil saat menjalankan `m2apps` tanpa subcommand

JANGAN implement:
- Banner di subcommand (install, update, list)
- Warna / styling tambahan
- Logic lain di luar CLI

---

## Requirements

### 1. Banner Content

Gunakan format berikut:

"M2Code Apps" -> ASCII STYLE WITH RED-BLUE COLORS
"Auto Updater Engine" -> small description text
"by Marij Mokoginta" -> signature label

---

### 2. Implementation Rules

- Buat fungsi khusus:
```go
func printBanner()
```

- Gunakan `fmt.Println` untuk output

- Letakkan di file:
```
cmd/root.go
```

---

### 3. Root Command Behavior

Update root command:

- Saat menjalankan:
```
m2apps
```

Harus:
1. Menampilkan banner
2. Menampilkan help command

---

### 4. Implementation Example

```go
Run: func(cmd *cobra.Command, args []string) {
    printBanner()
    cmd.Help()
},
```

---

### 5. Constraints

- Banner hanya muncul di root command
- Tidak muncul saat:
  - m2apps install
  - m2apps update
  - m2apps list

- Tidak menambahkan dependency baru

---

## Expected Output

### Command:
```
m2apps
```

### Output:
- ASCII banner tampil
- Diikuti help dari Cobra

---

## Done Criteria

- Banner tampil dengan benar
- Tidak mengganggu subcommands
- CLI tetap berjalan normal

---

## Notes

- Fokus pada clean implementation
- Jangan over-engineer
- Jangan ubah struktur project
