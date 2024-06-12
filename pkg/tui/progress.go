package tui

import (
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type ProgressLine struct {
	id      string
	spinner spinner.Model
	lock    *sync.Mutex
	status  string
}

type ProgressMsg struct {
	ID         string
	Time       time.Time
	SpinnerCmd tea.Cmd
}

func NewProgressLine(id string) *ProgressLine {
	return &ProgressLine{
		id:      id,
		spinner: spinner.New(),
		status:  "",
		lock:    &sync.Mutex{},
	}
}

func (z *ProgressLine) ID() string {
	return z.id
}

func (z *ProgressLine) GetStatus() string {
	z.lock.Lock()
	defer z.lock.Unlock()
	return z.status
}

func (z *ProgressLine) SetStatus(status string) {
	z.lock.Lock()
	defer z.lock.Unlock()
	z.status = status
}

func (z *ProgressLine) Init() tea.Cmd {
	return z.tick(z.spinner.Tick)
}

func (z *ProgressLine) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ProgressMsg:
		if msg.ID != z.id {
			return z, nil
		}
		if msg.SpinnerCmd != nil {
			spinnerModel, cmd := z.spinner.Update(msg.SpinnerCmd())
			z.spinner = spinnerModel
			return z, z.tick(cmd)
		}
		return z, z.tick(nil)

	default:
		return z, nil
	}
}

func (z *ProgressLine) View() string {
	str := &strings.Builder{}
	str.WriteString(z.spinner.View())
	str.WriteString(" ")
	str.WriteString(z.GetStatus())
	return str.String()
}

func (z *ProgressLine) tick(spinnerCmd tea.Cmd) tea.Cmd {
	return tea.Tick(z.spinner.Spinner.FPS, func(t time.Time) tea.Msg {
		return ProgressMsg{
			ID:         z.id,
			Time:       t,
			SpinnerCmd: spinnerCmd,
		}
	})
}
