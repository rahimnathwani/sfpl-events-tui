package tui

import "fmt"

// LoadingModel displays a progress counter while the scraper is running.
type LoadingModel struct {
	Completed int
	Total     int
}

func NewLoadingModel() LoadingModel {
	return LoadingModel{}
}

func (m LoadingModel) View() string {
	if m.Total == 0 {
		return "\n  Fetching SFPL events…\n"
	}
	return fmt.Sprintf("\n  Fetching SFPL events: %d / %d\n", m.Completed, m.Total)
}
