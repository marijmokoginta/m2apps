package ui

import (
	"errors"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var ErrMenuCancelled = errors.New("menu cancelled")

type MenuItem struct {
	Title  string
	Action string
}

type menuModel struct {
	title         string
	items         []MenuItem
	staticItems   []string
	selectedIndex int
	cancelled     bool
}

func RunMenu(title string, items []MenuItem, staticItems []string) (string, error) {
	if len(items) == 0 {
		return "", fmt.Errorf("menu items are empty")
	}

	if !isTerminal(os.Stdin) || !isTerminal(os.Stdout) {
		return "", fmt.Errorf("interactive menu requires terminal")
	}

	model := menuModel{
		title:       strings.TrimSpace(title),
		items:       items,
		staticItems: staticItems,
	}

	finalState, err := tea.NewProgram(model).Run()
	if err != nil {
		return "", err
	}

	state, ok := finalState.(menuModel)
	if !ok {
		return "", fmt.Errorf("invalid menu state")
	}
	if state.cancelled {
		return "", ErrMenuCancelled
	}
	if state.selectedIndex < 0 || state.selectedIndex >= len(state.items) {
		return "", fmt.Errorf("invalid selected menu item")
	}

	return state.items[state.selectedIndex].Action, nil
}

func (m menuModel) Init() tea.Cmd {
	return nil
}

func (m menuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m menuModel) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Underline(true)
	hintStyle := lipgloss.NewStyle().Faint(true)

	var buf strings.Builder
	if m.title != "" {
		buf.WriteString(titleStyle.Render(m.title))
		buf.WriteString("\n\n")
	}

	for i, item := range m.items {
		line := "  " + item.Title
		if i == m.selectedIndex {
			line = Info("> " + item.Title)
		}
		buf.WriteString(line)
		buf.WriteString("\n")
	}

	if len(m.staticItems) > 0 {
		buf.WriteString("\n")
		buf.WriteString(titleStyle.Render("Static Commands"))
		buf.WriteString("\n")
		for _, item := range m.staticItems {
			buf.WriteString("  - ")
			buf.WriteString(item)
			buf.WriteString("\n")
		}
	}

	buf.WriteString("\n")
	buf.WriteString(hintStyle.Render("Use ↑/↓ to navigate, Enter to select, Esc to exit."))
	buf.WriteString("\n")
	return buf.String()
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
