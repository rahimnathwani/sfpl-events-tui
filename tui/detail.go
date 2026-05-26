// tui/detail.go
package tui

import (
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"sfpl-events-tui/model"
	"sfpl-events-tui/scraper"
)

var (
	headerStyle       = lipgloss.NewStyle().Bold(true)
	dividerStyle      = lipgloss.NewStyle().Faint(true)
	selectedInstStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	descStyle         = lipgloss.NewStyle().Padding(0, 2)
)

const (
	modalOptOpenWeb = 0
	modalOptGCal    = 1
	modalOptCount   = 2
)

var modalOptions = [modalOptCount]string{"Open web page", "Add to Google Calendar"}

// DetailModel shows the list of instances for one event group and lazily loads
// the description for the selected instance.
type DetailModel struct {
	group       model.EventGroup
	cursor      int
	width       int
	height      int
	showModal   bool
	modalCursor int
}

func NewDetailModel(group model.EventGroup) DetailModel {
	return DetailModel{group: group}
}

// FetchDescriptionCmd fetches the description for the given event URL as a
// background Bubbletea command.
func FetchDescriptionCmd(url string) tea.Cmd {
	return func() tea.Msg {
		resp, err := http.Get(url)
		if err != nil {
			return DescFetchedMsg{URL: url, Text: ""}
		}
		defer resp.Body.Close()
		text := scraper.ParseDescription(resp.Body)
		return DescFetchedMsg{URL: url, Text: text}
	}
}

func (m DetailModel) Update(msg tea.Msg) (DetailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case DescFetchedMsg:
		for i, inst := range m.group.Instances {
			if inst.URL == msg.URL {
				text := msg.Text
				m.group.Instances[i].Description = &text
				break
			}
		}

	case tea.KeyMsg:
		if m.showModal {
			switch msg.String() {
			case "up", "k":
				if m.modalCursor > 0 {
					m.modalCursor--
				}
			case "down", "j":
				if m.modalCursor < modalOptCount-1 {
					m.modalCursor++
				}
			case "enter":
				inst := m.group.Instances[m.cursor]
				m.showModal = false
				switch m.modalCursor {
				case modalOptOpenWeb:
					return m, openURL(inst.URL)
				case modalOptGCal:
					return m, openURL(googleCalendarURL(inst))
				}
			case "esc":
				m.showModal = false
			}
			return m, nil
		}

		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				return m, m.fetchIfNeeded()
			}
		case "down", "j":
			if m.cursor < len(m.group.Instances)-1 {
				m.cursor++
				return m, m.fetchIfNeeded()
			}
		case "enter":
			if len(m.group.Instances) > 0 {
				m.showModal = true
				m.modalCursor = 0
				return m, nil
			}
		case "esc":
			return m, func() tea.Msg { return NavigateBackMsg{} }
		}
	}
	return m, nil
}

func (m DetailModel) fetchIfNeeded() tea.Cmd {
	if len(m.group.Instances) == 0 {
		return nil
	}
	inst := m.group.Instances[m.cursor]
	if inst.Description == nil {
		return FetchDescriptionCmd(inst.URL)
	}
	return nil
}

func (m DetailModel) View() string {
	var sb strings.Builder

	sb.WriteString("\n  ")
	sb.WriteString(headerStyle.Render("◀ " + m.group.Name))
	sb.WriteString("\n\n")

	for i, inst := range m.group.Instances {
		c := noCursorChar
		if i == m.cursor {
			c = cursorChar
		}

		timeStr := inst.Date.Format("Jan 2  15:04")
		if !inst.EndTime.IsZero() {
			timeStr += "–" + inst.EndTime.Format("15:04")
		}

		line := fmt.Sprintf("%s %-22s  %s", c, timeStr, inst.Location)
		if i == m.cursor {
			sb.WriteString(selectedInstStyle.Render(line))
		} else {
			sb.WriteString(line)
		}
		sb.WriteString("\n")
	}

	w := m.width - 2
	if w < 0 {
		w = 0
	}
	divider := dividerStyle.Render(strings.Repeat("─", w))
	sb.WriteString("\n  " + divider + "\n")

	if len(m.group.Instances) > 0 {
		inst := m.group.Instances[m.cursor]
		switch {
		case inst.Description == nil:
			sb.WriteString(descStyle.Render("Loading description…"))
		case *inst.Description == "":
			sb.WriteString(descStyle.Render("No description available."))
		default:
			sb.WriteString(descStyle.Render(*inst.Description))
		}
	}

	sb.WriteString("\n  " + divider + "\n\n")

	if m.showModal {
		for i, opt := range modalOptions {
			if i == m.modalCursor {
				sb.WriteString(selectedInstStyle.Render("  " + cursorChar + " " + opt))
			} else {
				sb.WriteString("    " + opt)
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
		sb.WriteString(lipgloss.NewStyle().Faint(true).Render(
			"  [enter] select   [esc] cancel",
		))
	} else {
		sb.WriteString(lipgloss.NewStyle().Faint(true).Render(
			"  [enter] open   [esc] back   [ctrl+c] quit",
		))
	}
	return sb.String()
}

func googleCalendarURL(inst model.EventInstance) string {
	const layout = "20060102T150405"
	start := inst.Date.Format(layout)
	end := inst.EndTime.Format(layout)
	if inst.EndTime.IsZero() {
		end = inst.Date.Add(time.Hour).Format(layout)
	}
	params := url.Values{}
	params.Set("action", "TEMPLATE")
	params.Set("text", inst.Name)
	params.Set("dates", start+"/"+end)
	if inst.Location != "" {
		params.Set("location", inst.Location)
	}
	if inst.Description != nil && *inst.Description != "" {
		params.Set("details", *inst.Description)
	}
	return "https://calendar.google.com/calendar/render?" + params.Encode()
}

// openURL opens the URL in the default browser and returns a no-op command.
func openURL(url string) tea.Cmd {
	return func() tea.Msg {
		exec.Command("open", url).Start() //nolint:errcheck
		return nil
	}
}
