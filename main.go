package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/richhaase/gh-prboard/cmd"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func init() {
	// Fallback to build info for `go install` builds
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok {
			for _, setting := range info.Settings {
				switch setting.Key {
				case "vcs.revision":
					if len(setting.Value) > 7 {
						commit = setting.Value[:7]
					} else {
						commit = setting.Value
					}
				case "vcs.time":
					date = setting.Value
				case "vcs.modified":
					if setting.Value == "true" {
						commit += "-dirty"
					}
				}
			}
		}
	}
	cmd.SetVersionInfo(version, commit, date)
}

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
