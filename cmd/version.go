package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	appVersion = "dev"
	appCommit  = "none"
	appDate    = "unknown"
)

func SetVersionInfo(version, commit, date string) {
	appVersion = version
	appCommit = commit
	appDate = date
	rootCmd.Version = version
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("gh-prboard %s (commit: %s, built: %s)\n", appVersion, appCommit, appDate)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
