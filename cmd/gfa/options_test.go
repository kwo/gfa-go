package main

import (
	"log/slog"
	"testing"
)

func TestParseOptions(t *testing.T) {
	for _, args := range [][]string{{"-Rsd"}, {"--restore-package-lock", "--switch-default-branch", "--delete-merged-branches"}} {
		opts, err := parseOptions(args)
		if err != nil || !opts.RestorePackageLock || !opts.SwitchDefaultBranch || !opts.DeleteMergedBranches {
			t.Fatalf("args=%v opts=%+v err=%v", args, opts, err)
		}
	}
	for _, arg := range []string{"-v", "--version"} {
		opts, err := parseOptions([]string{arg})
		if err != nil || !opts.PrintVersion || len(opts.Verbose) != 0 {
			t.Fatalf("version alias %q: %+v %v", arg, opts, err)
		}
	}
	opts, err := parseOptions([]string{"--verbose", "--verbose"})
	if err != nil || len(opts.Verbose) != 2 || opts.PrintVersion {
		t.Fatalf("verbosity: %+v %v", opts, err)
	}
	for _, args := range [][]string{{"--unknown"}, {"unexpected"}, {"--", "unexpected"}} {
		if _, err := parseOptions(args); err == nil {
			t.Fatalf("accepted invalid arguments: %v", args)
		}
	}
}

func TestOptionsSetLogLevel(t *testing.T) {
	tests := []struct {
		name    string
		verbose []bool
		want    slog.Level
	}{
		{name: "default error", verbose: nil, want: slog.LevelError},
		{name: "one warn", verbose: []bool{true}, want: slog.LevelWarn},
		{name: "two info", verbose: []bool{true, true}, want: slog.LevelInfo},
		{name: "three debug", verbose: []bool{true, true, true}, want: slog.LevelDebug},
		{name: "many debug", verbose: []bool{true, true, true, true}, want: slog.LevelDebug},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level := &slog.LevelVar{}
			opts := Options{Verbose: tt.verbose}
			opts.SetLogLevel(level)
			if got := level.Level(); got != tt.want {
				t.Fatalf("level = %v, want %v", got, tt.want)
			}
		})
	}
}
