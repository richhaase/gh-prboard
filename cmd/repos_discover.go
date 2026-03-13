package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/spf13/cobra"

	"github.com/rdh/gh-prboard/config"
	ghapi "github.com/rdh/gh-prboard/github"
)

var reposDiscoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Discover repos from configured orgs",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if len(cfg.Orgs) == 0 {
			fmt.Fprintln(os.Stderr, "No orgs configured. Add an org to your config file:")
			fmt.Fprintf(os.Stderr, "  %s\n", config.DefaultPath())
			fmt.Fprintln(os.Stderr, "\nExample:")
			fmt.Fprintln(os.Stderr, "  orgs:")
			fmt.Fprintln(os.Stderr, "    - your-org")
			return nil
		}

		client, err := api.DefaultGraphQLClient()
		if err != nil {
			fmt.Fprintln(os.Stderr, "GitHub auth error. Run: gh auth login")
			return err
		}

		repos, err := ghapi.DiscoverRepos(client, cfg.Orgs)
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

		for _, org := range cfg.Orgs {
			orgRepos := reposByOrg[org]
			fmt.Printf("Repos in %s (%d found):\n\n", org, len(orgRepos))
			for _, r := range orgRepos {
				tag := "[        ]"
				if watched[r.FullName] {
					tag = "[watching]"
				}
				fmt.Printf("  %s  %-45s pushed %s\n", tag, r.FullName, formatRelativeTime(r.PushedAt))
			}
			fmt.Println()
		}

		fmt.Println("Add/remove repos (prefix with - to remove, 'done' to finish):")
		scanner := bufio.NewScanner(os.Stdin)
		for {
			fmt.Print("> ")
			if !scanner.Scan() {
				break
			}
			input := strings.TrimSpace(scanner.Text())
			if input == "done" {
				break
			}

			if strings.HasPrefix(input, "-") {
				name := strings.TrimPrefix(input, "-")
				name = strings.TrimSpace(name)
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

func formatRelativeTime(t time.Time) string {
	d := time.Since(t)
	hours := int(d.Hours())
	if hours < 1 {
		return "just now"
	}
	if hours < 24 {
		return fmt.Sprintf("%dh ago", hours)
	}
	days := hours / 24
	if days < 7 {
		return fmt.Sprintf("%dd ago", days)
	}
	if days < 30 {
		return fmt.Sprintf("%dw ago", days/7)
	}
	months := days / 30
	return fmt.Sprintf("%dmo ago", months)
}

func init() {
	reposCmd.AddCommand(reposDiscoverCmd)
}
