package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type RepoState struct {
	Directory string
	Branch    string
	Status    string
	Remote    string
	Action    string
}

type RepoUpdate struct {
	Branch *string
	Status *string
	Remote *string
	Action *string
}

func NewRepoUpdate() RepoUpdate {
	return RepoUpdate{}
}

func StringValue(value string) *string {
	return &value
}

func renderRepoTable(repos []RepoState) string {
	if len(repos) == 0 {
		return ""
	}

	type columnName string
	const (
		directoryColumn columnName = "directory"
		branchColumn    columnName = "branch"
		statusColumn    columnName = "status"
		remoteColumn    columnName = "remote"
		actionColumn    columnName = "action"
	)

	widths := map[columnName]int{
		directoryColumn: len("DIRECTORY"),
		branchColumn:    len("BRANCH"),
		statusColumn:    len("STATUS"),
		remoteColumn:    len("REMOTE"),
		actionColumn:    len("ACTION"),
	}

	for _, repo := range repos {
		widths[directoryColumn] = max(widths[directoryColumn], len(repo.Directory))
		widths[branchColumn] = max(widths[branchColumn], len(repo.Branch))
		widths[statusColumn] = max(widths[statusColumn], len(repo.Status))
		widths[remoteColumn] = max(widths[remoteColumn], len(repo.Remote))
		widths[actionColumn] = max(widths[actionColumn], len(repo.Action))
	}

	var b strings.Builder
	writeCell(&b, headerStyle.Render("DIRECTORY"), widths[directoryColumn])
	writeCell(&b, headerStyle.Render("BRANCH"), widths[branchColumn])
	writeCell(&b, headerStyle.Render("STATUS"), widths[statusColumn])
	writeCell(&b, headerStyle.Render("REMOTE"), widths[remoteColumn])
	writeCell(&b, headerStyle.Render("ACTION"), widths[actionColumn])
	b.WriteString("\n")

	for _, repo := range repos {
		writeCell(&b, repo.Directory, widths[directoryColumn])
		writeCell(&b, repo.Branch, widths[branchColumn])
		writeCell(&b, repo.Status, widths[statusColumn])
		writeCell(&b, repo.Remote, widths[remoteColumn])
		writeCell(&b, repo.Action, widths[actionColumn])
		b.WriteString("\n")
	}

	return strings.TrimRight(b.String(), "\n")
}

func writeCell(b *strings.Builder, value string, width int) {
	b.WriteString(value)
	padding := width - lipgloss.Width(value) + 2
	if padding < 2 {
		padding = 2
	}
	b.WriteString(strings.Repeat(" ", padding))
}

var headerStyle = lipgloss.NewStyle().Bold(true)
