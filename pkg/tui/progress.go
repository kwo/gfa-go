package tui

import (
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ProgressLine struct {
	id       string
	spinner  spinner.Model
	lock     *sync.Mutex
	status   string
	style    lipgloss.Style
	dispatch func(tea.Msg)
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
		lock:    &sync.Mutex{},
		status:  "",
	}
}

func NewProgressLineHandle(id string, dispatch func(tea.Msg)) *ProgressLine {
	line := NewProgressLine(id)
	line.dispatch = dispatch
	return line
}

func (z *ProgressLine) ID() string {
	return z.id
}

func (z *ProgressLine) GetStatus() string {
	z.lock.Lock()
	defer z.lock.Unlock()
	return z.status
}

func (z *ProgressLine) SetStatus(status string) *ProgressLine {
	if z.dispatch != nil {
		z.dispatch(progressStatusMsg{ID: z.id, Status: status})
		return z
	}

	z.lock.Lock()
	defer z.lock.Unlock()
	z.status = status
	return z
}

func (z *ProgressLine) SetStyle(style lipgloss.Style) *ProgressLine {
	z.style = style
	return z
}

func (z *ProgressLine) SetSpinnerType(spinner spinner.Spinner) *ProgressLine {
	z.spinner.Spinner = spinner
	return z
}

func (z *ProgressLine) SetSpinnerStyle(style lipgloss.Style) *ProgressLine {
	z.spinner.Style = style
	return z
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
	str.WriteString(z.style.Render(z.GetStatus()))
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
