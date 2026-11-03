//go:build unix

package git

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func installGitScript(t *testing.T, script string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "git")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o700); err != nil { //nolint:gosec // This temporary test helper must be executable.
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return dir
}

func TestGitKeepsSuccessfulStderrOutOfParsedOutput(t *testing.T) {
	dir := installGitScript(t, "echo warning >&2\nprintf '1\\t2\\n'\n")
	output, err := runGit(context.Background(), dir, "anything")
	if err != nil || output != "1\t2\n" {
		t.Fatalf("output=%q err=%v", output, err)
	}
}

func TestGitCancellationStopsDescendants(t *testing.T) {
	dir := installGitScript(t, "(sleep 0.5; echo leaked > child-finished) &\necho ready > ready\nwait\n")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	finished := make(chan error, 1)
	go func() { _, err := runGit(ctx, dir, "anything"); finished <- err }()
	deadline := time.Now().Add(3 * time.Second)
	for {
		if _, err := os.Stat(filepath.Join(dir, "ready")); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("Git subprocess did not start")
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	select {
	case err := <-finished:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected cancellation, got %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Git subprocess did not stop")
	}
	time.Sleep(600 * time.Millisecond)
	if _, err := os.Stat(filepath.Join(dir, "child-finished")); !os.IsNotExist(err) {
		t.Fatal("Git child process survived cancellation")
	}
}
