package tui

import (
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

type RepoState struct {
	Directory string
	Branch    string
	Status    string
	Remote    string
	Action    string
	Cleanup   string
	Error     string
}

type RepoUpdate struct {
	Branch  *string
	Status  *string
	Remote  *string
	Action  *string
	Cleanup *string
	Error   *string
}

func StringValue(value string) *string { return &value }

// Column widths depend only on the terminal, not on changing repository state.
func columnWidths(width int, cleanup bool) ([]int, string) {
	widths := []int{24, 24, 10, 20, 12}
	if cleanup {
		widths = append(widths, 32)
	}
	gap := "  "
	if width < len(widths)*3-2 {
		gap = ""
	}
	budget := max(len(widths), width-len(gap)*(len(widths)-1))
	for {
		total, largest := 0, 0
		for i, w := range widths {
			total += w
			if w > widths[largest] {
				largest = i
			}
		}
		if total <= budget {
			return widths, gap
		}
		widths[largest]--
	}
}

func singleLine(value string) string {
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, ansi.Strip(value))
	return strings.Join(strings.Fields(value), " ")
}

func renderRepoTable(repos []RepoState, width int, cleanup, color bool) string {
	if len(repos) == 0 {
		return ""
	}
	width = max(1, width)
	widths, gap := columnWidths(width, cleanup)
	renderer := lipgloss.NewRenderer(io.Discard)
	renderer.SetColorProfile(termenv.Ascii)
	if color {
		renderer.SetColorProfile(termenv.ANSI)
	}
	header := renderer.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	cyan := renderer.NewStyle().Foreground(lipgloss.Color("6"))
	style := func(value, kind string) lipgloss.Style {
		s := renderer.NewStyle()
		switch {
		case strings.Contains(value, "error"):
			return s.Foreground(lipgloss.Color("1"))
		case value == "dirty", value == "no git", strings.Contains(value, "behind"),
			strings.Contains(value, "no remote"), strings.Contains(value, "skipped"), value == "gone":
			return s.Foreground(lipgloss.Color("3"))
		case value == "clean", value == "up-to-date", value == "pulled", value == "fetched",
			kind == "cleanup" && value != "" && value != "...":
			return s.Foreground(lipgloss.Color("2"))
		default:
			return s
		}
	}
	var b strings.Builder
	writeRow := func(values []string, styles []lipgloss.Style) {
		for i, value := range values {
			cell := ansi.Truncate(singleLine(value), widths[i], "…")
			b.WriteString(styles[i].Render(cell))
			if i < len(values)-1 {
				b.WriteString(strings.Repeat(" ", widths[i]-ansi.StringWidth(cell)))
				b.WriteString(gap)
			}
		}
		b.WriteByte('\n')
	}
	headings := []string{"DIRECTORY", "BRANCH", "STATUS", "REMOTE", "ACTION"}
	headerStyles := []lipgloss.Style{header, header, header, header, header}
	if cleanup {
		headings = append(headings, "CLEANUP")
		headerStyles = append(headerStyles, header)
	}
	writeRow(headings, headerStyles)
	for _, repo := range repos {
		values := []string{repo.Directory, repo.Branch, repo.Status, repo.Remote, repo.Action}
		styles := []lipgloss.Style{cyan, renderer.NewStyle(), style(repo.Status, ""), style(repo.Remote, ""), style(repo.Action, "")}
		if cleanup {
			values = append(values, repo.Cleanup)
			styles = append(styles, style(repo.Cleanup, "cleanup"))
		}
		writeRow(values, styles)
	}
	// Extremely narrow terminals cannot show even one cell per column.
	lines := strings.Split(strings.TrimSuffix(b.String(), "\n"), "\n")
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], width, "")
	}
	return strings.Join(lines, "\n")
}

func renderDetails(repos []RepoState, width int, cleanup bool) string {
	var b strings.Builder
	for _, repo := range repos {
		var details []string
		if repo.Error != "" {
			details = append(details, repo.Error)
		}
		if cleanup && repo.Cleanup != "" {
			details = append(details, "cleanup: "+repo.Cleanup)
		}
		if len(details) == 0 {
			continue
		}
		fmt.Fprintf(&b, "\n%s\n", ansi.Hardwrap(singleLine(repo.Directory)+":", max(1, width), true))
		for _, detail := range details {
			// Preserve diagnostic line breaks, but never terminal control codes.
			detail = strings.Map(func(r rune) rune {
				if r != '\n' && unicode.IsControl(r) {
					return ' '
				}
				return r
			}, ansi.Strip(detail))
			indent := "  "
			if width < 3 {
				indent = ""
			}
			for _, line := range strings.Split(ansi.Hardwrap(detail, max(1, width-len(indent)), true), "\n") {
				b.WriteString(indent + line + "\n")
			}
		}
	}
	return b.String()
}

func applyRepoUpdate(repo RepoState, update RepoUpdate) RepoState {
	for _, field := range []struct {
		dst *string
		src *string
	}{
		{&repo.Branch, update.Branch},
		{&repo.Status, update.Status},
		{&repo.Remote, update.Remote},
		{&repo.Action, update.Action},
		{&repo.Cleanup, update.Cleanup},
		{&repo.Error, update.Error},
	} {
		if field.src != nil {
			*field.dst = *field.src
		}
	}
	return repo
}
