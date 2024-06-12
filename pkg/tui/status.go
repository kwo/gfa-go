package tui

import (
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type StatusLine struct {
	id     string
	lock   *sync.Mutex
	status string
	style  lipgloss.Style
}

type StatusMsg struct {
	ID   string
	Time time.Time
}

func NewStatusLine(id string) *StatusLine {
	return &StatusLine{
		id:     id,
		lock:   &sync.Mutex{},
		status: "",
	}
}

func (z *StatusLine) ID() string {
	return z.id
}

func (z *StatusLine) GetStatus() string {
	z.lock.Lock()
	defer z.lock.Unlock()
	return z.status
}

func (z *StatusLine) SetStatus(status string) *StatusLine {
	z.lock.Lock()
	defer z.lock.Unlock()
	z.status = status
	return z
}

func (z *StatusLine) SetStyle(style lipgloss.Style) *StatusLine {
	z.style = style
	return z
}

func (z *StatusLine) Init() tea.Cmd {
	return z.tick()
}

func (z *StatusLine) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case StatusMsg:
		if msg.ID != z.id {
			return z, nil
		}
		return z, z.tick()
	default:
		return z, nil
	}
}

func (z *StatusLine) View() string {
	return z.style.Render(z.status)
}

func (z *StatusLine) tick() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return StatusMsg{
			ID:   z.id,
			Time: t,
		}
	})
}
