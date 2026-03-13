package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rdh/gh-prboard/config"
)

var reposRemoveCmd = &cobra.Command{
	Use:   "remove <owner/repo>",
	Short: "Remove a repo from watch list",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if !cfg.RemoveRepo(args[0]) {
			fmt.Printf("Repo %s was not being watched.\n", args[0])
			return nil
		}

		if err := cfg.Save(); err != nil {
			return err
		}

		fmt.Printf("Stopped watching %s\n", args[0])
		return nil
	},
}

func init() {
	reposCmd.AddCommand(reposRemoveCmd)
}
