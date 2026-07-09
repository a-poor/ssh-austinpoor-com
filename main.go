package main

import (
	"embed"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
)

//go:embed content/*.md
var contentfs embed.FS

func main() {
	p := tea.NewProgram(makeModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
