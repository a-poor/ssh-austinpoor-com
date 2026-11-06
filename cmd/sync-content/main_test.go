package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFailedImportPreservesSnapshot(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "content"), 0o755); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "posts.json")
	for _, input := range []string{"not json", "[]", `[{"slug":"missing","path":"content/blog-posts/missing.md","frontmatter":{"title":"Missing","publishDate":"2024-01-01"}}]`} {
		if err := os.WriteFile(output, []byte("previous snapshot"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "content/seed.json"), []byte(input), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := run(dir, output); err == nil {
			t.Fatalf("expected failure for %s", input)
		}
		data, err := os.ReadFile(output)
		if err != nil || string(data) != "previous snapshot" {
			t.Fatalf("previous snapshot was damaged: %q, %v", data, err)
		}
	}
}
