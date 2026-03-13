package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/spf13/cobra"

	"github.com/richhaase/gh-prboard/config"
	ghapi "github.com/richhaase/gh-prboard/github"
	"github.com/richhaase/gh-prboard/render"
)

var (
	flagGroup         string
	flagAuthor        string
	flagRepo          string
	flagStatus        string
	flagCI            string
	flagMine          bool
	flagNeedsReview   bool
	flagIncludeDrafts bool
)

var rootCmd = &cobra.Command{
	Use:   "prboard",
	Short: "Show open PRs needing attention across watched repos",
	RunE:  runRoot,
}

func init() {
	rootCmd.Flags().StringVar(&flagGroup, "group", "", "filter to a specific repo group")
	rootCmd.Flags().StringVar(&flagAuthor, "author", "", "filter by PR author")
	rootCmd.Flags().StringVar(&flagRepo, "repo", "", "filter by repo name (substring match)")
	rootCmd.Flags().StringVar(&flagStatus, "status", "", "filter by review status (needs-review, approved, changes-requested, re-review)")
	rootCmd.Flags().StringVar(&flagCI, "ci", "", "filter by CI status (passing, failing, pending)")
	rootCmd.Flags().BoolVar(&flagMine, "mine", false, "show only PRs you authored")
	rootCmd.Flags().BoolVar(&flagNeedsReview, "needs-review", false, "show only PRs needing review")
	rootCmd.Flags().BoolVar(&flagIncludeDrafts, "include-drafts", false, "include draft PRs")
}

func Execute() error {
	return rootCmd.Execute()
}

func runRoot(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if len(cfg.Repos) == 0 {
		fmt.Fprintln(os.Stderr, "No repos configured. Run `gh prboard init` to get started.")
		return nil
	}

	// Filter repos by group if specified
	repos := cfg.Repos
	if flagGroup != "" {
		var filtered []config.Repo
		for _, r := range repos {
			if r.Group == flagGroup {
				filtered = append(filtered, r)
			}
		}
		repos = filtered
		if len(filtered) == 0 {
			fmt.Fprintf(os.Stderr, "No repos in group %q.\n", flagGroup)
			return nil
		}
	}

	// Build repo name list
	repoNames := make([]string, len(repos))
	for i, r := range repos {
		repoNames[i] = r.Name
	}

	// Build repo→group lookup
	groupLookup := map[string]string{}
	for _, r := range repos {
		groupLookup[r.Name] = r.Group
	}

	client, err := api.DefaultGraphQLClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, "GitHub auth error. Run: gh auth login")
		return err
	}

	prs, err := ghapi.FetchPRs(client, repoNames, []string{"OPEN"})
	if err != nil {
		return fmt.Errorf("fetching PRs: %w", err)
	}

	// Attach group info
	for i := range prs {
		prs[i].RepoGroup = groupLookup[prs[i].Repo]
	}

	// Resolve username for --mine
	username := cfg.Defaults.Username
	if flagMine && username == "" {
		username, err = ghapi.FetchUsername(client)
		if err != nil {
			return fmt.Errorf("could not detect username: %w", err)
		}
	}

	// Apply filters
	prs = filterPRs(prs, username)

	// Sort
	prs = ghapi.SortByAttention(prs)

	fmt.Print(render.RenderPRs(prs, len(repoNames)))
	return nil
}

func filterPRs(prs []ghapi.PR, username string) []ghapi.PR {
	var filtered []ghapi.PR
	for _, pr := range prs {
		if !flagIncludeDrafts && pr.IsDraft {
			continue
		}
		if flagMine && pr.Author != username {
			continue
		}
		if flagAuthor != "" && pr.Author != flagAuthor {
			continue
		}
		if flagRepo != "" && !strings.Contains(strings.ToLower(pr.Repo), strings.ToLower(flagRepo)) {
			continue
		}
		if flagNeedsReview {
			status := pr.ReviewStatus()
			if status != ghapi.ReviewNone && status != ghapi.ReviewReReviewNeeded && status != ghapi.ReviewChangesRequested {
				continue
			}
		}
		if flagStatus != "" && !matchReviewStatus(pr.ReviewStatus(), flagStatus) {
			continue
		}
		if flagCI != "" && !matchCheckStatus(pr.Checks, flagCI) {
			continue
		}
		filtered = append(filtered, pr)
	}
	return filtered
}

func matchReviewStatus(status ghapi.ReviewStatusType, filter string) bool {
	switch strings.ToLower(filter) {
	case "needs-review":
		return status == ghapi.ReviewNone
	case "re-review":
		return status == ghapi.ReviewReReviewNeeded
	case "changes-requested":
		return status == ghapi.ReviewChangesRequested
	case "approved":
		return status == ghapi.ReviewApproved
	default:
		return false
	}
}

func matchCheckStatus(status ghapi.CheckStatus, filter string) bool {
	switch strings.ToLower(filter) {
	case "passing":
		return status == ghapi.CheckPassing
	case "failing":
		return status == ghapi.CheckFailing
	case "pending":
		return status == ghapi.CheckPending
	default:
		return false
	}
}
