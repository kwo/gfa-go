package git

import (
	"context"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"github.com/kwo/gitterbug/pkg/tui"
)

func Run(ctx context.Context, ui *tui.TUI) error {
	ui.SetStatus("finding git repositories...")

	// TODO recursívely walk subdirs to add git repos including current dir

	// get current directory
	currdir, err := os.Getwd()
	if err != nil {
		ui.SetStatus("failed to get current directory")
		return err
	}

	// read files in current directory
	entries, err := os.ReadDir(currdir)
	if err != nil {
		ui.SetStatus("failed to list files in current directory")
		return err
	}

	repos := make([]tui.RepoState, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		repos = append(repos, tui.RepoState{
			Directory: entry.Name(),
			Branch:    "...",
			Status:    "...",
			Remote:    "...",
			Action:    "...",
		})
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
				ui.UpdateRepo(repo.Directory, tui.RepoUpdate{
					Status: tui.StringValue("processing"),
					Action: tui.StringValue("checking"),
				})

				// check for git subfolder
				if fi, err := os.Stat(path.Join(repo.Directory, ".git")); err != nil || !fi.IsDir() {
					ui.UpdateRepo(repo.Directory, tui.RepoUpdate{
						Branch: tui.StringValue(""),
						Status: tui.StringValue("no git"),
						Remote: tui.StringValue(""),
						Action: tui.StringValue(""),
					})
					return
				}

				time.Sleep(time.Second * 3)
				ui.UpdateRepo(repo.Directory, tui.RepoUpdate{
					Status: tui.StringValue("done"),
					Action: tui.StringValue("checked"),
				})
			}(repo)
		}
	} // entries

	ui.SetStatus("processing repositories...")

	wg.Wait()
	ui.SetStatus("processing complete")
	if len(repos) == 0 {
		ui.SetStatus(fmt.Sprintf("processing complete: no directories in %s", currdir))
	}
	time.Sleep(time.Millisecond * 200) // wait for ui to update
	return nil
}
