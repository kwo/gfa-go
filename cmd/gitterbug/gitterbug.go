package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kwo/gitterbug/pkg/git"
	"github.com/kwo/gitterbug/pkg/tui"
)

func main() {
	verbosity := initLogging()
	opts, ko := initOptions(verbosity)
	if opts == nil {
		if ko {
			os.Exit(1)
		} else {
			return
		}
	}

	ctx, cancelFn := signalNotifyContextCause(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelFn(nil)

	t := tui.New()
	t.SetStatus("ready")
	go func() {
		cancelFn(t.Run(ctx))
	}()

	go func() {
		git.Run(ctx, t)
	}()

	<-ctx.Done()
	if err := context.Cause(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.ErrorContext(ctx, "tui exit", "err", err)
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
