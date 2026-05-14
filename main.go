package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"sfpl-events-tui/scraper"
	"sfpl-events-tui/store"
	"sfpl-events-tui/tui"
)

func main() {
	months := flag.Int("months", 2, "number of months ahead to fetch")
	flag.Parse()

	archived, err := store.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not load archive: %v\n", err)
		archived = map[string]bool{}
	}

	model := tui.NewAppModel(archived)
	p := tea.NewProgram(model, tea.WithAltScreen())

	go func() {
		instances, err := scraper.Scrape(*months, func(completed, total int) {
			p.Send(tui.ProgressMsg{Completed: completed, Total: total})
		})
		if err != nil {
			p.Send(tui.ScrapeErrMsg{Err: err})
			return
		}
		groups := scraper.GroupInstances(instances, archived)
		p.Send(tui.ScrapeCompleteMsg{Groups: groups})
	}()

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
