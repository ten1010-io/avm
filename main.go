package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joonseolee/avm/internal/tui"
)

func main() {
	demoMode := flag.Bool("demo", false, "Run with demo data (no real sysfs access)")
	flag.Parse()

	model := tui.NewModel(*demoMode)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
