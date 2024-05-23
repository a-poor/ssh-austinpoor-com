package main

import (
	"github.com/a-poor/ssh-austinpoor-com/pkg/app"
	tea "github.com/charmbracelet/bubbletea"
)

// import (
// 	"fmt"

// 	_ "github.com/a-poor/ssh-austinpoor-com/pkg/app"
// 	"github.com/charmbracelet/glamour"
// )

// const txt = `# Hi, I'm Austin! :wave:

// ![Austin Poor](https://austinpoor.com/static/images/austinpoor.jpg)

// I'm a software engineer living in Los Angeles, CA.
// `

// func main() {
// 	r, err := glamour.NewTermRenderer(
// 		// glamour.WithAutoStyle(),
// 		glamour.WithStandardStyle("dracula"),
// 		glamour.WithEmoji(),
// 		glamour.WithWordWrap(80),
// 	)
// 	if err != nil {
// 		panic(err)
// 	}

// 	out, err := r.Render(txt)
// 	if err != nil {
// 		panic(err)
// 	}
// 	fmt.Println(out)
// }

func main() {
	m, err := app.NewMDViewer()
	if err != nil {
		panic(err)
	}
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		panic(err)
	}
}
