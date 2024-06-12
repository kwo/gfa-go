package tui

import (
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type StatusLine struct {
	id     string
	lock   *sync.Mutex
	status string
}

type StatusMsg struct {
	ID   string
	Time time.Time
}

func NewStatusLine(id string) *StatusLine {
	return &StatusLine{
		id:     id,
		status: "",
		lock:   &sync.Mutex{},
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

func (z *StatusLine) SetStatus(status string) {
	z.lock.Lock()
	defer z.lock.Unlock()
	z.status = status
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
	return z.GetStatus()
}

func (z *StatusLine) tick() tea.Cmd {
	return tea.Tick(time.Second*1, func(t time.Time) tea.Msg {
		return StatusMsg{
			ID:   z.id,
			Time: t,
		}
	})
}
