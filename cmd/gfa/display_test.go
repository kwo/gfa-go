package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLICompactStates(t *testing.T) {
	binary := buildCLI(t)
	t.Run("non-Git directories", func(t *testing.T) {
		dir := t.TempDir()
		fixtureGit(t, dir, "init", "-b", "main", "repo")
		if err := os.Mkdir(filepath.Join(dir, "ignore-me"), 0o700); err != nil {
			t.Fatal(err)
		}
		stdout, stderr, err := runCLI(t, binary, dir)
		if err != nil || stderr != "" {
			t.Fatalf("err=%v stderr=%q", err, stderr)
		}
		if !strings.Contains(stdout, "1/1 F0 P0 !0 · 1 non-Git") || strings.Contains(stdout, "ignore-me") {
			t.Fatalf("non-Git directory was not compacted into the footer: %q", stdout)
		}
		if row := strings.Join(strings.Fields(strings.Split(stdout, "\n")[1]), " "); row != "repo main — —" {
			t.Fatalf("unexpected repository row: %q", row)
		}
	})
	t.Run("dirty branch", func(t *testing.T) {
		dir := t.TempDir()
		fixtureGit(t, dir, "init", "-b", "main")
		if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("keep me"), 0o600); err != nil {
			t.Fatal(err)
		}
		stdout, stderr, err := runCLI(t, binary, dir)
		if err != nil || stderr != "" {
			t.Fatalf("err=%v stderr=%q", err, stderr)
		}
		if row := strings.Join(strings.Fields(strings.Split(stdout, "\n")[1]), " "); row != ". main* — —" {
			t.Fatalf("unexpected dirty branch row: %q", row)
		}
	})
	t.Run("cleanup skip", func(t *testing.T) {
		dir := t.TempDir()
		fixtureGit(t, dir, "init", "-b", "main")
		for _, name := range []string{"README.md", "package-lock.json"} {
			if err := os.WriteFile(filepath.Join(dir, name), []byte("keep me"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		stdout, stderr, err := runCLI(t, binary, dir, "-R")
		if err != nil || stderr != "" {
			t.Fatalf("err=%v stderr=%q", err, stderr)
		}
		if !strings.Contains(stdout, "R?") || !strings.Contains(stdout, "package-lock skipped") {
			t.Fatalf("skip reason was lost: %q", stdout)
		}
		if _, err := os.Stat(filepath.Join(dir, "package-lock.json")); err != nil {
			t.Fatal("skipped cleanup removed the lock file")
		}
	})
}
