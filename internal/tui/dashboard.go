package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gh-xj/omnicontext/internal/store"
)

type model struct {
	lines []string
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		s := msg.String()
		if s == "q" || s == "ctrl+c" {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	return strings.Join(m.lines, "\n") + "\n"
}

func RunDashboard(st *store.Store) error {
	contexts, err := st.ListContexts()
	if err != nil {
		return err
	}
	lines := []string{
		"ocx dashboard (MVP)",
		"",
		"Contexts:",
	}
	if len(contexts) == 0 {
		lines = append(lines, "- (none)")
	}
	for _, c := range contexts {
		lines = append(lines, fmt.Sprintf("- %s (%s)", c.ID, c.Name))
	}
	lines = append(lines, "", "Press q to quit.")
	p := tea.NewProgram(model{lines: lines})
	_, err = p.Run()
	return err
}
