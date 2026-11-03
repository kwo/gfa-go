package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func buildCLI(t *testing.T) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	binary := filepath.Join(t.TempDir(), "gfa")
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}
	return binary
}

func runCLI(t *testing.T, binary, dir string, args ...string) (string, string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if strings.ContainsAny(stdout.String()+stderr.String(), "\x1b\x07") {
		t.Fatal("redirected output contains terminal control codes")
	}
	return stdout.String(), stderr.String(), err
}

func fixtureGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Git fixture: %v\n%s", err, output)
	}
}

func TestCLI(t *testing.T) {
	binary := buildCLI(t)
	t.Run("help and version", func(t *testing.T) {
		for _, arg := range []string{"-v", "--version"} {
			stdout, stderr, err := runCLI(t, binary, t.TempDir(), arg)
			if err != nil || stdout != "dev\n" || stderr != "" {
				t.Fatalf("%s: stdout=%q stderr=%q err=%v", arg, stdout, stderr, err)
			}
		}
		stdout, stderr, err := runCLI(t, binary, t.TempDir(), "--help")
		if err != nil || !strings.Contains(stdout, "--restore-package-lock") || stderr != "" {
			t.Fatalf("help: stdout=%q stderr=%q err=%v", stdout, stderr, err)
		}
		_, stderr, err = runCLI(t, binary, t.TempDir(), "--unknown")
		if err == nil || !strings.Contains(stderr, "unknown") {
			t.Fatal("unknown flag did not fail")
		}
	})
	t.Run("empty directory", func(t *testing.T) {
		stdout, stderr, err := runCLI(t, binary, t.TempDir())
		if err != nil || stderr != "" || !strings.Contains(stdout, "0/0 F0 P0 !0") {
			t.Fatalf("stdout=%q stderr=%q err=%v", stdout, stderr, err)
		}
	})
	t.Run("final state and packed flags", func(t *testing.T) {
		dir := t.TempDir()
		fixtureGit(t, dir, "init", "-b", "main")
		lock := filepath.Join(dir, "package-lock.json")
		if err := os.WriteFile(lock, []byte("discard\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		stdout, stderr, err := runCLI(t, binary, dir, "-Rsd")
		if err != nil || stderr != "" || !strings.Contains(stdout, "CLEANUP") || !strings.Contains(stdout, "1/1 F0 P0 !0") {
			t.Fatalf("stdout=%q stderr=%q err=%v", stdout, stderr, err)
		}
		lines := strings.Split(stdout, "\n")
		if got := strings.Join(strings.Fields(lines[1]), " "); got != ". main — — R" {
			t.Fatalf("unexpected compact cleanup row: %q", got)
		}
		if _, err := os.Stat(lock); !os.IsNotExist(err) {
			t.Fatal("packed cleanup flag did not remove untracked lock")
		}
		if strings.Contains(stdout, "package-lock restored") {
			t.Fatal("successful cleanup was repeated in full")
		}
	})
	t.Run("repository error", func(t *testing.T) {
		dir := t.TempDir()
		fixtureGit(t, dir, "init", "-b", "main")
		fixtureGit(t, dir, "remote", "add", "origin", filepath.Join(t.TempDir(), "missing.git"))
		stdout, stderr, err := runCLI(t, binary, dir)
		if err == nil || !strings.Contains(stdout, "fatal:") || !strings.Contains(stderr, "repositories failed") {
			t.Fatalf("stdout=%q stderr=%q err=%v", stdout, stderr, err)
		}
		if strings.Contains(stdout+stderr, "program was killed") {
			t.Fatal("repository error killed the UI")
		}
	})
}
