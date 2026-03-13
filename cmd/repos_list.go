package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rdh/gh-prboard/config"
)

var reposListCmd = &cobra.Command{
	Use:   "list",
	Short: "List watched repos",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if len(cfg.Repos) == 0 {
			fmt.Println("No repos configured.")
			return nil
		}

		groups := cfg.ReposByGroup()
		for group, repos := range groups {
			label := group
			if label == "" {
				label = "Other"
			}
			fmt.Printf("## %s\n", label)
			for _, r := range repos {
				fmt.Printf("  %s\n", r.Name)
			}
			fmt.Println()
		}

		return nil
	},
}

func init() {
	reposCmd.AddCommand(reposListCmd)
}
