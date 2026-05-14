// tui/list.go
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"sfpl-events-tui/model"
)

var (
	activeTabStyle   = lipgloss.NewStyle().Bold(true).Underline(true).Padding(0, 1)
	inactiveTabStyle = lipgloss.NewStyle().Padding(0, 1).Faint(true)
	selectedRowStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	archivedRowStyle = lipgloss.NewStyle().Faint(true)
	cursorChar       = "▶"
	noCursorChar     = " "
)

// ListModel renders the two-tab event group list.
type ListModel struct {
	groups   []model.EventGroup
	archived map[string]bool
	tab      int    // 0 = Events, 1 = All Events
	cursors  [2]int // cursor per tab
	viewport int    // index of first visible row in current tab
	width    int
	height   int
}

func NewListModel(groups []model.EventGroup, archived map[string]bool) ListModel {
	return ListModel{
		groups:   groups,
		archived: archived,
	}
}

func (m ListModel) visible() []model.EventGroup {
	if m.tab == 0 {
		var result []model.EventGroup
		for _, g := range m.groups {
			if !m.archived[g.Name] {
				result = append(result, g)
			}
		}
		return result
	}
	return m.groups
}

func (m *ListModel) clampCursors() {
	for t := 0; t < 2; t++ {
		prev := m.tab
		m.tab = t
		vis := m.visible()
		m.tab = prev
		if len(vis) == 0 {
			m.cursors[t] = 0
		} else if m.cursors[t] >= len(vis) {
			m.cursors[t] = len(vis) - 1
		}
	}
	m.scrollToCursor()
}

func (m *ListModel) scrollToCursor() {
	cursor := m.cursors[m.tab]
	visH := m.height - 6 // reserve rows for tabs + footer
	if visH < 1 {
		visH = 1
	}
	if cursor < m.viewport {
		m.viewport = cursor
	} else if cursor >= m.viewport+visH {
		m.viewport = cursor - visH + 1
	}
	if m.viewport < 0 {
		m.viewport = 0
	}
}

func (m ListModel) Update(msg tea.Msg) (ListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		vis := m.visible()
		cursor := m.cursors[m.tab]

		switch msg.String() {
		case "up", "k":
			if cursor > 0 {
				m.cursors[m.tab]--
				m.scrollToCursor()
			}
		case "down", "j":
			if cursor < len(vis)-1 {
				m.cursors[m.tab]++
				m.scrollToCursor()
			}
		case "tab":
			m.tab = 1 - m.tab
			m.viewport = 0
			m.scrollToCursor()
		case "e":
			if len(vis) > 0 {
				name := vis[cursor].Name
				return m, func() tea.Msg { return ArchiveToggleMsg{Name: name} }
			}
		case "enter":
			if len(vis) > 0 {
				group := vis[cursor]
				return m, func() tea.Msg { return NavigateToDetailMsg{Group: group} }
			}
		}
	}
	return m, nil
}

func (m ListModel) View() string {
	var sb strings.Builder

	// Tab bar
	tab0 := inactiveTabStyle.Render("Events")
	tab1 := inactiveTabStyle.Render("All Events")
	if m.tab == 0 {
		tab0 = activeTabStyle.Render("Events")
	} else {
		tab1 = activeTabStyle.Render("All Events")
	}
	sb.WriteString(fmt.Sprintf("  %s  %s\n\n", tab0, tab1))

	vis := m.visible()
	cursor := m.cursors[m.tab]
	visH := m.height - 6
	if visH < 1 {
		visH = 1
	}

	nameWidth := m.width - 20
	if nameWidth < 10 {
		nameWidth = 10
	}

	end := m.viewport + visH
	if end > len(vis) {
		end = len(vis)
	}

	for i := m.viewport; i < end; i++ {
		g := vis[i]
		c := noCursorChar
		if i == cursor {
			c = cursorChar
		}

		name := g.Name
		if len(name) > nameWidth {
			name = name[:nameWidth-1] + "…"
		}

		dateStr := ""
		if !g.NextDate().IsZero() {
			dateStr = g.NextDate().Format("Jan 2 3:04pm")
		}

		line := fmt.Sprintf("%s %-*s  %s", c, nameWidth, name, dateStr)

		if i == cursor {
			sb.WriteString(selectedRowStyle.Render(line))
		} else if m.tab == 1 && m.archived[g.Name] {
			sb.WriteString(archivedRowStyle.Render(line))
		} else {
			sb.WriteString(line)
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Faint(true).Render(
		"  [e] archive   [tab] switch tab   [enter] view dates   [ctrl+c] quit",
	))
	return sb.String()
}
