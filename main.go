package main

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/a-poor/ssh-austinpoor-com/internal/sitecontent"
)

//go:embed content
var contentfs embed.FS

func main() {
	catalog, err := loadCatalog()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Load content:", err)
		os.Exit(1)
	}
	m, err := makeModel(catalog.Posts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Create viewer:", err)
		os.Exit(1)
	}
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Run viewer:", err)
		os.Exit(1)
	}
}

func loadCatalog() (sitecontent.Catalog, error) {
	data, err := contentfs.ReadFile("content/posts.json")
	if errors.Is(err, fs.ErrNotExist) {
		return sitecontent.Catalog{}, nil
	}
	if err != nil {
		return sitecontent.Catalog{}, err
	}
	var catalog sitecontent.Catalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return catalog, fmt.Errorf("decode embedded posts: %w", err)
	}
	return catalog, nil
}
