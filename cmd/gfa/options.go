package main

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/jessevdk/go-flags"
)

// initOptions processes the options and returns the exit command:
// nil options, true - exit with error code 1
// nil options, false - exit with error code 0
// non-nil options - continue, ignore bool
func initOptions(verbosity *slog.LevelVar) (*Options, bool) {
	opts, err := getOptions()
	if err != nil {
		// don't log, already printed - slog.ErrorContext(ctx, "cannot load options", "err", err)
		return nil, true // os.Exit(1)
	} else if opts == nil {
		return nil, false // help requests, just exit
	}

	// set log level
	opts.SetLogLevel(verbosity)

	// print version and exit
	if opts.PrintVersion {
		fmt.Println(version(false))
		return nil, false
	}

	return opts, false
}

func getOptions() (*Options, error) {
	opts := &Options{}
	_, err := flags.Parse(opts)
	if err != nil {
		var e *flags.Error
		if errors.As(err, &e) && e.Type == flags.ErrHelp {
			return nil, nil // help requested
		}
		return nil, fmt.Errorf("cannot parse flags: %w", err)
	}
	return opts, nil
}

// see https://pkg.go.dev/github.com/jessevdk/go-flags#hdr-Available_field_tags
type Options struct {
	PrintVersion bool   `long:"version" description:"print version and exit"`
	Verbose      []bool `short:"v" long:"verbose" description:"increase verbosity (repeatable)"`
}

func (o *Options) SetLogLevel(l *slog.LevelVar) {
	switch len(o.Verbose) {
	case 0:
		l.Set(slog.LevelError)
	case 1:
		l.Set(slog.LevelWarn)
	case 2:
		l.Set(slog.LevelInfo)
	default:
		l.Set(slog.LevelDebug)
	}
}
