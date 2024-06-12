package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	status    *StatusLine
	processes []*ProgressLine
	quitting  bool
	err       error
	counter   int
}

type tickMsg time.Time

func Run(ctx context.Context) error {
	status := NewStatusLine("s1")
	status.SetStatus("ready")

	p1 := NewProgressLine("p1")
	p1.SetStatus("updating")

	p2 := NewProgressLine("p2")
	p2.SetStatus("updating")

	m := &model{
		status:    status,
		processes: []*ProgressLine{p1, p2},
	}

	p := tea.NewProgram(m, tea.WithContext(ctx))
	_, err := p.Run()
	return err
}

func (m *model) Init() tea.Cmd {
	return m.tick()
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tickMsg:
		m.counter++
		m.status.SetStatus(time.Now().Format(time.TimeOnly))
		m.processes[0].SetStatus(fmt.Sprintf("%d", m.counter*2))
		m.processes[1].SetStatus(fmt.Sprintf("%d", m.counter*3))
		return m, m.tick()

	case error:
		m.err = msg
		return m, nil

	case StatusMsg:
		model, cmd := m.status.Update(msg)
		if sl, ok := model.(*StatusLine); ok {
			m.status = sl
		}
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

	if m.quitting {
		str.WriteString("\n")
	}
	return str.String()
}

func (m *model) tick() tea.Cmd {
	t1 := tea.Tick(time.Second*1, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
	return tea.Batch(t1, m.status.Init(), m.processes[0].Init(), m.processes[1].Init())
}
