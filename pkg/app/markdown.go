package app

import (
	_ "embed"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

const (
	defaultWidth  = 80
	defaultHeight = 24
)

//go:embed index.md
var IndexPage string

//go:embed about.md
var AboutPage string

type MDViewer struct {
	Text   string
	Width  int
	Height int
	VP     *viewport.Model
	GTR    *glamour.TermRenderer
}

func NewMDViewer() (*MDViewer, error) {
	// Get the color scheme...
	scheme := "dracula"
	if !lipgloss.HasDarkBackground() {
		scheme = "light"
	}

	// Create the markdown renderer
	gtr, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(scheme),
		glamour.WithEmoji(),
		glamour.WithWordWrap(defaultWidth-1),
	)
	if err != nil {
		return nil, err
	}

	// Create the scrollable viewport
	vp := viewport.New(defaultWidth, defaultHeight)

	// Create the MDViewer
	mdv := &MDViewer{
		Text:   "",
		Width:  defaultWidth,
		Height: defaultHeight,
		VP:     &vp,
		GTR:    gtr,
	}

	// Initialize it to be the index page
	if err := mdv.SetContentIndex(); err != nil {
		return nil, err
	}

	// Return it!
	return mdv, nil
}

func (m *MDViewer) SetContent(content string) error {
	c, err := m.GTR.Render(content)
	if err != nil {
		return err
	}
	m.VP.SetContent(c)
	return nil
}

func (m *MDViewer) SetContentIndex() error {
	return m.SetContent(IndexPage)
}

func (m *MDViewer) SetContentAbout() error {
	return m.SetContent(AboutPage)
}

func (m *MDViewer) Init() tea.Cmd {
	return nil
}

func (m *MDViewer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.setSize(msg.Width, msg.Height)
	}

	vp, cmd := m.VP.Update(msg)
	m.VP = &vp
	return m, cmd
}

func (m *MDViewer) setSize(w, h int) {
	m.Width = w
	m.Height = h
	m.VP.Width = w
	m.VP.Height = h
}

func (m *MDViewer) View() string {
	vp := m.VP.View()
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		vp,
	)
}
