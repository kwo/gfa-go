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
	Dirty     bool
	Done      bool
}

type RepoUpdate struct {
	Branch  *string
	Status  *string
	Remote  *string
	Action  *string
	Cleanup *string
	Error   *string
	Done    *bool
}

func StringValue(value string) *string { return &value }

func repositoryRows(repos []RepoState) []RepoState {
	rows := make([]RepoState, 0, len(repos))
	for _, repo := range repos {
		if repo.Status != "no git" {
			rows = append(rows, repo)
		}
	}
	return rows
}

func repositoryWidth(repos []RepoState) int {
	width := 4
	for _, repo := range repos {
		width = max(width, ansi.StringWidth(singleLine(repo.Directory)))
	}
	return min(24, width)
}

// Freeze the repository width at discovery. Only a terminal resize changes
// the layout; branch and operation updates must never move the columns.
func columnWidths(width, repoWidth int, cleanup bool) ([]int, string) {
	widths := []int{max(4, repoWidth), 18, 9, 2}
	if cleanup {
		widths = append(widths, 16)
	}
	gap := " "
	if width < len(widths)*2-1 {
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
		switch {
		case widths[1] > 6:
			widths[1]--
		case widths[0] > 4:
			widths[0]--
		case cleanup && widths[4] > 7:
			widths[4]--
		case widths[2] > 4:
			widths[2]--
		default:
			widths[largest]--
		}
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

func branchCell(repo RepoState, width int) string {
	branch := singleLine(repo.Branch)
	if branch == "..." {
		branch = "…"
	}
	if branch == "" {
		branch = "?"
	}
	if repo.Dirty || repo.Status == "dirty" {
		return ansi.Truncate(branch, max(0, width-1), "…") + "*"
	}
	return ansi.Truncate(branch, width, "…")
}

func compactSync(remote string) string {
	switch remote {
	case "up-to-date":
		return "="
	case "no remotes", "no remote branch":
		return "—"
	case "...":
		return "…"
	case "", "gone":
		return "?"
	}
	var parts []string
	for _, part := range strings.Split(remote, ", ") {
		switch {
		case strings.HasPrefix(part, "ahead "):
			parts = append(parts, "↑"+strings.TrimPrefix(part, "ahead "))
		case strings.HasPrefix(part, "behind "):
			parts = append(parts, "↓"+strings.TrimPrefix(part, "behind "))
		default:
			return "?"
		}
	}
	return strings.Join(parts, " ")
}

func operationSymbol(repo RepoState) string {
	switch {
	case repo.Error != "" || repo.Status == "error" || repo.Action == "error":
		return "!"
	case repo.Action == "pulled":
		return "P"
	case repo.Action == "fetched":
		return "F"
	case repo.Done:
		return "—"
	default:
		return "…"
	}
}

func compactCleanup(cleanup string) (string, bool) {
	var codes []string
	details := false
	for _, part := range strings.Split(cleanup, "; ") {
		switch {
		case part == "package-lock restored":
			codes = append(codes, "R")
		case strings.HasPrefix(part, "switched "):
			codes = append(codes, "S")
		case part == "package-lock skipped":
			codes = append(codes, "R?")
			details = true
		case part == "skipped switch":
			codes = append(codes, "S?")
			details = true
		case part == "skipped delete":
			codes = append(codes, "D?")
			details = true
		case part == "", part == "package-lock unchanged", part == "on default", part == "dirty", part == "deleted 0", part == "unmerged 0":
		case strings.HasPrefix(part, "deleted "):
			codes = append(codes, "D"+strings.TrimPrefix(part, "deleted "))
		case strings.HasPrefix(part, "unmerged "):
			codes = append(codes, "U"+strings.TrimPrefix(part, "unmerged "))
		case strings.HasPrefix(part, "skipped "):
			codes = append(codes, "D?"+strings.TrimPrefix(part, "skipped "))
			details = true
		default:
			codes = append(codes, "?")
			details = true
		}
	}
	if len(codes) == 0 {
		return "—", details
	}
	return strings.Join(codes, " "), details
}

func renderRepoTable(repos []RepoState, width, repoWidth int, cleanup, color bool) string {
	if len(repos) == 0 {
		return ""
	}
	width = max(1, width)
	widths, gap := columnWidths(width, repoWidth, cleanup)
	renderer := lipgloss.NewRenderer(io.Discard)
	renderer.SetColorProfile(termenv.Ascii)
	if color {
		renderer.SetColorProfile(termenv.ANSI)
	}
	normal := renderer.NewStyle()
	header := normal.Bold(true)
	green := normal.Foreground(lipgloss.Color("2"))
	yellow := normal.Foreground(lipgloss.Color("3"))
	red := normal.Foreground(lipgloss.Color("1"))
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
	headings := []string{"REPO", "BRANCH", "SYNC", "OP"}
	headerStyles := []lipgloss.Style{header, header, header, header}
	if cleanup {
		headings = append(headings, "CLEANUP")
		headerStyles = append(headerStyles, header)
	}
	writeRow(headings, headerStyles)
	for _, repo := range repos {
		branchStyle, syncStyle, opStyle := normal, normal, normal
		if repo.Dirty || repo.Status == "dirty" {
			branchStyle = yellow
		}
		sync := compactSync(repo.Remote)
		if sync == "=" {
			syncStyle = green
		} else if sync != "…" {
			syncStyle = yellow
		}
		op := operationSymbol(repo)
		switch op {
		case "!":
			opStyle = red
			if sync == "…" {
				sync = "?"
			}
		case "F", "P":
			opStyle = green
		}
		values := []string{repo.Directory, branchCell(repo, widths[1]), sync, op}
		styles := []lipgloss.Style{normal, branchStyle, syncStyle, opStyle}
		if cleanup {
			code, details := compactCleanup(repo.Cleanup)
			cleanupStyle := green
			if details {
				cleanupStyle = yellow
			}
			if repo.Cleanup == "" && !repo.Done {
				code = "…"
			}
			if code == "—" || code == "…" {
				cleanupStyle = normal
			}
			values = append(values, code)
			styles = append(styles, cleanupStyle)
		}
		writeRow(values, styles)
	}
	// A terminal narrower than the column count cannot show every column.
	lines := strings.Split(strings.TrimSuffix(b.String(), "\n"), "\n")
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], width, "")
	}
	return strings.Join(lines, "\n")
}

func renderDetails(repos []RepoState, width, repoWidth int, cleanup bool) string {
	widths, _ := columnWidths(width, repoWidth, cleanup)
	var b strings.Builder
	for _, repo := range repos {
		var details []string
		if repo.Error != "" {
			details = append(details, repo.Error)
		}
		if ansi.StringWidth(compactSync(repo.Remote)) > widths[2] {
			details = append(details, "sync: "+repo.Remote)
		}
		if cleanup && repo.Cleanup != "" {
			code, exceptional := compactCleanup(repo.Cleanup)
			if exceptional || ansi.StringWidth(code) > widths[4] {
				details = append(details, "cleanup: "+repo.Cleanup)
			}
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
			for _, line := range strings.Split(ansi.Wrap(detail, max(1, width-len(indent)), ""), "\n") {
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
	if update.Status != nil && (*update.Status == "clean" || *update.Status == "dirty") {
		repo.Dirty = *update.Status == "dirty"
	}
	if update.Done != nil {
		repo.Done = *update.Done
	}
	return repo
}
