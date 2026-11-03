package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestCompactTable(t *testing.T) {
	repos := []RepoState{
		{Directory: "gfa-go", Branch: "main", Status: "clean", Remote: "up-to-date", Action: "fetched", Done: true},
		{Directory: "baserss", Branch: "main", Status: "dirty", Remote: "behind 2", Action: "fetched", Done: true},
		{Directory: "homelab", Branch: "main", Status: "clean", Remote: "ahead 1, behind 2", Action: "pulled", Done: true},
		{Directory: "broken", Branch: "topic", Status: "error", Action: "error", Done: true},
	}
	lines := strings.Split(renderRepoTable(repos, 80, repositoryWidth(repos), false, false), "\n")
	want := [][]string{
		{"REPO", "BRANCH", "SYNC", "OP"},
		{"gfa-go", "main", "=", "F"},
		{"baserss", "main*", "↓2", "F"},
		{"homelab", "main", "↑1", "↓2", "P"},
		{"broken", "topic", "?", "!"},
	}
	if len(lines) != len(want) {
		t.Fatalf("got %d rows, want %d", len(lines), len(want))
	}
	for i, line := range lines {
		if got := strings.Join(strings.Fields(line), " "); got != strings.Join(want[i], " ") {
			t.Errorf("row %d: %q, want %v", i, got, want[i])
		}
	}
	if ansi.StringWidth(lines[0]) >= 60 {
		t.Fatal("normal table is not compact")
	}
}

func TestCompactStateSymbols(t *testing.T) {
	for _, tt := range []struct{ input, want string }{
		{"up-to-date", "="},
		{"ahead 2", "↑2"},
		{"behind 3", "↓3"},
		{"ahead 12, behind 34", "↑12 ↓34"},
		{"no remotes", "—"},
		{"no remote branch", "—"},
		{"gone", "?"},
		{"...", "…"},
		{"", "?"},
	} {
		if got := compactSync(tt.input); got != tt.want {
			t.Errorf("sync %q: %q, want %q", tt.input, got, tt.want)
		}
	}
	for _, tt := range []struct {
		repo RepoState
		want string
	}{
		{RepoState{Action: "fetched"}, "F"},
		{RepoState{Action: "pulled"}, "P"},
		{RepoState{Action: "checking"}, "…"},
		{RepoState{Status: "error"}, "!"},
		{RepoState{Error: "fatal"}, "!"},
		{RepoState{Done: true}, "—"},
		{RepoState{}, "…"},
	} {
		if got := operationSymbol(tt.repo); got != tt.want {
			t.Errorf("operation %+v: %q, want %q", tt.repo, got, tt.want)
		}
	}
}

func TestCompactCleanup(t *testing.T) {
	for _, tt := range []struct {
		input, want string
		details     bool
	}{
		{"package-lock restored; switched main; deleted 2; unmerged 1", "R S D2 U1", false},
		{"package-lock unchanged; on default; deleted 0; unmerged 0", "—", false},
		{"package-lock skipped; dirty; skipped switch; dirty; skipped delete", "R? S? D?", true},
		{"deleted 2; skipped 1; unmerged 3", "D2 D?1 U3", true},
		{"unexpected diagnostic", "?", true},
	} {
		got, details := compactCleanup(tt.input)
		if got != tt.want || details != tt.details {
			t.Errorf("cleanup %q: %q/%v, want %q/%v", tt.input, got, details, tt.want, tt.details)
		}
	}
}

func TestTableFitsTerminal(t *testing.T) {
	repos := []RepoState{
		{Directory: "非常に長い名前-é-🙂", Branch: strings.Repeat("feature/", 10), Status: "dirty", Remote: "ahead 12, behind 34", Action: "error\nsecond line\t\x1b[31mred\x1b[0m", Cleanup: "package-lock restored; switched main"},
		{Directory: "small", Branch: "main", Status: "clean", Remote: "up-to-date", Action: "fetched"},
	}
	for _, width := range []int{1, 2, 5, 20, 40, 80, 100, 160} {
		for _, cleanup := range []bool{false, true} {
			for _, color := range []bool{false, true} {
				output := renderRepoTable(repos, width, repositoryWidth(repos), cleanup, color)
				lines := strings.Split(output, "\n")
				if len(lines) != len(repos)+1 {
					t.Fatalf("control characters broke rows: %q", output)
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
		{Directory: "ascii", Branch: "topic", Status: "dirty", Remote: "behind 2", Action: "pulled"},
	}
	repoWidth := repositoryWidth(repos)
	lines := strings.Split(ansi.Strip(renderRepoTable(repos, 80, repoWidth, false, true)), "\n")
	column := ansi.StringWidth(lines[0][:strings.Index(lines[0], "SYNC")])
	for i, symbol := range []string{"=", "↓2"} {
		got := ansi.StringWidth(lines[i+1][:strings.Index(lines[i+1], symbol)])
		if got != column {
			t.Fatalf("sync starts at %d, header at %d", got, column)
		}
	}
	header := lines[0]
	repos[0].Branch = strings.Repeat("x", 100)
	repos[0].Status = "dirty"
	updated := strings.Split(ansi.Strip(renderRepoTable(repos, 80, repoWidth, false, true)), "\n")
	if updated[0] != header {
		t.Fatal("column positions changed with repository state")
	}
	if !strings.Contains(updated[1], "…*") {
		t.Fatal("branch truncation hid the dirty marker")
	}
}

func TestDirtyMarkerSurvivesAnError(t *testing.T) {
	for _, width := range []int{1, 2, 10} {
		cell := branchCell(RepoState{Branch: "feature/長い名前", Status: "dirty"}, width)
		if ansi.StringWidth(cell) > width || !strings.HasSuffix(cell, "*") {
			t.Fatalf("dirty marker at width %d: %q", width, cell)
		}
	}
	repo := applyRepoUpdate(RepoState{Branch: "main"}, RepoUpdate{Status: StringValue("dirty")})
	repo = applyRepoUpdate(repo, RepoUpdate{Status: StringValue("error"), Error: StringValue("fetch failed")})
	if got := branchCell(repo, 20); got != "main*" {
		t.Fatalf("error lost dirty state: %q", got)
	}
	repo = applyRepoUpdate(repo, RepoUpdate{Status: StringValue("clean")})
	if got := branchCell(repo, 20); got != "main" {
		t.Fatalf("clean update retained dirty marker: %q", got)
	}
}

func TestDetailsShowExceptionsWithoutRepeatingSuccesses(t *testing.T) {
	repos := []RepoState{
		{Directory: "success", Cleanup: "package-lock restored; switched main; deleted 2; unmerged 1"},
		{Directory: "skipped", Cleanup: "dirty; skipped switch"},
		{Directory: "broken", Error: "fatal: first line\n\x1b[31msecond line with a long explanation\x1b[0m\r\x07"},
	}
	output := renderDetails(repos, 80, repositoryWidth(repos), true)
	if strings.Contains(output, "success:") {
		t.Fatal("successful cleanup was repeated in details")
	}
	for _, expected := range []string{"skipped:", "skipped switch", "broken:", "fatal: first line", "second line"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("missing %q in %q", expected, output)
		}
	}
	if strings.ContainsAny(output, "\x1b\r\x07") {
		t.Fatal("unsafe terminal controls in details")
	}
	for _, line := range strings.Split(output, "\n") {
		if ansi.StringWidth(line) > 80 {
			t.Errorf("over-wide detail: %q", line)
		}
	}
}

func TestTruncatedCountsRemainAvailableInDetails(t *testing.T) {
	repos := []RepoState{{Directory: "alpha", Cleanup: "package-lock restored; switched main; deleted 12345; unmerged 67890", Remote: "ahead 123456789, behind 987654321"}}
	output := renderDetails(repos, 40, repositoryWidth(repos), true)
	for _, expected := range []string{"12345", "67890", "123456789", "987654321"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("missing full count %q in %q", expected, output)
		}
	}
}
