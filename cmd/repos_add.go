package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rdh/gh-prboard/config"
)

var addGroupFlag string

var reposAddCmd = &cobra.Command{
	Use:   "add <owner/repo>",
	Short: "Add a repo to watch",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		cfg.AddRepo(args[0], addGroupFlag)

		if err := cfg.Save(); err != nil {
			return err
		}

		fmt.Printf("Watching %s", args[0])
		if addGroupFlag != "" {
			fmt.Printf(" (group: %s)", addGroupFlag)
		}
		fmt.Println()
		return nil
	},
}

func init() {
	reposAddCmd.Flags().StringVar(&addGroupFlag, "group", "", "assign repo to a group")
	reposCmd.AddCommand(reposAddCmd)
}
