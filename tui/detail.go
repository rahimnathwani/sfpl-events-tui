// tui/detail.go
package tui

import (
	"fmt"
	"net/http"
	"os/exec"
	"strings"

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

// DetailModel shows the list of instances for one event group and lazily loads
// the description for the selected instance.
type DetailModel struct {
	group  model.EventGroup
	cursor int
	width  int
	height int
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
				url := m.group.Instances[m.cursor].URL
				return m, openURL(url)
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
	sb.WriteString(lipgloss.NewStyle().Faint(true).Render(
		"  [enter] open in browser   [esc] back   [ctrl+c] quit",
	))
	return sb.String()
}

// openURL opens the URL in the default browser and returns a no-op command.
func openURL(url string) tea.Cmd {
	return func() tea.Msg {
		exec.Command("open", url).Start() //nolint:errcheck
		return nil
	}
}
