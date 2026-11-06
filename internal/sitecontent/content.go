// Package sitecontent prepares a published snapshot of the website for the TUI.
package sitecontent

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Post struct {
	Slug        string   `json:"slug"`
	Title       string   `json:"title"`
	Summary     string   `json:"summary"`
	Tags        []string `json:"tags"`
	PublishedAt string   `json:"publishedAt"`
	Markdown    string   `json:"markdown"`
}

func (p Post) URL() string { return "https://austinpoor.com/blog/" + p.Slug }

type Catalog struct {
	Posts []Post `json:"posts"`
}

type seedPost struct {
	Path        string `json:"path"`
	Slug        string `json:"slug"`
	Frontmatter struct {
		IsDraft     bool     `json:"isDraft"`
		Title       string   `json:"title"`
		Subtitle    string   `json:"subtitle"`
		Description string   `json:"description"`
		PublishDate string   `json:"publishDate"`
		Tags        []string `json:"tags"`
	} `json:"frontmatter"`
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// Import reads metadata from content/seed.json and current bodies from the
// corresponding Markdown files, rather than the seed's duplicate body text.
// Drafts and scheduled posts are excluded before their bodies are read.
func Import(directory string, now time.Time) (Catalog, error) {
	root, err := os.OpenRoot(directory)
	if err != nil {
		return Catalog{}, err
	}
	defer root.Close()
	data, err := root.ReadFile("content/seed.json")
	if err != nil {
		return Catalog{}, fmt.Errorf("read website metadata: %w", err)
	}
	var entries []seedPost
	if err := json.Unmarshal(data, &entries); err != nil {
		return Catalog{}, fmt.Errorf("decode website metadata: %w", err)
	}
	catalog := Catalog{Posts: []Post{}}
	seen := make(map[string]bool)
	for _, entry := range entries {
		meta := entry.Frontmatter
		if meta.IsDraft || meta.PublishDate == "" || strings.HasPrefix(path.Base(entry.Path), "_") || strings.HasPrefix(entry.Slug, "_") {
			continue
		}
		published, err := publicationDate(meta.PublishDate)
		if err != nil {
			return Catalog{}, fmt.Errorf("post %q: %w", entry.Slug, err)
		}
		if published.After(now) {
			continue
		}
		if !slugPattern.MatchString(entry.Slug) || entry.Path != "content/blog-posts/"+entry.Slug+".md" {
			return Catalog{}, fmt.Errorf("post %q: expected a matching file in content/blog-posts", entry.Slug)
		}
		if seen[entry.Slug] {
			return Catalog{}, fmt.Errorf("duplicate post slug %q", entry.Slug)
		}
		seen[entry.Slug] = true
		if strings.TrimSpace(meta.Title) == "" {
			return Catalog{}, fmt.Errorf("post %q: missing title", entry.Slug)
		}
		body, err := root.ReadFile(entry.Path)
		if err != nil {
			return Catalog{}, fmt.Errorf("read post %q: %w", entry.Slug, err)
		}
		if strings.TrimSpace(string(body)) == "" {
			return Catalog{}, fmt.Errorf("post %q: empty Markdown", entry.Slug)
		}
		summary := meta.Subtitle
		if summary == "" {
			summary = meta.Description
		}
		catalog.Posts = append(catalog.Posts, Post{
			Slug: entry.Slug, Title: meta.Title, Summary: summary,
			Tags: meta.Tags, PublishedAt: published.UTC().Format(time.RFC3339),
			Markdown: string(body),
		})
	}
	sort.Slice(catalog.Posts, func(i, j int) bool {
		if catalog.Posts[i].PublishedAt == catalog.Posts[j].PublishedAt {
			return catalog.Posts[i].Slug < catalog.Posts[j].Slug
		}
		return catalog.Posts[i].PublishedAt > catalog.Posts[j].PublishedAt
	})
	return catalog, nil
}

func publicationDate(value string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339, time.DateOnly} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid publication date %q", value)
}
