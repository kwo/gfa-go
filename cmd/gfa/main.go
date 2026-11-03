package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kwo/gfa/pkg/git"
	"github.com/kwo/gfa/pkg/tui"
)

func main() {
	verbosity := initLogging()
	opts, failed := initOptions(verbosity)
	if opts == nil {
		if failed {
			os.Exit(1)
		}
		return
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	ui := tui.New(opts.SwitchDefaultBranch || opts.DeleteMergedBranches || opts.RestorePackageLock)
	err := ui.Run(ctx, func(ctx context.Context) error {
		return git.Run(ctx, ui, git.Options{
			SwitchDefaultBranch:  opts.SwitchDefaultBranch,
			DeleteMergedBranches: opts.DeleteMergedBranches,
			RestorePackageLock:   opts.RestorePackageLock,
		})
	})
	if errors.Is(err, context.Canceled) {
		os.Exit(130)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "gfa:", err)
		os.Exit(1)
	}
	slog.InfoContext(ctx, "done")
}
