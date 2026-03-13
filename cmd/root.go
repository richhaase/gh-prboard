package cmd

import (
	"fmt"
	"os"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/spf13/cobra"

	"github.com/rdh/gh-prboard/config"
	ghapi "github.com/rdh/gh-prboard/github"
	"github.com/rdh/gh-prboard/render"
)

var (
	flagGroup         string
	flagAuthor        string
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
		fmt.Fprintln(os.Stderr, "No repos configured. Get started with:")
		fmt.Fprintln(os.Stderr, "  gh prboard repos add owner/repo")
		fmt.Fprintln(os.Stderr, "  gh prboard repos discover")
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

	prs, err := ghapi.FetchPRs(client, repoNames)
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
		username, err = resolveUsername(client)
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
		if flagNeedsReview {
			status := pr.ReviewStatus()
			if status != ghapi.ReviewNone && status != ghapi.ReviewReReviewNeeded && status != ghapi.ReviewChangesRequested {
				continue
			}
		}
		filtered = append(filtered, pr)
	}
	return filtered
}

func resolveUsername(client *api.GraphQLClient) (string, error) {
	var query struct {
		Viewer struct {
			Login string
		}
	}
	err := client.Query("CurrentUser", &query, nil)
	if err != nil {
		return "", err
	}
	return query.Viewer.Login, nil
}
