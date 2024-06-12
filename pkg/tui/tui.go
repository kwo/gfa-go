package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func New() *TUI {
	// https://github.com/charmbracelet/lipgloss
	// statusStyle := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "12", Dark: "5"})

	status := NewStatusLine("s1").SetStatus("ready") //.SetStyle(statusStyle)
	m := &model{
		status: status,
	}
	return &TUI{m: m}
}

type TUI struct {
	m *model
}

func (z *TUI) Run(ctx context.Context) error {
	p := tea.NewProgram(z.m, tea.WithContext(ctx))
	_, err := p.Run()
	return err
}

func (z *TUI) SetStatus(status string) {
	z.m.status.SetStatus(status)
}

func (z *TUI) AddProcess() *ProgressLine {
	i := len(z.m.processes)
	p := NewProgressLine(fmt.Sprintf("p%d", i+1))
	z.m.processes = append(z.m.processes, p)
	return p
}

type model struct {
	status    *StatusLine
	processes []*ProgressLine
	quitting  bool
	err       error
}

func (m *model) Init() tea.Cmd {
	return m.tick()
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case error:
		m.err = msg
		return m, nil

	case StatusMsg:
		model, cmd := m.status.Update(msg)
		m.status = model.(*StatusLine)
		return m, cmd

	case ProgressMsg:
		for i, p := range m.processes {
			if p.ID() == msg.ID {
				model, cmd := m.processes[i].Update(msg)
				m.processes[i] = model.(*ProgressLine)
				return m, cmd
			}
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			// TODO: set state in each element
			m.quitting = true
			return m, tea.Quit
		default:
			return m, nil
		}

	default:
		return m, nil
	}
}

func (m *model) View() string {
	if m.err != nil {
		return m.err.Error()
	}

	str := &strings.Builder{}

	str.WriteString(m.status.View())
	str.WriteString("\n")

	for _, line := range m.processes {
		str.WriteString(line.View())
		str.WriteString("\n")
	}

	return str.String()
}

func (m *model) tick() tea.Cmd {
	cmds := []tea.Cmd{m.status.Init()}
	for _, p := range m.processes {
		cmds = append(cmds, p.Init())
	}
	return tea.Batch(cmds...)
}
