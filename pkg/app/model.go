package app

import tea "github.com/charmbracelet/bubbletea"

type Model struct{}

func NewModel() tea.Model {
	return &Model{}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m Model) View() string {
	return "Hello, World!"
}
