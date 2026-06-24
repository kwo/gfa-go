package git

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

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
				ui.UpdateRepo(repo.Directory, tui.RepoUpdate{
					Status: tui.StringValue("processing"),
					Action: tui.StringValue("checking"),
				})

				if !isGitRepo(repo.Directory) {
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
