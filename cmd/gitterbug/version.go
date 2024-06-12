package main

import (
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
)

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
