package main

var buildVersion = "dev" //nolint:gochecknoglobals // Set by the release build with -ldflags.

func version(_ bool) string {
	return buildVersion
}
