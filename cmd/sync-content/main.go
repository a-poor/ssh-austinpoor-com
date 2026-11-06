// sync-content creates the content snapshot that the next build embeds.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/a-poor/ssh-austinpoor-com/internal/sitecontent"
)

func main() {
	source := flag.String("source", "", "path to the austinpoor-dot-com website checkout (required)")
	output := flag.String("output", "content/posts.json", "generated content bundle")
	flag.Parse()
	if *source == "" || flag.NArg() != 0 {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(*source, *output); err != nil {
		fmt.Fprintln(os.Stderr, "sync content:", err)
		os.Exit(1)
	}
}

func run(source, output string) error {
	catalog, err := sitecontent.Import(source, time.Now())
	if err != nil {
		return err
	}
	if len(catalog.Posts) == 0 {
		return fmt.Errorf("no published posts found; existing snapshot was not replaced")
	}
	data, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if existing, err := os.ReadFile(output); err == nil && bytes.Equal(existing, data) {
		fmt.Printf("Content is current (%d published posts).\n", len(catalog.Posts))
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(output), ".posts-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), output); err != nil {
		return err
	}
	fmt.Printf("Imported %d published posts into %s.\n", len(catalog.Posts), output)
	return nil
}
