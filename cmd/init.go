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
const repoDisplayCap = 50              // per org

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

	// Step 2: Discover orgs
	orgs, err := ghapi.FetchOrgs(client)
	if err != nil {
		return fmt.Errorf("fetching orgs: %w", err)
	}

	var selectedOrgs []string
	if len(orgs) == 0 {
		fmt.Println("No organizations found for your account.")
		fmt.Println("You can add repos manually later with: gh prboard repos add owner/repo")
		fmt.Println()
	} else {
		fmt.Printf("Found %d organizations:\n", len(orgs))
		for i, org := range orgs {
			fmt.Printf("  %d. %s\n", i+1, org)
		}
		fmt.Println()

		indices, err := promptSelection("Select orgs to watch (e.g. 1,3 or all): ", len(orgs))
		if err != nil {
			return err
		}

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
		fmt.Println()
	}

	// Step 3: Discover repos (per-org with per-org display cap)
	if len(selectedOrgs) > 0 {
		fmt.Println("Discovering repos...")
		allRepos, err := ghapi.DiscoverRepos(client, selectedOrgs)
		if err != nil {
			return fmt.Errorf("discovering repos: %w", err)
		}

		recentRepos := ghapi.FilterByAge(allRepos, repoMaxAge)

		if len(recentRepos) == 0 {
			fmt.Printf("No repos found with activity in the last 90 days.\n")
			fmt.Println("You can add repos manually: gh prboard repos add owner/repo")
		} else {
			// Group by org, cap per org
			var groupNames []string

			for _, org := range selectedOrgs {
				var orgRepos []ghapi.DiscoveredRepo
				for _, r := range recentRepos {
					parts := strings.SplitN(r.FullName, "/", 2)
					if parts[0] == org {
						orgRepos = append(orgRepos, r)
					}
				}
				if len(orgRepos) == 0 {
					continue
				}

				// Cap per org
				if len(orgRepos) > repoDisplayCap {
					orgRepos = orgRepos[:repoDisplayCap]
				}

				fmt.Printf("\nActive repos in %s (%d found, pushed in last 90 days):\n", org, len(orgRepos))
				for i, r := range orgRepos {
					fmt.Printf("  %d. %-45s pushed %s\n", i+1, r.FullName, FormatRelativeTime(r.PushedAt))
				}
				fmt.Println()

				indices, err := promptSelection("Select repos to watch (e.g. 1-3,5 or all): ", len(orgRepos))
				if err != nil {
					return err
				}

				// Step 4: Group assignment for selected repos
				fmt.Println("\nAssign groups to repos (enter to skip):")
				for _, repoIdx := range indices {
					repo := orgRepos[repoIdx-1]
					prompt := fmt.Sprintf("  %s group", repo.FullName)
					if len(groupNames) > 0 {
						shortcuts := make([]string, len(groupNames))
						for i, g := range groupNames {
							shortcuts[i] = fmt.Sprintf("%d=%s", i+1, g)
						}
						prompt += fmt.Sprintf(" [%s]", strings.Join(shortcuts, ", "))
					}
					prompt += ": "

					groupInput, err := PromptLine(prompt)
					if err != nil {
						return err
					}

					group := ""
					if groupInput != "" {
						// Check if it's a numbered shortcut
						var shortcutNum int
						if n, parseErr := fmt.Sscanf(groupInput, "%d", &shortcutNum); n == 1 && parseErr == nil {
							if shortcutNum >= 1 && shortcutNum <= len(groupNames) {
								group = groupNames[shortcutNum-1]
							} else {
								group = groupInput // treat as literal group name
							}
						} else {
							group = groupInput
						}

						// Track new group names
						isNew := true
						for _, g := range groupNames {
							if g == group {
								isNew = false
								break
							}
						}
						if isNew {
							groupNames = append(groupNames, group)
						}
					}

					cfg.AddRepo(repo.FullName, group)
				}
			}
		}
		fmt.Println()
	}

	// Step 5: Save config
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("Config saved to %s\n", config.DefaultPath())
	fmt.Printf("Watching %d repos across %d orgs.\n\n", len(cfg.Repos), len(cfg.Orgs))

	// Step 6: Best-effort PR fetch
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
