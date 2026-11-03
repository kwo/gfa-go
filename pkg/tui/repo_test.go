package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestTableFitsTerminal(t *testing.T) {
	repos := []RepoState{
		{Directory: "非常に長い名前-é-🙂", Branch: strings.Repeat("feature/", 10), Status: "processing", Remote: "ahead 12, behind 34", Action: "error\nsecond line\t\x1b[31mred\x1b[0m", Cleanup: "restored package-lock; switched main"},
		{Directory: "small", Branch: "main", Status: "clean", Remote: "up-to-date", Action: "fetched"},
	}
	for _, width := range []int{1, 2, 5, 20, 40, 80, 100, 160} {
		for _, cleanup := range []bool{false, true} {
			for _, color := range []bool{false, true} {
				output := renderRepoTable(repos, width, cleanup, color)
				lines := strings.Split(output, "\n")
				if len(lines) != len(repos)+1 {
					t.Fatalf("embedded control characters broke rows: %q", output)
				}
				for _, line := range lines {
					if got := ansi.StringWidth(line); got > width {
						t.Errorf("width %d, cleanup %v: line is %d cells: %q", width, cleanup, got, line)
					}
				}
				if !color && strings.Contains(output, "\x1b") {
					t.Fatal("plain output contains escape sequences")
				}
			}
		}
	}
}

func TestTableColumnsAreStableAndUnicodeAware(t *testing.T) {
	repos := []RepoState{
		{Directory: "é漢🙂", Branch: "main", Status: "clean", Remote: "up-to-date", Action: "fetched"},
		{Directory: "ascii", Branch: "feature/long-name", Status: "dirty", Remote: "behind 2", Action: "pulled"},
	}
	lines := strings.Split(ansi.Strip(renderRepoTable(repos, 100, false, true)), "\n")
	column := ansi.StringWidth(lines[0][:strings.Index(lines[0], "STATUS")])
	for i, status := range []string{"clean", "dirty"} {
		got := ansi.StringWidth(lines[i+1][:strings.Index(lines[i+1], status)])
		if got != column {
			t.Fatalf("status starts at %d, header at %d", got, column)
		}
	}
	header := lines[0]
	repos[0].Branch = strings.Repeat("x", 100)
	repos[0].Status = "processing"
	if got := strings.Split(ansi.Strip(renderRepoTable(repos, 100, false, true)), "\n")[0]; got != header {
		t.Fatal("column positions changed with repository state")
	}
}

func TestDetailsPreserveErrorsWithoutTerminalControls(t *testing.T) {
	repos := []RepoState{{Directory: "alpha", Error: "fatal: first line\n\x1b[31msecond line with a long explanation\x1b[0m\r\x07", Cleanup: "switched main; deleted 2; unmerged 1"}}
	output := renderDetails(repos, 30, true)
	for _, expected := range []string{"alpha:", "fatal: first line", "second line", "cleanup:"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("missing %q in %q", expected, output)
		}
	}
	for _, line := range strings.Split(output, "\n") {
		if ansi.StringWidth(line) > 30 {
			t.Errorf("over-wide detail: %q", line)
		}
	}
	if strings.ContainsAny(output, "\x1b\r\x07") {
		t.Fatal("unsafe terminal controls in details")
	}
}
