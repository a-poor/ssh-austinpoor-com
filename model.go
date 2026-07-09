package main

import (
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
)

type model struct {
	w, h int
	vp   viewport.Model
}

func makeModel() model {
	// Read the markdown file
	b, err := contentfs.ReadFile("content/js-in-rs.md")
	if err != nil {
		panic(err)
	}
	s, err := glamour.Render(string(b), "dark")
	if err != nil {
		panic(err)
	}

	// Get a viewport
	vp := viewport.New()

	// Set the size
	vp.SetWidth(120)
	vp.SetHeight(30)

	// Set the content
	vp.SetContent(s)

	// Done
	return model{
		w:  120,
		h:  30,
		vp: vp,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		default:
			vp, cmd := m.vp.Update(msg)
			return model{
				w:  m.w,
				h:  m.h,
				vp: vp,
			}, cmd
		}
	case tea.WindowSizeMsg:
		vp := m.vp
		vp.SetWidth(msg.Width)
		vp.SetHeight(msg.Height)
		return model{
			w:  msg.Width,
			h:  msg.Height,
			vp: vp,
		}, nil
	}
	return m, nil
}

func (m model) View() tea.View {
	// s := "Hello, World!"
	// return tea.NewView(s)
	v := tea.NewView(m.vp.View())
	v.AltScreen = true
	return v
}
