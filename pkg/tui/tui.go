package tui

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"
)

type statusUpdateMsg struct {
	Status string
}

type addProcessMsg struct {
	ID string
}

type progressStatusMsg struct {
	ID     string
	Status string
}

type setReposMsg struct {
	Repos []RepoState
}

type repoUpdateMsg struct {
	Directory string
	Update    RepoUpdate
}

func New() *TUI {
	updates := make(chan tea.Msg, 1024)
	status := NewStatusLine("s1").SetStatus("ready")
	m := &model{
		status:  status,
		updates: updates,
	}
	return &TUI{
		m:       m,
		updates: updates,
		ready:   make(chan struct{}),
	}
}

type TUI struct {
	m             *model
	updates       chan tea.Msg
	nextProcessID atomic.Int64
	programMu     sync.Mutex
	program       *tea.Program
	ready         chan struct{}
}

func (z *TUI) Run(ctx context.Context) error {
	p := tea.NewProgram(z.m, tea.WithContext(ctx))
	z.programMu.Lock()
	z.program = p
	close(z.ready)
	z.programMu.Unlock()
	defer func() {
		z.programMu.Lock()
		z.program = nil
		z.programMu.Unlock()
	}()

	_, err := p.Run()
	return err
}

func (z *TUI) Quit() {
	<-z.ready
	z.programMu.Lock()
	defer z.programMu.Unlock()
	if z.program != nil {
		z.program.Quit()
	}
}

func (z *TUI) SetStatus(status string) {
	z.dispatch(statusUpdateMsg{Status: status})
}

func (z *TUI) AddProcess() *ProgressLine {
	id := fmt.Sprintf("p%d", z.nextProcessID.Add(1))
	z.dispatch(addProcessMsg{ID: id})
	return NewProgressLineHandle(id, z.dispatch)
}

func (z *TUI) SetRepos(repos []RepoState) {
	z.dispatch(setReposMsg{Repos: repos})
}

func (z *TUI) UpdateRepo(directory string, update RepoUpdate) {
	z.dispatch(repoUpdateMsg{Directory: directory, Update: update})
}

func (z *TUI) dispatch(msg tea.Msg) {
	z.updates <- msg
}

type model struct {
	status    *StatusLine
	processes []*ProgressLine
	repos     []RepoState
	updates   <-chan tea.Msg
	quitting  bool
	err       error
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(m.status.Init(), m.waitForUpdate())
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case error:
		m.err = msg
		return m, nil

	case statusUpdateMsg:
		m.status.SetStatus(msg.Status)
		return m, m.waitForUpdate()

	case addProcessMsg:
		process := NewProgressLine(msg.ID)
		m.processes = append(m.processes, process)
		return m, tea.Batch(process.Init(), m.waitForUpdate())

	case progressStatusMsg:
		for _, process := range m.processes {
			if process.ID() == msg.ID {
				process.SetStatus(msg.Status)
				break
			}
		}
		return m, m.waitForUpdate()

	case setReposMsg:
		m.repos = msg.Repos
		return m, m.waitForUpdate()

	case repoUpdateMsg:
		for i, repo := range m.repos {
			if repo.Directory == msg.Directory {
				m.repos[i] = applyRepoUpdate(repo, msg.Update)
				break
			}
		}
		return m, m.waitForUpdate()

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

	if len(m.repos) > 0 {
		str.WriteString(renderRepoTable(m.repos))
		str.WriteString("\n\n")
	}

	str.WriteString(m.status.View())
	str.WriteString("\n")

	for _, line := range m.processes {
		str.WriteString(line.View())
		str.WriteString("\n")
	}

	return str.String()
}

func applyRepoUpdate(repo RepoState, update RepoUpdate) RepoState {
	if update.Branch != nil {
		repo.Branch = *update.Branch
	}
	if update.Status != nil {
		repo.Status = *update.Status
	}
	if update.Remote != nil {
		repo.Remote = *update.Remote
	}
	if update.Action != nil {
		repo.Action = *update.Action
	}
	return repo
}

func (m *model) waitForUpdate() tea.Cmd {
	return func() tea.Msg {
		return <-m.updates
	}
}
