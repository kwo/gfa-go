package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/jessevdk/go-flags"
)

// initOptions processes the options and returns the exit command:
// nil options, true - exit with error code 1
// nil options, false - exit with error code 0
// non-nil options - continue, ignore bool
func initOptions(verbosity *slog.LevelVar) (*Options, bool) {
	opts, err := getOptions()
	if err != nil {
		fmt.Fprintln(os.Stderr, "gfa:", err)
		return nil, true
	} else if opts == nil {
		return nil, false
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
	return parseOptions(os.Args[1:])
}

func parseOptions(args []string) (*Options, error) {
	opts := &Options{}
	parser := flags.NewParser(opts, flags.HelpFlag|flags.PassDoubleDash)
	remaining, err := parser.ParseArgs(args)
	if err != nil {
		var e *flags.Error
		if errors.As(err, &e) && e.Type == flags.ErrHelp {
			fmt.Println(e.Message)
			return nil, nil
		}
		return nil, fmt.Errorf("cannot parse flags: %w", err)
	}
	if len(remaining) != 0 {
		return nil, fmt.Errorf("unexpected arguments: %v", remaining)
	}
	return opts, nil
}

// see https://pkg.go.dev/github.com/jessevdk/go-flags#hdr-Available_field_tags
type Options struct {
	PrintVersion         bool   `short:"v" long:"version" description:"print version and exit"`
	Verbose              []bool `long:"verbose" description:"increase verbosity (repeatable)"`
	SwitchDefaultBranch  bool   `short:"s" long:"switch-default-branch" description:"switch clean repositories to the origin default branch"`
	DeleteMergedBranches bool   `short:"d" long:"delete-merged-branches" description:"delete local branches merged into the origin default branch"`
	RestorePackageLock   bool   `short:"R" long:"restore-package-lock" description:"discard package-lock.json changes only when no other files are dirty"`
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
