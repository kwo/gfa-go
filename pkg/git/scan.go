package git

import (
	"os"
	"path/filepath"

	"github.com/kwo/gfa/pkg/tui"
)

func scanDirectories(dir string) ([]tui.RepoState, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	if isGitRepo(dir) {
		return []tui.RepoState{newRepoState(".")}, nil
	}

	repos := make([]tui.RepoState, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		repos = append(repos, newRepoState(entry.Name()))
	}

	return repos, nil
}

func isGitRepo(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil && (info.IsDir() || info.Mode().IsRegular())
}

func newRepoState(directory string) tui.RepoState {
	return tui.RepoState{
		Directory: directory,
		Branch:    "...",
		Status:    "...",
		Remote:    "...",
		Action:    "...",
	}
}
