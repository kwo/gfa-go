//go:build unix

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
)

func TestTerminalCancellation(t *testing.T) {
	binary := buildCLI(t)
	dir := t.TempDir()
	fixtureGit(t, dir, "init", "-b", "main")
	wrappers := t.TempDir()
	script := filepath.Join(wrappers, "git")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nsleep 2\nprintf main\n"), 0o700); err != nil { //nolint:gosec // This temporary test helper must be executable.
		t.Fatal(err)
	}
	for _, key := range []string{"q", "\x1b", "\x03"} {
		t.Run(fmt.Sprintf("key %q", key), func(t *testing.T) {
			terminalCancellation(t, binary, dir, wrappers, key, nil)
		})
	}
	for _, sig := range []os.Signal{syscall.SIGINT, syscall.SIGTERM} {
		// Repetition catches the old race between Bubble Tea's signal handler
		// and the application's signal-aware context.
		for i := range 10 {
			t.Run(fmt.Sprintf("%s %d", sig, i), func(t *testing.T) {
				terminalCancellation(t, binary, dir, wrappers, "", sig)
			})
		}
	}
}

func terminalCancellation(t *testing.T, binary, dir, wrappers, key string, sig os.Signal) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "NO_COLOR=1", "PATH="+wrappers+string(os.PathListSeparator)+os.Getenv("PATH"))
	terminal, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 12, Cols: 80})
	if err != nil {
		t.Fatal(err)
	}
	defer terminal.Close()
	ready := make(chan struct{})
	readDone := make(chan string, 1)
	go func() {
		var text strings.Builder
		buf := make([]byte, 4096)
		signaled := false
		for {
			n, err := terminal.Read(buf)
			text.Write(buf[:n])
			if !signaled && strings.Contains(text.String(), "REPO") {
				close(ready)
				signaled = true
			}
			if err != nil {
				readDone <- text.String()
				return
			}
		}
	}()
	select {
	case <-ready:
	case <-ctx.Done():
		t.Fatal("terminal did not show its initial frame")
	}
	if sig != nil {
		err = cmd.Process.Signal(sig)
	} else {
		_, err = terminal.Write([]byte(key))
	}
	if err != nil {
		t.Fatal(err)
	}
	err = cmd.Wait()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 130 {
		t.Fatalf("cancellation returned %v", err)
	}
	select {
	case text := <-readDone:
		if !strings.Contains(text, "cancelled") || strings.Contains(text, "program was killed") {
			t.Fatalf("bad cancellation report: %q", text)
		}
	case <-ctx.Done():
		t.Fatal("terminal reader did not stop")
	}
}
