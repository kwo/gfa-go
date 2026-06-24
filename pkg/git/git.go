package git

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/kwo/gitterbug/pkg/tui"
)

func Run(ctx context.Context, ui *tui.TUI) error {
	ui.SetStatus("finding git repositories...")

	currdir, err := os.Getwd()
	if err != nil {
		ui.SetStatus("failed to get current directory")
		return err
	}

	repos, err := scanDirectories(".")
	if err != nil {
		ui.SetStatus("failed to list files in current directory")
		return err
	}
	ui.SetRepos(repos)

	ui.SetStatus("loading repositories...")

	wg := &sync.WaitGroup{}
	for _, repo := range repos {
		select {
		case <-ctx.Done():
			return nil
		default:
			wg.Add(1)
			go func(repo tui.RepoState) {
				defer wg.Done()
				processRepo(ctx, ui, repo)
			}(repo)
		}
	} // entries

	ui.SetStatus("processing repositories...")

	wg.Wait()
	ui.SetStatus("processing complete")
	if len(repos) == 0 {
		ui.SetStatus(fmt.Sprintf("processing complete: no directories in %s", currdir))
	}
	return nil
}
