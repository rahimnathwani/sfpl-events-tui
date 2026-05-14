package tui

import "sfpl-events-tui/model"

// ProgressMsg is sent by the scraper goroutine after each completed base request.
type ProgressMsg struct{ Completed, Total int }

// ScrapeCompleteMsg is sent when all scrape requests have finished.
type ScrapeCompleteMsg struct{ Groups []model.EventGroup }

// ScrapeErrMsg is sent if the scraper fails fatally.
type ScrapeErrMsg struct{ Err error }

// NavigateToDetailMsg is sent by ListModel when the user presses Enter.
type NavigateToDetailMsg struct{ Group model.EventGroup }

// NavigateBackMsg is sent by DetailModel when the user presses Esc.
type NavigateBackMsg struct{}

// ArchiveToggleMsg is sent by ListModel when the user presses 'e'.
type ArchiveToggleMsg struct{ Name string }

// DescFetchedMsg is sent when a description HTTP fetch completes.
type DescFetchedMsg struct{ URL, Text string }
