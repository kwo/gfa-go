package git

import (
	"context"
	"fmt"
	"io/fs"
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

	ui.SetStatus("loading repositories...")

	wg := &sync.WaitGroup{}
	for _, e := range entries {
		select {
		case <-ctx.Done():
			return nil
		default:
			if e.IsDir() {
				wg.Add(1)
				go func(entry fs.DirEntry, line *tui.ProgressLine) {
					defer wg.Done()
					line.SetStatus(fmt.Sprintf("%s - processing ...", entry.Name()))
					// check for git subfolder
					if fi, err := os.Stat(path.Join(entry.Name(), ".git")); err != nil || !fi.IsDir() {
						line.SetStatus(fmt.Sprintf("%s - is not a git repository", entry.Name()))
						return
					}
					time.Sleep(time.Second * 3)
					line.SetStatus(fmt.Sprintf("%s - done", entry.Name()))
				}(e, ui.AddProcess())
			}
		}
	} // entries

	ui.SetStatus("processing repositories...")

	wg.Wait()
	ui.SetStatus("processing complete")
	time.Sleep(time.Millisecond * 200) // wait for ui to update
	return nil
}
