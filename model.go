package main

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/a-poor/ssh-austinpoor-com/internal/sitecontent"
	"github.com/charmbracelet/x/ansi"
)

type section int

const (
	home section = iota
	about
	blog
	project
	sectionCount
)

var sectionNames = [...]string{"Home", "About", "Blog", "This project"}

type postItem struct{ post sitecontent.Post }

type postFilterMsg struct {
	query   string
	matches list.FilterMatchesMsg
}

// Tag asynchronous list results so fast typing cannot apply an older query.
func filterCommand(cmd tea.Cmd, query string) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg {
		switch msg := cmd().(type) {
		case list.FilterMatchesMsg:
			return postFilterMsg{query, msg}
		case tea.BatchMsg:
			commands := make(tea.BatchMsg, len(msg))
			for i, child := range msg {
				commands[i] = filterCommand(child, query)
			}
			return commands
		default:
			return msg
		}
	}
}

// Match each search word literally, preserving the catalog's newest-first order.
func filterPosts(query string, targets []string) []list.Rank {
	words := strings.Fields(strings.ToLower(query))
	var matches []list.Rank
	for i, target := range targets {
		target = strings.ToLower(target)
		found := true
		for _, word := range words {
			if !strings.Contains(target, word) {
				found = false
				break
			}
		}
		if found {
			matches = append(matches, list.Rank{Index: i})
		}
	}
	return matches
}

func (i postItem) Title() string { return i.post.Title }
func (i postItem) Description() string {
	date, _ := time.Parse(time.RFC3339, i.post.PublishedAt)
	return date.Format(time.DateOnly) + " · " + strings.Join(i.post.Tags, ", ") + " — " + i.post.Summary
}
func (i postItem) FilterValue() string {
	return i.post.Title + " " + i.post.Summary + " " + strings.Join(i.post.Tags, " ")
}

type model struct {
	w, h    int
	section section
	pages   map[section]string
	posts   list.Model
	article *sitecontent.Post
	vp      viewport.Model
	err     error
}

func makeModel(posts []sitecontent.Post) (model, error) {
	pages := make(map[section]string)
	for section, name := range map[section]string{home: "home", about: "about", project: "project"} {
		data, err := contentfs.ReadFile("content/pages/" + name + ".md")
		if err != nil {
			return model{}, err
		}
		pages[section] = string(data)
	}
	items := make([]list.Item, len(posts))
	for i, post := range posts {
		items[i] = postItem{post}
	}
	postList := list.New(items, list.NewDefaultDelegate(), 80, 19)
	postList.Title = "Blog"
	postList.Filter = filterPosts
	postList.SetStatusBarItemName("post", "posts")
	postList.SetShowHelp(false)
	postList.DisableQuitKeybindings()
	postList.KeyMap.ShowFullHelp.SetEnabled(false)
	postList.KeyMap.CloseFullHelp.SetEnabled(false)
	m := model{w: 80, h: 24, pages: pages, posts: postList, vp: viewport.New()}
	m.resize(80, 24)
	return m, m.err
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case postFilterMsg:
		if msg.query != m.posts.FilterValue() || m.posts.FilterState() == list.Unfiltered {
			return m, nil
		}
		var cmd tea.Cmd
		m.posts, cmd = m.posts.Update(msg.matches)
		return m, cmd
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
		return m, nil
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		// The list owns editing keys, including q and numbers, during search.
		if m.section == blog && m.article == nil && m.posts.FilterState() == list.Filtering {
			if msg.String() == "enter" {
				// Applying a query must not depend on a pending filter command.
				if strings.TrimSpace(m.posts.FilterValue()) == "" {
					m.posts.ResetFilter()
				} else {
					m.posts.SetFilterText(m.posts.FilterValue())
				}
				m.posts.FilterInput.Blur()
				return m, nil
			}
			var cmd tea.Cmd
			m.posts, cmd = m.posts.Update(msg)
			return m, filterCommand(cmd, m.posts.FilterValue())
		}
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "tab":
			m.navigate((m.section + 1) % sectionCount)
			return m, nil
		case "shift+tab":
			m.navigate((m.section + sectionCount - 1) % sectionCount)
			return m, nil
		case "1", "2", "3", "4":
			m.navigate(section(msg.String()[0] - '1'))
			return m, nil
		case "esc":
			if m.article != nil {
				m.article = nil
				m.err = nil
				return m, nil
			}
		case "enter":
			if m.section == blog && m.article == nil {
				if item, ok := m.posts.SelectedItem().(postItem); ok {
					m.article = &item.post
					m.renderDocument()
				}
				return m, nil
			}
		}
	}
	var cmd tea.Cmd
	if m.section == blog && m.article == nil {
		m.posts, cmd = m.posts.Update(msg)
	} else {
		m.vp, cmd = m.vp.Update(msg)
	}
	return m, filterCommand(cmd, m.posts.FilterValue())
}

func (m *model) navigate(next section) {
	m.section = next
	m.article = nil
	m.err = nil
	if next != blog {
		m.renderDocument()
	}
}

func (m *model) resize(w, h int) {
	position := 0.0
	if m.vp.YOffset() > 0 {
		position = m.vp.ScrollPercent()
	}
	m.w, m.h = max(1, w), max(1, h)
	m.posts.SetSize(max(24, m.w), max(5, m.h-5))
	m.vp.SetWidth(m.w)
	m.vp.SetHeight(max(1, m.h-5))
	if m.section != blog || m.article != nil {
		m.renderDocument()
		m.vp.SetYOffset(int(position * float64(max(0, m.vp.TotalLineCount()-m.vp.Height()))))
	}
}

func (m *model) renderDocument() {
	markdown := m.pages[m.section]
	baseURL := "https://austinpoor.com/"
	if m.article != nil {
		post := m.article
		published, _ := time.Parse(time.RFC3339, post.PublishedAt)
		markdown = fmt.Sprintf("# %s\n\n%s\n\n%s · %s\n\n[Read on the website](%s)\n\n---\n\n%s",
			post.Title, post.Summary, published.Format("January 2, 2006"), strings.Join(post.Tags, ", "), post.URL(), post.Markdown)
		baseURL = post.URL()
	}
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(max(1, m.w-4)),
		glamour.WithBaseURL(baseURL),
	)
	if err != nil {
		m.err = err
		return
	}
	rendered, err := renderer.Render(markdown)
	if err != nil {
		m.err = err
		return
	}
	m.err = nil
	m.vp.SetContent(rendered)
	m.vp.GotoTop()
}

func (m model) View() tea.View {
	if m.w < 24 || m.h < 10 {
		view := tea.NewView(ansi.Truncate("Please enlarge the terminal (24×10 minimum).", m.w, ""))
		view.AltScreen = true
		return view
	}
	var tabs []string
	for i, name := range sectionNames {
		if m.w < 40 {
			name = []string{"H", "A", "B", "P"}[i]
		} else if m.w < 55 {
			name = []string{"Home", "Me", "Blog", "Project"}[i]
		}
		label := fmt.Sprintf("%d %s", i+1, name)
		if section(i) == m.section {
			label = activeTabStyle.Render(label)
		} else {
			label = mutedStyle.Render(label)
		}
		tabs = append(tabs, label)
	}
	line := mutedStyle.Render(strings.Repeat("─", m.w))
	header := brandStyle.Render("austin / ssh") + "\n" + ansi.Truncate(strings.Join(tabs, "   "), m.w, "") + "\n" + line
	body := m.vp.View()
	help := "tab / shift+tab sections · ↑/↓ scroll · q quit"
	if m.section == blog && m.article == nil {
		body = m.posts.View()
		help = "↑/↓ select · / search · enter open · tab sections · q quit"
		if m.posts.FilterState() == list.Filtering {
			help = "Search titles, summaries, tags · enter apply · esc cancel"
		} else if m.posts.IsFiltered() {
			help = "↑/↓ select · enter open · esc clear · / edit · tab sections · q quit"
		}
		if len(m.posts.Items()) == 0 {
			body = "\n  No posts are bundled in this build yet."
		}
	} else if m.article != nil {
		help = "↑/↓ scroll · pgup/pgdown page · esc back · tab sections · q quit"
	}
	if m.err != nil {
		body = "Unable to render this page: " + m.err.Error()
	}
	body = lipgloss.NewStyle().Width(m.w).Height(m.h - 5).MaxWidth(m.w).MaxHeight(m.h - 5).Render(body)
	footer := line + "\n" + mutedStyle.Render(ansi.Truncate(help, m.w, "…"))
	view := tea.NewView(header + "\n" + body + "\n" + footer)
	view.AltScreen = true
	return view
}
