package cmd

import "github.com/spf13/cobra"

var reposCmd = &cobra.Command{
	Use:   "repos",
	Short: "Manage watched repositories",
}

func init() {
	rootCmd.AddCommand(reposCmd)
}
