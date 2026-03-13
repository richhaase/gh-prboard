package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/spf13/cobra"

	"github.com/richhaase/gh-prboard/config"
	ghapi "github.com/richhaase/gh-prboard/github"
)

var reposDiscoverCmd = &cobra.Command{
	Use:   "discover [org]",
	Short: "Discover repos from configured orgs or a specific org",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		var orgs []string
		if len(args) == 1 {
			orgs = []string{args[0]}
		} else {
			orgs = cfg.Orgs
			if len(orgs) == 0 {
				fmt.Fprintln(os.Stderr, "No orgs configured. Run `gh prboard init` to get started.")
				fmt.Fprintln(os.Stderr, "Or specify an org directly: gh prboard repos discover <org>")
				return nil
			}
		}

		client, err := api.DefaultGraphQLClient()
		if err != nil {
			fmt.Fprintln(os.Stderr, "GitHub auth error. Run: gh auth login")
			return err
		}

		repos, err := ghapi.DiscoverRepos(client, orgs)
		if err != nil {
			return fmt.Errorf("discovering repos: %w", err)
		}

		// Build watched set for display
		watched := map[string]bool{}
		for _, r := range cfg.Repos {
			watched[r.Name] = true
		}

		// Display grouped by org
		reposByOrg := map[string][]ghapi.DiscoveredRepo{}
		for _, r := range repos {
			parts := strings.SplitN(r.FullName, "/", 2)
			org := parts[0]
			reposByOrg[org] = append(reposByOrg[org], r)
		}

		for _, org := range orgs {
			orgRepos := reposByOrg[org]
			fmt.Printf("Repos in %s (%d found):\n\n", org, len(orgRepos))
			for _, r := range orgRepos {
				tag := "[        ]"
				if watched[r.FullName] {
					tag = "[watching]"
				}
				fmt.Printf("  %s  %-45s pushed %s\n", tag, r.FullName, FormatRelativeTime(r.PushedAt))
			}
			fmt.Println()
		}

		fmt.Println("Add/remove repos (prefix with - to remove, 'done' to finish):")
		for {
			input, err := PromptLine("> ")
			if err != nil {
				break
			}
			if input == "done" {
				break
			}

			if strings.HasPrefix(input, "-") {
				name := strings.TrimSpace(strings.TrimPrefix(input, "-"))
				if cfg.RemoveRepo(name) {
					fmt.Printf("  Removed %s\n", name)
				} else {
					fmt.Printf("  %s was not being watched\n", name)
				}
			} else {
				cfg.AddRepo(input, "")
				fmt.Printf("  Added %s\n", input)
			}
		}

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		fmt.Println("Config updated.")
		return nil
	},
}

func init() {
	reposCmd.AddCommand(reposDiscoverCmd)
}
