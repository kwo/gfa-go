package main

import (
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
)

func initLogging() *slog.LevelVar {
	verbosity := &slog.LevelVar{}
	slog.SetDefault(slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level:   verbosity,
		NoColor: true,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			if a.Key == slog.LevelKey {
				return slog.Attr{}
			}
			return a
		},
	})))
	return verbosity
}
