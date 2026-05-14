package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"sfpl-events-tui/store"
)

type screen int

const (
	screenLoading screen = iota
	screenList
	screenDetail
)

// AppModel is the root Bubbletea model. It owns all sub-models and routes
// messages between them.
type AppModel struct {
	screen   screen
	archived map[string]bool
	loading  LoadingModel
	list     ListModel
	detail   DetailModel
}

func NewAppModel(archived map[string]bool) AppModel {
	return AppModel{
		screen:   screenLoading,
		archived: archived,
		loading:  NewLoadingModel(),
	}
}

func (m AppModel) Init() tea.Cmd { return nil }

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case ProgressMsg:
		m.loading.Completed = msg.Completed
		m.loading.Total = msg.Total
		return m, nil

	case ScrapeCompleteMsg:
		m.list = NewListModel(msg.Groups, m.archived)
		m.screen = screenList
		return m, nil

	case ScrapeErrMsg:
		// Quit on fatal scrape error; real app could show an error screen
		return m, tea.Quit

	case NavigateToDetailMsg:
		m.detail = NewDetailModel(msg.Group)
		m.screen = screenDetail
		var cmd tea.Cmd
		if len(msg.Group.Instances) > 0 && msg.Group.Instances[0].Description == nil {
			cmd = FetchDescriptionCmd(msg.Group.Instances[0].URL)
		}
		return m, cmd

	case NavigateBackMsg:
		m.screen = screenList
		return m, nil

	case ArchiveToggleMsg:
		if m.archived[msg.Name] {
			delete(m.archived, msg.Name)
		} else {
			m.archived[msg.Name] = true
		}
		store.Save(m.archived) // synchronous; file is tiny
		m.list.archived = m.archived
		for i, g := range m.list.groups {
			if g.Name == msg.Name {
				m.list.groups[i].Archived = m.archived[msg.Name]
			}
		}
		m.list.clampCursors()
		return m, nil

	case DescFetchedMsg:
		newDetail, cmd := m.detail.Update(msg)
		m.detail = newDetail
		return m, cmd
	}

	// Delegate to the active screen.
	switch m.screen {
	case screenLoading:
		return m, nil
	case screenList:
		newList, cmd := m.list.Update(msg)
		m.list = newList
		return m, cmd
	case screenDetail:
		newDetail, cmd := m.detail.Update(msg)
		m.detail = newDetail
		return m, cmd
	}

	return m, nil
}

func (m AppModel) View() string {
	switch m.screen {
	case screenLoading:
		return m.loading.View()
	case screenList:
		return m.list.View()
	case screenDetail:
		return m.detail.View()
	}
	return ""
}

// Ensure AppModel satisfies tea.Model at compile time.
var _ tea.Model = AppModel{}
