package tui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestCompletionDrainsUpdatesAndPrintsEveryRow(t *testing.T) {
	for _, interactive := range []bool{false, true} {
		t.Run(fmt.Sprint(interactive), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			ui := New(false)
			var output bytes.Buffer
			err := ui.run(ctx, func(context.Context) error {
				repos := make([]RepoState, 200)
				for i := range repos {
					repos[i].Directory = fmt.Sprintf("repo-%03d", i)
				}
				ui.SetRepos(repos)
				for _, repo := range repos {
					ui.UpdateRepo(repo.Directory, RepoUpdate{Status: StringValue("clean"), Action: StringValue("fetched")})
				}
				return nil
			}, nil, &output, 80, 10, interactive, false)
			if err != nil {
				t.Fatalf("successful run returned %v", err)
			}
			text := output.String()
			report := text
			if interactive {
				start := strings.LastIndex(text, "\rDIRECTORY")
				if start < 0 {
					t.Fatal("final header did not return to column zero")
				}
				report = text[start+1:]
			}
			if strings.Count(report, "fetched") != 200 {
				t.Fatal("final report lost repository updates")
			}
			for i := range 200 {
				if !strings.Contains(text, fmt.Sprintf("repo-%03d", i)) {
					t.Fatalf("missing final row %d", i)
				}
			}
			if !strings.Contains(text, "done") || strings.Contains(text, "program was killed") {
				t.Fatalf("bad final report: %q", text)
			}
		})
	}
}

func TestWorkerFailureAppearsInFinalReport(t *testing.T) {
	ui := New(false)
	var output bytes.Buffer
	want := errors.New("1 repository failed")
	err := ui.run(context.Background(), func(context.Context) error {
		ui.SetRepos([]RepoState{{Directory: "broken"}})
		ui.UpdateRepo("broken", RepoUpdate{Status: StringValue("error"), Error: StringValue("fatal: inaccessible\nsecond line")})
		return want
	}, nil, &output, 80, 24, false, false)
	if !errors.Is(err, want) || !strings.Contains(output.String(), "second line") {
		t.Fatalf("err=%v, output=%q", err, output.String())
	}
}

func TestCancellationStopsWorkersWithoutBlockedUpdates(t *testing.T) {
	for _, interactive := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		ui := New(false)
		var output bytes.Buffer
		finished := make(chan error, 1)
		go func() {
			finished <- ui.run(ctx, func(ctx context.Context) error {
				cancel()
				<-ctx.Done()
				for range 1000 {
					ui.SetStatus("stopping")
				}
				return ctx.Err()
			}, nil, &output, 80, 24, interactive, false)
		}()
		select {
		case err := <-finished:
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("expected cancellation, got %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("cancellation left a blocked worker")
		}
		cancel()
	}
}

func TestQuitKeysCancelWork(t *testing.T) {
	for _, key := range []string{"q", "\x1b", "\x03"} {
		t.Run(fmt.Sprintf("%q", key), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
			defer cancel()
			ui := New(false)
			var output bytes.Buffer
			err := ui.run(ctx, func(ctx context.Context) error {
				<-ctx.Done()
				return ctx.Err()
			}, strings.NewReader(key), &output, 80, 24, true, false)
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("quit returned %v", err)
			}
			if !strings.Contains(output.String(), "cancelled") {
				t.Fatal("missing cancellation report")
			}
		})
	}
}

func TestViewportResizeAndScrolling(t *testing.T) {
	m := &model{ctx: context.Background(), width: 80, height: 8, status: "working"}
	for i := range 30 {
		m.repos = append(m.repos, RepoState{Directory: fmt.Sprintf("repo-%02d", i)})
	}
	assertFits := func() {
		t.Helper()
		lines := strings.Split(m.View(), "\n")
		if len(lines) > m.height {
			t.Fatalf("view is %d lines in %d-line terminal", len(lines), m.height)
		}
		for _, line := range lines {
			if ansi.StringWidth(line) > m.width {
				t.Fatal("view exceeds terminal width")
			}
		}
	}
	assertFits()
	m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	if !strings.Contains(m.View(), "repo-29") {
		t.Fatal("cannot scroll to final repository")
	}
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 5})
	assertFits()
	m.Update(tea.KeyMsg{Type: tea.KeyHome})
	if m.offset != 0 {
		t.Fatal("cannot return to first repository")
	}
	m.Update(tea.WindowSizeMsg{Width: 2, Height: 1})
	assertFits()
}
