package tui

import (
	"context"
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
)

type statusUpdateMsg string

type setReposMsg []RepoState

type repoUpdateMsg struct {
	Directory string
	Update    RepoUpdate
}

type completeMsg struct{ err error }

type TUI struct {
	updates chan tea.Msg
	ctx     context.Context
	cleanup bool
}

func New(cleanup bool) *TUI {
	return &TUI{updates: make(chan tea.Msg, 128), cleanup: cleanup}
}

// Run owns the worker lifetime. A completion message follows every repository
// update in the same queue, so the final report cannot overtake pending updates.
func (t *TUI) Run(ctx context.Context, work func(context.Context) error) error {
	width, height, err := term.GetSize(os.Stdout.Fd())
	if err != nil {
		width, height = 120, 24
	}
	interactive := term.IsTerminal(os.Stdin.Fd()) && term.IsTerminal(os.Stdout.Fd()) && os.Getenv("TERM") != "dumb"
	color := term.IsTerminal(os.Stdout.Fd()) && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb"
	return t.run(ctx, work, os.Stdin, os.Stdout, width, height, interactive, color)
}

func (t *TUI) run(ctx context.Context, work func(context.Context) error, input io.Reader, output io.Writer, width, height int, interactive, color bool) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	t.ctx = ctx
	m := &model{ctx: ctx, updates: t.updates, status: "scanning...", width: max(1, width), height: max(1, height), cleanup: t.cleanup, color: color}
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		t.dispatch(completeMsg{err: work(ctx)})
	}()

	var runErr error
	if interactive {
		_, runErr = tea.NewProgram(m, tea.WithContext(ctx), tea.WithInput(input), tea.WithOutput(output), tea.WithoutSignalHandler()).Run()
	} else {
	loop:
		for {
			select {
			case <-ctx.Done():
				runErr = ctx.Err()
				break loop
			case msg := <-t.updates:
				m.Update(msg)
				if m.done {
					break loop
				}
			}
		}
	}
	if !m.done {
		m.status = "cancelled"
		if runErr == nil {
			runErr = context.Canceled
		}
		if ctx.Err() != nil {
			runErr = ctx.Err()
		}
	}
	cancel()
	<-workerDone
	prefix := ""
	if interactive {
		// Bubble Tea clears its last frame with a space. Return to column zero
		// before the permanent report so its header aligns with the rows.
		prefix = "\r"
	}
	if _, err := fmt.Fprintln(output, prefix+m.finalView()); err != nil {
		return err
	}
	if runErr != nil {
		return runErr
	}
	return m.err
}

func (t *TUI) SetStatus(status string) { t.dispatch(statusUpdateMsg(status)) }

func (t *TUI) SetRepos(repos []RepoState) {
	t.dispatch(setReposMsg(append([]RepoState(nil), repos...)))
}

func (t *TUI) UpdateRepo(directory string, update RepoUpdate) {
	t.dispatch(repoUpdateMsg{Directory: directory, Update: update})
}

func (t *TUI) dispatch(msg tea.Msg) {
	select {
	case t.updates <- msg:
	case <-t.ctx.Done():
	}
}

type model struct {
	ctx      context.Context
	updates  <-chan tea.Msg
	repos    []RepoState
	status   string
	width    int
	height   int
	offset   int
	cleanup  bool
	color    bool
	done     bool
	quitting bool
	err      error
}

func (m *model) Init() tea.Cmd { return m.waitForUpdate() }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case statusUpdateMsg:
		m.status = string(msg)
	case setReposMsg:
		m.repos = []RepoState(msg)
	case repoUpdateMsg:
		for i, repo := range m.repos {
			if repo.Directory == msg.Directory {
				m.repos[i] = applyRepoUpdate(repo, msg.Update)
				break
			}
		}
	case completeMsg:
		m.done, m.quitting, m.err = true, true, msg.err
		m.status = "done"
		if len(m.repos) == 0 {
			m.status = "done: no directories found"
		}
		if msg.err != nil {
			m.status = "done: " + singleLine(msg.err.Error())
		}
		return m, tea.Quit
	case tea.WindowSizeMsg:
		m.width, m.height = max(1, msg.Width), max(1, msg.Height)
		m.clampOffset()
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "down", "j":
			m.offset++
		case "up", "k":
			m.offset--
		case "pgdown":
			m.offset += m.pageSize()
		case "pgup":
			m.offset -= m.pageSize()
		case "home":
			m.offset = 0
		case "end":
			m.offset = len(m.repos)
		}
		m.clampOffset()
		return m, nil
	default:
		return m, nil
	}
	return m, m.waitForUpdate()
}

func (m *model) pageSize() int { return max(1, m.height-3) }

func (m *model) clampOffset() {
	m.offset = max(0, min(m.offset, len(m.repos)-m.pageSize()))
}

func (m *model) View() string {
	if m.quitting {
		return ""
	}
	m.clampOffset()
	end := min(len(m.repos), m.offset+m.pageSize())
	table := renderRepoTable(m.repos[m.offset:end], m.width, m.cleanup, m.color)
	status := m.status
	if len(m.repos) > m.pageSize() {
		status += fmt.Sprintf(" | %d-%d of %d | up/down, PgUp/PgDn", m.offset+1, end, len(m.repos))
	}
	view := table + "\n\n" + ansi.Truncate(singleLine(status), m.width, "…")
	if m.height < 4 {
		return ansi.Truncate(singleLine(status), m.width, "…")
	}
	return view
}

func (m *model) finalView() string {
	return renderRepoTable(m.repos, m.width, m.cleanup, m.color) + "\n\n" + ansi.Hardwrap(singleLine(m.status), m.width, true) + "\n" + renderDetails(m.repos, m.width, m.cleanup)
}

func (m *model) waitForUpdate() tea.Cmd {
	return func() tea.Msg {
		select {
		case msg := <-m.updates:
			return msg
		case <-m.ctx.Done():
			return nil
		}
	}
}
