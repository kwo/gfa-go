package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/kwo/gfa/pkg/git"
	"github.com/kwo/gfa/pkg/tui"
)

func main() {
	verbosity := initLogging()
	opts, ko := initOptions(verbosity)
	if opts == nil {
		if ko {
			os.Exit(1)
		}
		return
	}

	ctx, cancelFn := signalNotifyContextCause(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelFn(nil)

	t := tui.New()
	t.SetStatus("ready")
	tuiErr := make(chan error, 1)
	go func() {
		err := t.Run(ctx)
		cancelFn(err)
		tuiErr <- err
	}()

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		cancelFn(git.Run(ctx, t))
	}()

	wg.Wait()
	t.Quit()
	<-ctx.Done()
	if err := <-tuiErr; err != nil && !errors.Is(err, context.Canceled) {
		slog.ErrorContext(ctx, "tui exit", "err", err)
		os.Exit(1)
	}
	if err := context.Cause(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.ErrorContext(ctx, "gfa exit", "err", err)
		os.Exit(1)
	}
	slog.InfoContext(ctx, "done")
}

// context
func signalNotifyContextCause(parent context.Context, signals ...os.Signal) (context.Context, context.CancelCauseFunc) {
	signalCtx, _ := signal.NotifyContext(parent, signals...)
	ctx2, cancel := context.WithCancelCause(signalCtx)
	return ctx2, cancel
}
