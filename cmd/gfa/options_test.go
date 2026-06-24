package main

import (
	"log/slog"
	"testing"
)

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
