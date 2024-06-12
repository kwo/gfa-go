package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/lmittmann/tint"
)

func main() {
	verbosity := &slog.LevelVar{}
	slog.SetDefault(slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		Level:      verbosity,
		TimeFormat: time.TimeOnly,
	})))

	opts, err := getOptions()
	if err != nil {
		// don't log, already printed - slog.ErrorContext(ctx, "cannot load options", "err", err)
		os.Exit(1)
	} else if opts == nil {
		return // help requests, just exit
	}

	// set log level
	opts.SetLogLevel(verbosity)

	// print version and exit
	if opts.PrintVersion {
		fmt.Println(version(true))
		return
	}

	_, cancelFn := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelFn()

	slog.Info("Gitterbug")
}

// options

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
	PrintVersion bool   `                                          long:"version"  description:"print version and exit"`
	Verbosity    string `default:"INFO"  env:"GITTERBUG_LOGLEVEL"  long:"loglevel" description:"set the verbosity level" choice:"ERROR" choice:"WARN" choice:"INFO" choice:"DEBUG"`
}

func (o *Options) SetLogLevel(l *slog.LevelVar) {
	switch o.Verbosity {
	case "ERROR":
		l.Set(slog.LevelError)
	case "WARN":
		l.Set(slog.LevelWarn)
	case "INFO":
		l.Set(slog.LevelInfo)
	case "DEBUG":
		l.Set(slog.LevelDebug)
	}
}

func version(extended bool) string {
	// see https://pkg.go.dev/runtime/debug#BuildSetting
	if info, ok := debug.ReadBuildInfo(); ok {
		var revision, revisionTime string
		var dirty bool
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				revision = setting.Value
			case "vcs.time":
				revisionTime = setting.Value
			case "vcs.modified":
				dirty, _ = strconv.ParseBool(setting.Value)
			}
		}
		if len(revision) > 0 {
			if len(revision) > 8 {
				revision = revision[:8]
			}
			if len(revisionTime) > 10 {
				revisionTime = revisionTime[:10] // date only
			}
			if dirty {
				revision += "*"
			}
			if extended {
				names := strings.Split(info.Main.Path, "/")
				name := names[len(names)-1]
				return fmt.Sprintf("%s %s %s %s", name, revision, revisionTime, info.GoVersion)
			}
			return revision
		}
	}
	return "unknown"
}
