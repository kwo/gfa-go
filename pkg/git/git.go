package git

import (
	"context"
	"fmt"
	"sync"

	"github.com/kwo/gfa/pkg/tui"
)

type Options struct {
	SwitchDefaultBranch  bool
	DeleteMergedBranches bool
	RestorePackageLock   bool
}

type Reporter interface {
	SetStatus(string)
	SetRepos([]tui.RepoState)
	UpdateRepo(string, tui.RepoUpdate)
}

func Run(ctx context.Context, ui Reporter, options Options) error {
	ui.SetStatus("finding git repositories...")
	repos, err := scanDirectories(".")
	if err != nil {
		return fmt.Errorf("list directories: %w", err)
	}
	ui.SetRepos(repos)
	ui.SetStatus("processing repositories...")

	// Limit simultaneous Git/network operations without stopping other rows
	// when one repository fails.
	jobs := make(chan int)
	errs := make([]error, len(repos))
	var wg sync.WaitGroup
	for range min(8, len(repos)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				if ctx.Err() != nil {
					continue
				}
				err := processRepo(ctx, ui, repos[i], options)
				if err != nil {
					setRepoError(ui, repos[i].Directory, err)
					errs[i] = err
				}
				done := ctx.Err() == nil
				ui.UpdateRepo(repos[i].Directory, tui.RepoUpdate{Done: &done})
			}
		}()
	}
launch:
	for i := range repos {
		select {
		case jobs <- i:
		case <-ctx.Done():
			break launch
		}
	}
	close(jobs)
	wg.Wait()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	failed := 0
	for _, err := range errs {
		if err != nil {
			failed++
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d repositories failed", failed)
	}
	return nil
}
