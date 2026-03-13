package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/spf13/cobra"

	"github.com/richhaase/gh-prboard/config"
	ghapi "github.com/richhaase/gh-prboard/github"
	"github.com/richhaase/gh-prboard/render"
)

const repoMaxAge = 90 * 24 * time.Hour // 90 days
const repoDisplayCap = 50              // per source

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive setup wizard for first-time configuration",
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

// promptSelection prompts the user with re-prompting on invalid input.
func promptSelection(prompt string, max int) ([]int, error) {
	for {
		input, err := PromptLine(prompt)
		if err != nil {
			return nil, err
		}
		indices, err := ParseNumberSelection(input, max)
		if err != nil {
			fmt.Printf("Invalid selection: %v. Try again.\n", err)
			continue
		}
		return indices, nil
	}
}

// selectRepos displays a list of repos and lets the user pick which to watch.
func selectRepos(cfg *config.Config, repos []ghapi.DiscoveredRepo, label string) error {
	if len(repos) > repoDisplayCap {
		repos = repos[:repoDisplayCap]
	}

	fmt.Printf("\nActive repos in %s (%d found, pushed in last 90 days):\n", label, len(repos))
	for i, r := range repos {
		fmt.Printf("  %d. %-45s pushed %s\n", i+1, r.FullName, FormatRelativeTime(r.PushedAt))
	}
	fmt.Println()

	indices, err := promptSelection("Select repos to watch (e.g. 1-3,5 or all): ", len(repos))
	if err != nil {
		return err
	}

	for _, repoIdx := range indices {
		cfg.AddRepo(repos[repoIdx-1].FullName, "")
	}

	return nil
}

func runInit(cmd *cobra.Command, args []string) error {
	// Check for existing config
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if len(cfg.Repos) > 0 || len(cfg.Orgs) > 0 {
		choice, err := PromptLine(fmt.Sprintf(
			"Config already exists at %s\nOverwrite, merge, or cancel? [o/m/c]: ",
			config.DefaultPath()))
		if err != nil {
			return err
		}
		switch strings.ToLower(strings.TrimSpace(choice)) {
		case "o", "overwrite":
			cfg = &config.Config{}
		case "m", "merge":
			// keep existing cfg, will add to it
		default:
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Step 1: Auth + username
	client, err := api.DefaultGraphQLClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Not authenticated. Run `gh auth login` first.")
		return err
	}

	username, err := ghapi.FetchUsername(client)
	if err != nil {
		return fmt.Errorf("fetching username: %w", err)
	}
	fmt.Printf("Detected GitHub user: %s\n\n", username)
	cfg.Defaults.Username = username
	cfg.Defaults.HideDrafts = true

	// Step 2: Discover personal repos
	fmt.Println("Discovering your repos...")
	userRepos, err := ghapi.DiscoverUserRepos(client)
	if err != nil {
		return fmt.Errorf("discovering repos: %w", err)
	}

	recentUserRepos := ghapi.FilterByAge(userRepos, repoMaxAge)
	if len(recentUserRepos) == 0 {
		fmt.Println("No personal repos found with activity in the last 90 days.")
	} else {
		if err := selectRepos(cfg, recentUserRepos, username); err != nil {
			return err
		}
	}
	fmt.Println()

	// Step 3: Optionally discover org repos
	orgs, err := ghapi.FetchOrgs(client)
	if err != nil {
		return fmt.Errorf("fetching orgs: %w", err)
	}

	if len(orgs) > 0 {
		fmt.Printf("Found %d organizations:\n", len(orgs))
		for i, org := range orgs {
			fmt.Printf("  %d. %s\n", i+1, org)
		}
		fmt.Println()

		input, err := PromptLine("Select orgs to include (e.g. 1,3 or all), or enter to skip: ")
		if err != nil {
			return err
		}

		if input != "" {
			indices, err := ParseNumberSelection(input, len(orgs))
			if err != nil {
				fmt.Printf("Invalid selection: %v. Skipping orgs.\n", err)
			} else {
				var selectedOrgs []string
				for _, idx := range indices {
					selectedOrgs = append(selectedOrgs, orgs[idx-1])
				}

				// Merge orgs into config
				existingOrgs := map[string]bool{}
				for _, o := range cfg.Orgs {
					existingOrgs[o] = true
				}
				for _, o := range selectedOrgs {
					if !existingOrgs[o] {
						cfg.Orgs = append(cfg.Orgs, o)
					}
				}

				// Discover org repos
				fmt.Println("\nDiscovering org repos...")
				orgRepos, err := ghapi.DiscoverRepos(client, selectedOrgs)
				if err != nil {
					return fmt.Errorf("discovering repos: %w", err)
				}

				recentOrgRepos := ghapi.FilterByAge(orgRepos, repoMaxAge)
				for _, org := range selectedOrgs {
					var thisOrgRepos []ghapi.DiscoveredRepo
					for _, r := range recentOrgRepos {
						parts := strings.SplitN(r.FullName, "/", 2)
						if parts[0] == org {
							thisOrgRepos = append(thisOrgRepos, r)
						}
					}
					if len(thisOrgRepos) == 0 {
						fmt.Printf("\nNo active repos found in %s.\n", org)
						continue
					}
					if err := selectRepos(cfg, thisOrgRepos, org); err != nil {
						return err
					}
				}
			}
		}
		fmt.Println()
	}

	// Step 4: Save config
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("Config saved to %s\n", config.DefaultPath())
	fmt.Printf("Watching %d repos", len(cfg.Repos))
	if len(cfg.Orgs) > 0 {
		fmt.Printf(" across %d orgs", len(cfg.Orgs))
	}
	fmt.Println(".")
	fmt.Println()

	// Step 5: Best-effort PR fetch
	if len(cfg.Repos) == 0 {
		return nil
	}

	fmt.Println("Fetching PRs...")
	fmt.Println()

	repoNames := cfg.RepoNames()
	groupLookup := map[string]string{}
	for _, r := range cfg.Repos {
		groupLookup[r.Name] = r.Group
	}

	prs, err := ghapi.FetchPRs(client, repoNames)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not fetch PRs: %v\n", err)
		fmt.Fprintln(os.Stderr, "Config was saved successfully. Run `gh prboard` to try again.")
		return nil
	}

	for i := range prs {
		prs[i].RepoGroup = groupLookup[prs[i].Repo]
	}

	// Filter drafts by default
	var filtered []ghapi.PR
	for _, pr := range prs {
		if !pr.IsDraft {
			filtered = append(filtered, pr)
		}
	}

	filtered = ghapi.SortByAttention(filtered)
	fmt.Println(render.RenderPRs(filtered, len(repoNames)))
	return nil
}
