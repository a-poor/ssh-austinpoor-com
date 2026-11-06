package sitecontent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testEntry(slug, date string) seedPost {
	var entry seedPost
	entry.Slug = slug
	entry.Path = "content/blog-posts/" + slug + ".md"
	entry.Frontmatter.Title = "Title for " + slug
	entry.Frontmatter.PublishDate = date
	entry.Frontmatter.Tags = []string{"go"}
	return entry
}

func writeSource(t *testing.T, entries []seedPost, bodies map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "content/blog-posts"), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(entries)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "content/seed.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	for slug, body := range bodies {
		if err := os.WriteFile(filepath.Join(dir, "content/blog-posts", slug+".md"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestImportPublishedSnapshot(t *testing.T) {
	older := testEntry("older", "2024-01-01")
	older.Frontmatter.Description = "Fallback summary"
	newer := testEntry("newer", "2024-06-01T12:30:00-07:00")
	newer.Frontmatter.Subtitle = "Current summary"
	draft := testEntry("draft", "2024-01-01")
	draft.Frontmatter.IsDraft = true
	entries := []seedPost{older, draft, testEntry("scheduled", "2099-01-01"), testEntry("undated", ""), testEntry("_hidden", "2024-01-01"), newer}
	// Excluded entries intentionally have no files: their private bodies must
	// never be read or required for a published content build.
	dir := writeSource(t, entries, map[string]string{"older": "Old body", "newer": "Latest body from disk"})
	result, err := Import(dir, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Posts) != 2 || result.Posts[0].Slug != "newer" || result.Posts[1].Slug != "older" {
		t.Fatalf("expected only published posts, newest first: %+v", result.Posts)
	}
	post := result.Posts[0]
	if post.Markdown != "Latest body from disk" || post.Summary != "Current summary" || post.PublishedAt != "2024-06-01T19:30:00Z" {
		t.Fatalf("wrong published content: %+v", post)
	}
	if result.Posts[1].Summary != "Fallback summary" || post.URL() != "https://austinpoor.com/blog/newer" {
		t.Fatal("missing summary or canonical URL")
	}
}

func TestImportRejectsInvalidPublishedEntries(t *testing.T) {
	tests := []struct {
		name string
		edit func(*seedPost)
		body string
	}{
		{"bad date", func(p *seedPost) { p.Frontmatter.PublishDate = "yesterday" }, "Body"},
		{"missing title", func(p *seedPost) { p.Frontmatter.Title = "" }, "Body"},
		{"path escape", func(p *seedPost) { p.Path = "../private.md" }, "Body"},
		{"wrong file", func(p *seedPost) { p.Path = "content/blog-posts/another.md" }, "Body"},
		{"invalid slug", func(p *seedPost) { p.Slug = "../../private" }, "Body"},
		{"empty body", func(p *seedPost) {}, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			entry := testEntry("post", "2024-01-01")
			tc.edit(&entry)
			dir := writeSource(t, []seedPost{entry}, map[string]string{"post": tc.body})
			if _, err := Import(dir, time.Now()); err == nil {
				t.Fatal("expected import to fail")
			}
		})
	}
}

func TestImportRejectsDuplicateAndMissingPosts(t *testing.T) {
	entry := testEntry("post", "2024-01-01")
	dir := writeSource(t, []seedPost{entry, entry}, map[string]string{"post": "Body"})
	if _, err := Import(dir, time.Now()); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate error, got %v", err)
	}
	dir = writeSource(t, []seedPost{entry}, nil)
	if _, err := Import(dir, time.Now()); err == nil {
		t.Fatal("missing published Markdown must fail the import")
	}
}

func TestImportCannotFollowSymlinkOutsideWebsite(t *testing.T) {
	entry := testEntry("post", "2024-01-01")
	dir := writeSource(t, []seedPost{entry}, nil)
	outside := filepath.Join(t.TempDir(), "private.md")
	if err := os.WriteFile(outside, []byte("Private"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, entry.Path)); err != nil {
		t.Fatal(err)
	}
	if _, err := Import(dir, time.Now()); err == nil {
		t.Fatal("external symlink should not be imported")
	}
}
