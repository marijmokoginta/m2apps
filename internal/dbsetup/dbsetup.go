package dbsetup

import (
	"bufio"
	"fmt"
	"m2apps/internal/ui"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DBConfig holds the database connection configuration collected from the user.
type DBConfig struct {
	Driver   string // mysql, postgres, sqlite, sqlserver
	Host     string
	Port     string
	DBName   string
	Username string
	Password string
}

// ToEnvMap converts a DBConfig into a map of standard env variable names.
func ToEnvMap(cfg DBConfig) map[string]string {
	return map[string]string{
		"DB_CONNECTION": cfg.Driver,
		"DB_HOST":       cfg.Host,
		"DB_PORT":       cfg.Port,
		"DB_DATABASE":   cfg.DBName,
		"DB_USERNAME":   cfg.Username,
		"DB_PASSWORD":   cfg.Password,
	}
}

// EnvKeys returns the list of env keys managed by dbsetup.
func EnvKeys() []string {
	return []string{
		"DB_CONNECTION",
		"DB_HOST",
		"DB_PORT",
		"DB_DATABASE",
		"DB_USERNAME",
		"DB_PASSWORD",
	}
}

// DBConfigFromEnvMap builds a DBConfig from a map of env values (e.g. from env.ReadValues).
func DBConfigFromEnvMap(m map[string]string) DBConfig {
	return DBConfig{
		Driver:   m["DB_CONNECTION"],
		Host:     m["DB_HOST"],
		Port:     m["DB_PORT"],
		DBName:   m["DB_DATABASE"],
		Username: m["DB_USERNAME"],
		Password: m["DB_PASSWORD"],
	}
}

// PromptDBConfig runs an interactive wizard to collect DB configuration.
// defaults is used to pre-fill each prompt; the user can press Enter to keep defaults.
func PromptDBConfig(defaults DBConfig) (DBConfig, error) {
	// Step 1: driver selection via bubbletea menu
	driver, err := runDriverMenu(defaults.Driver)
	if err != nil {
		return DBConfig{}, err
	}

	cfg := DBConfig{Driver: driver}

	// SQLite only needs a file path as DB name; skip host/port/user/pass
	if driver == "sqlite" {
		dbName, err := promptField("Database file path", coalesce(defaults.DBName, "database/database.sqlite"))
		if err != nil {
			return DBConfig{}, err
		}
		cfg.DBName = dbName
		return cfg, nil
	}

	// Step 2: remaining fields via plain prompts
	cfg.Host, err = promptField("Host", coalesce(defaults.Host, "127.0.0.1"))
	if err != nil {
		return DBConfig{}, err
	}

	cfg.Port, err = promptField("Port", coalesce(defaults.Port, defaultPortFor(driver)))
	if err != nil {
		return DBConfig{}, err
	}

	cfg.DBName, err = promptField("Database name", coalesce(defaults.DBName, ""))
	if err != nil {
		return DBConfig{}, err
	}

	cfg.Username, err = promptField("Username", coalesce(defaults.Username, "root"))
	if err != nil {
		return DBConfig{}, err
	}

	cfg.Password, err = promptField("Password", defaults.Password)
	if err != nil {
		return DBConfig{}, err
	}

	return cfg, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Driver selection — bubbletea model

type driverModel struct {
	items         []driverItem
	selectedIndex int
	cancelled     bool
}

type driverItem struct {
	label  string
	driver string
}

var drivers = []driverItem{
	{label: "MySQL", driver: "mysql"},
	{label: "PostgreSQL", driver: "pgsql"},
	{label: "SQLite", driver: "sqlite"},
	{label: "SQL Server", driver: "sqlsrv"},
}

func runDriverMenu(currentDriver string) (string, error) {
	if !isTerminal(os.Stdin) || !isTerminal(os.Stdout) {
		return coalesce(currentDriver, "mysql"), nil
	}

	initial := 0
	for i, d := range drivers {
		if strings.EqualFold(d.driver, strings.TrimSpace(currentDriver)) {
			initial = i
			break
		}
	}

	m := driverModel{items: drivers, selectedIndex: initial}
	finalState, err := tea.NewProgram(m).Run()
	if err != nil {
		return "", fmt.Errorf("driver menu error: %w", err)
	}
	state, ok := finalState.(driverModel)
	if !ok {
		return "", fmt.Errorf("invalid driver menu state")
	}
	if state.cancelled {
		return "", fmt.Errorf("database setup cancelled")
	}
	return state.items[state.selectedIndex].driver, nil
}

func (m driverModel) Init() tea.Cmd { return nil }

func (m driverModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.KeyMsg:
		switch typed.String() {
		case "esc", "ctrl+c", "q":
			m.cancelled = true
			return m, tea.Quit
		case "up", "k":
			if m.selectedIndex == 0 {
				m.selectedIndex = len(m.items) - 1
			} else {
				m.selectedIndex--
			}
		case "down", "j":
			m.selectedIndex++
			if m.selectedIndex >= len(m.items) {
				m.selectedIndex = 0
			}
		case "enter":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m driverModel) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Underline(true)
	hintStyle := lipgloss.NewStyle().Faint(true)

	var buf strings.Builder
	buf.WriteString(titleStyle.Render("Select Database Driver"))
	buf.WriteString("\n\n")

	for i, item := range m.items {
		line := "  " + item.label
		if i == m.selectedIndex {
			line = ui.Info("> " + item.label)
		}
		buf.WriteString(line)
		buf.WriteString("\n")
		if i < len(m.items)-1 {
			buf.WriteString("\n")
		}
	}

	buf.WriteString("\n")
	buf.WriteString(hintStyle.Render("Use ↑/↓ to navigate, Enter to select, Esc to cancel."))
	buf.WriteString("\n\n")
	return buf.String()
}

// ──────────────────────────────────────────────────────────────────────────────
// Plain text prompt helpers

func promptField(label, defaultValue string) (string, error) {
	reader := bufio.NewReader(os.Stdin)

	if strings.TrimSpace(defaultValue) != "" {
		fmt.Printf("  %s [%s]: ", ui.Info(label), defaultValue)
	} else {
		fmt.Printf("  %s: ", ui.Info(label))
	}

	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input for %s: %w", label, err)
	}

	value := strings.TrimSpace(strings.TrimRight(input, "\r\n"))
	if value == "" {
		return defaultValue, nil
	}
	return value, nil
}

func defaultPortFor(driver string) string {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "mysql":
		return "3306"
	case "pgsql", "postgres", "postgresql":
		return "5432"
	case "sqlsrv", "sqlserver":
		return "1433"
	default:
		return ""
	}
}

func coalesce(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
