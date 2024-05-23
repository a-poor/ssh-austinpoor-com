package main

import (
	"github.com/a-poor/ssh-austinpoor-com/pkg/app"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	m, err := app.NewMDViewer()
	if err != nil {
		panic(err)
	}
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		panic(err)
	}
}
