package tui

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

var (
	titleStyle = lipgloss.NewStyle().
			MarginLeft(2).
			Bold(true)
	activeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))
	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))
)

type listItem struct {
	value    string
	label    string
	isActive bool
}

func (i listItem) Title() string       { return i.label }
func (i listItem) Description() string { return "" }
func (i listItem) FilterValue() string { return i.label }

type model struct {
	list     list.Model
	selected string
	quitting bool
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			item, ok := m.list.SelectedItem().(listItem)
			if ok {
				m.selected = item.value
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.quitting {
		return ""
	}
	return m.list.View()
}

type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	it, ok := item.(listItem)
	if !ok {
		return
	}

	cursor := " "
	if index == m.Index() {
		cursor = ">"
	}

	marker := " "
	if it.isActive {
		marker = "*"
	}

	str := fmt.Sprintf("%s %s %s", cursor, marker, it.label)
	if index == m.Index() {
		fmt.Fprint(w, activeStyle.Render(str))
	} else {
		fmt.Fprint(w, dimStyle.Render(str))
	}
}

// Pick shows an interactive full-screen list picker and returns the selected
// value. title is the heading shown above the list. items is a map of
// value→label; the entry whose value equals active is marked as active.
// Returns the selected value and nil on selection, or "" and nil if the user
// cancelled (pressed q/Esc).
func Pick(title string, items []string, active string) (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", fmt.Errorf("no project specified and stdin is not a terminal")
	}

	entries := make([]list.Item, 0, len(items))
	for _, v := range items {
		entries = append(entries, listItem{
			value:    v,
			label:    v,
			isActive: v == active,
		})
	}

	delegate := itemDelegate{}
	m := model{
		list: list.New(entries, delegate, 80, 20),
	}
	m.list.Title = title
	m.list.SetShowStatusBar(false)
	m.list.SetFilteringEnabled(false)
	m.list.SetShowHelp(false)
	m.list.Styles.Title = titleStyle

	// Default-select the active item so Enter immediately confirms the current
	// active one.
	for i, it := range entries {
		if l, ok := it.(listItem); ok && l.isActive {
			m.list.Select(i)
		}
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("running picker: %w", err)
	}

	res, ok := finalModel.(model)
	if !ok {
		return "", fmt.Errorf("unexpected model type")
	}
	if res.quitting {
		return "", nil
	}
	return res.selected, nil
}
