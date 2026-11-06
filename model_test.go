package main

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/a-poor/ssh-austinpoor-com/internal/sitecontent"
)

func testModel(t *testing.T) model {
	t.Helper()
	m, err := makeModel([]sitecontent.Post{
		{Slug: "first", Title: "First article", Summary: "Terminal experiments", Tags: []string{"golang"}, PublishedAt: "2024-06-01T00:00:00Z", Markdown: strings.Repeat("A paragraph about building things.\n\n", 60)},
		{Slug: "second", Title: "Second article", Summary: "Data experiments", Tags: []string{"python"}, PublishedAt: "2024-01-01T00:00:00Z", Markdown: "A short post."},
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func press(m model, code rune, text string) (model, tea.Cmd) {
	next, cmd := m.Update(tea.KeyPressMsg{Code: code, Text: text})
	return next.(model), cmd
}

func TestBrowseFilterReadAndBack(t *testing.T) {
	m := testModel(t)
	m, _ = press(m, '3', "3")
	// Search must cover tags and summaries as well as titles.
	for _, query := range []string{"golang", "Terminal"} {
		m.posts.SetFilterText(query)
		if len(m.posts.VisibleItems()) != 1 || m.posts.SelectedItem().(postItem).post.Slug != "first" {
			t.Fatalf("query %q did not find the expected post", query)
		}
	}
	m, _ = press(m, tea.KeyEnter, "")
	if m.article == nil || m.article.Slug != "first" {
		t.Fatal("enter did not open the filtered article")
	}
	m, _ = press(m, tea.KeyPgDown, "")
	if m.vp.YOffset() == 0 {
		t.Fatal("article did not scroll")
	}
	m, _ = press(m, tea.KeyEscape, "")
	if m.article != nil || !m.posts.IsFiltered() || m.posts.SelectedItem().(postItem).post.Slug != "first" {
		t.Fatal("back did not preserve the filtered list")
	}
	m, _ = press(m, tea.KeyEscape, "")
	if len(m.posts.VisibleItems()) != 2 {
		t.Fatal("escape did not clear the filter")
	}
	m.posts.SetFilterText("zzzzzzzzzz")
	m, _ = press(m, tea.KeyEnter, "")
	if m.article != nil {
		t.Fatal("empty results must not open an article")
	}
}

func TestSearchOwnsQuitAndNavigationKeys(t *testing.T) {
	m := testModel(t)
	m, _ = press(m, '3', "3")
	m, _ = press(m, '/', "/")
	if m.posts.FilterState() != list.Filtering {
		t.Fatal("slash did not focus the search")
	}
	m, _ = press(m, 'q', "q")
	m, _ = press(m, '1', "1")
	if m.section != blog || m.posts.FilterValue() != "q1" {
		t.Fatal("search intercepted by global shortcuts")
	}
	m, _ = press(m, tea.KeyEscape, "")
	_, cmd := press(m, 'q', "q")
	if cmd == nil {
		t.Fatal("q should quit outside search")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("q did not issue quit")
	}
}

func TestSearchIgnoresStaleResults(t *testing.T) {
	m := testModel(t)
	m.navigate(blog)
	m.posts.SetFilterText("python")
	next, _ := m.Update(postFilterMsg{query: "old query", matches: nil})
	m = next.(model)
	if len(m.posts.VisibleItems()) != 1 || m.posts.SelectedItem().(postItem).post.Slug != "second" {
		t.Fatal("an older query replaced the current search results")
	}
}

func TestSearchCanBeAppliedBeforeAsyncResults(t *testing.T) {
	m := testModel(t)
	m, _ = press(m, '3', "3")
	m, _ = press(m, '/', "/")
	for _, char := range "python" {
		// Intentionally leave filtering commands pending, as with fast typing.
		m, _ = press(m, char, string(char))
	}
	m, _ = press(m, tea.KeyEnter, "")
	m, _ = press(m, tea.KeyEnter, "")
	if m.article == nil || m.article.Slug != "second" {
		t.Fatal("enter opened a result from before the query was applied")
	}
}

func TestSearchMatchesWordsAcrossMetadata(t *testing.T) {
	matches := filterPosts("DENO rust", []string{
		"The Question is the Answer documentation knowledge-base",
		"Running JavaScript in Rust with Deno rust javascript deno",
	})
	if len(matches) != 1 || matches[0].Index != 1 {
		t.Fatal("search should match words, ignoring case, across the post metadata")
	}
}

func TestNavigationAndResponsiveLayout(t *testing.T) {
	m := testModel(t)
	if m.vp.YOffset() != 0 || !strings.Contains(m.vp.View(), "Austin") {
		t.Fatal("home should start at its heading")
	}
	for _, dimensions := range [][2]int{{120, 36}, {80, 24}, {40, 15}, {24, 10}, {10, 4}} {
		for section := home; section < sectionCount; section++ {
			m.navigate(section)
			m.resize(dimensions[0], dimensions[1])
			if m.err != nil {
				t.Fatal(m.err)
			}
			if m.vp.YOffset() != 0 {
				t.Fatal("resize should keep the top of a newly opened page visible")
			}
			view := m.View().Content
			if lipgloss.Width(view) > dimensions[0] || lipgloss.Height(view) > dimensions[1] {
				t.Fatalf("section %d overflowed %dx%d: %dx%d", section, dimensions[0], dimensions[1], lipgloss.Width(view), lipgloss.Height(view))
			}
		}
	}
	m.navigate(project)
	m, _ = press(m, tea.KeyTab, "")
	if m.section != home {
		t.Fatal("tab should wrap to Home")
	}
}
