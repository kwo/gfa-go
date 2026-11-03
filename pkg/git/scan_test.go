package git

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/kwo/gfa/pkg/tui"
)

func TestScanDirectoriesUsesCurrentDirectoryWhenItIsGitRepo(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, ".git"))
	mustMkdir(t, filepath.Join(dir, "child"))

	repos, err := scanDirectories(dir)
	if err != nil {
		t.Fatalf("scan directories: %v", err)
	}

	want := []tui.RepoState{newRepoState(".")}
	if !reflect.DeepEqual(repos, want) {
		t.Fatalf("repos = %#v, want %#v", repos, want)
	}
}

func TestScanDirectoriesUsesImmediateChildDirectories(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "alpha"))
	mustMkdir(t, filepath.Join(dir, "beta"))
	mustMkdir(t, filepath.Join(dir, "beta", "nested"))
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("ignore me"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	repos, err := scanDirectories(dir)
	if err != nil {
		t.Fatalf("scan directories: %v", err)
	}

	want := []tui.RepoState{newRepoState("alpha"), newRepoState("beta")}
	if !reflect.DeepEqual(repos, want) {
		t.Fatalf("repos = %#v, want %#v", repos, want)
	}
}

func TestIsGitRepo(t *testing.T) {
	dir := t.TempDir()
	if isGitRepo(dir) {
		t.Fatal("empty directory reported as git repo")
	}

	mustMkdir(t, filepath.Join(dir, ".git"))
	if !isGitRepo(dir) {
		t.Fatal("directory with .git was not reported as git repo")
	}
}

func TestScanDirectoriesRecognizesGitFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".git"), "gitdir: /some/worktree/path\n")
	repos, err := scanDirectories(dir)
	if err != nil || len(repos) != 1 || repos[0].Directory != "." || !isGitRepo(dir) {
		t.Fatalf("worktree not recognized: repos=%+v err=%v", repos, err)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o700); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}
