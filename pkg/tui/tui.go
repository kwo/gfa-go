package tui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

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
	ctx       context.Context
	updates   <-chan tea.Msg
	repos     []RepoState
	status    string
	width     int
	repoWidth int
	height    int
	offset    int
	cleanup   bool
	color     bool
	done      bool
	quitting  bool
	err       error
}

func (m *model) Init() tea.Cmd { return m.waitForUpdate() }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case statusUpdateMsg:
		m.status = string(msg)
	case setReposMsg:
		m.repos = []RepoState(msg)
		m.repoWidth = repositoryWidth(m.repos)
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
		if errors.Is(msg.err, context.Canceled) {
			m.status = "cancelled"
		}
		return m, tea.Quit
	case tea.WindowSizeMsg:
		m.width, m.height = max(1, msg.Width), max(1, msg.Height)
		m.repoWidth = repositoryWidth(repositoryRows(m.repos))
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

func (m *model) pageSize() int { return max(1, m.height-2) }

func (m *model) clampOffset() {
	count := 0
	for _, repo := range m.repos {
		if repo.Status != "no git" {
			count++
		}
	}
	m.offset = max(0, min(m.offset, count-m.pageSize()))
}

func (m *model) summary() string {
	total, done, fetched, pulled, failed, ignored := 0, 0, 0, 0, 0, 0
	for _, repo := range m.repos {
		if repo.Status == "no git" {
			ignored++
			continue
		}
		total++
		if !repo.Done {
			continue
		}
		done++
		switch operationSymbol(repo) {
		case "F":
			fetched++
		case "P":
			pulled++
		case "!":
			failed++
		}
	}
	text := fmt.Sprintf("%d/%d F%d P%d !%d", done, total, fetched, pulled, failed)
	if ignored > 0 {
		text += fmt.Sprintf(" · %d non-Git", ignored)
	}
	if m.status == "cancelled" {
		text = "cancelled · " + text
	} else if m.done && m.err != nil && failed == 0 {
		text += " · ! " + singleLine(m.err.Error())
	} else if !m.done && len(m.repos) == 0 {
		text = "…"
	}
	return text
}

func (m *model) View() string {
	if m.quitting {
		return ""
	}
	m.clampOffset()
	rows := repositoryRows(m.repos)
	end := min(len(rows), m.offset+m.pageSize())
	if m.repoWidth == 0 {
		m.repoWidth = repositoryWidth(m.repos)
	}
	table := renderRepoTable(rows[m.offset:end], m.width, m.repoWidth, m.cleanup, m.color)
	status := m.summary()
	if len(rows) > m.pageSize() {
		status += fmt.Sprintf(" [%d-%d] ↑/↓ PgUp/PgDn", m.offset+1, end)
	}
	footer := ansi.Truncate(singleLine(status), m.width, "…")
	if m.height < 3 || table == "" {
		return footer
	}
	return table + "\n" + footer
}

func (m *model) finalView() string {
	rows := repositoryRows(m.repos)
	if m.repoWidth == 0 {
		m.repoWidth = repositoryWidth(m.repos)
	}
	var parts []string
	if table := renderRepoTable(rows, m.width, m.repoWidth, m.cleanup, m.color); table != "" {
		parts = append(parts, table)
	}
	parts = append(parts, ansi.Hardwrap(m.summary(), m.width, true))
	if len(rows) > 0 {
		parts = append(parts, ansi.Hardwrap("* dirty  ↑ ahead  ↓ behind  F fetched  P pulled  ! error", m.width, true))
		if m.cleanup {
			parts = append(parts, ansi.Hardwrap("R locks  S branch  D# deleted  U# unmerged  ? skipped", m.width, true))
		}
	}
	return strings.Join(parts, "\n") + renderDetails(rows, m.width, m.repoWidth, m.cleanup)
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
