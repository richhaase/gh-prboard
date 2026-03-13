package github

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
)

type DiscoveredRepo struct {
	FullName string
	PushedAt time.Time
}

// DiscoverRepos fetches all non-archived repos from the given orgs, sorted by most recently pushed.
func DiscoverRepos(client *api.GraphQLClient, orgs []string) ([]DiscoveredRepo, error) {
	var allRepos []DiscoveredRepo

	for _, org := range orgs {
		repos, err := discoverOrgRepos(client, org)
		if err != nil {
			return nil, fmt.Errorf("discovering repos for %s: %w", org, err)
		}
		allRepos = append(allRepos, repos...)
	}

	return allRepos, nil
}

func discoverOrgRepos(client *api.GraphQLClient, org string) ([]DiscoveredRepo, error) {
	var allRepos []DiscoveredRepo
	hasNextPage := true
	cursor := ""

	for hasNextPage {
		query := `query DiscoverRepos($org: String!, $cursor: String) {
			organization(login: $org) {
				repositories(first: 100, after: $cursor, isArchived: false, orderBy: {field: PUSHED_AT, direction: DESC}) {
					pageInfo {
						hasNextPage
						endCursor
					}
					nodes {
						nameWithOwner
						pushedAt
					}
				}
			}
		}`

		variables := map[string]interface{}{
			"org": org,
		}
		if cursor != "" {
			variables["cursor"] = cursor
		}

		var result json.RawMessage
		if err := client.Do(query, variables, &result); err != nil {
			return nil, err
		}

		var parsed struct {
			Organization struct {
				Repositories struct {
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
					Nodes []struct {
						NameWithOwner string    `json:"nameWithOwner"`
						PushedAt      time.Time `json:"pushedAt"`
					} `json:"nodes"`
				} `json:"repositories"`
			} `json:"organization"`
		}

		if err := json.Unmarshal(result, &parsed); err != nil {
			return nil, fmt.Errorf("parsing response: %w", err)
		}

		for _, node := range parsed.Organization.Repositories.Nodes {
			allRepos = append(allRepos, DiscoveredRepo{
				FullName: node.NameWithOwner,
				PushedAt: node.PushedAt,
			})
		}

		hasNextPage = parsed.Organization.Repositories.PageInfo.HasNextPage
		cursor = parsed.Organization.Repositories.PageInfo.EndCursor
	}

	return allRepos, nil
}

// DiscoverUserRepos fetches the authenticated user's own non-archived repos, sorted by most recently pushed.
func DiscoverUserRepos(client *api.GraphQLClient) ([]DiscoveredRepo, error) {
	var allRepos []DiscoveredRepo
	hasNextPage := true
	cursor := ""

	for hasNextPage {
		query := `query DiscoverUserRepos($cursor: String) {
			viewer {
				repositories(first: 100, after: $cursor, isArchived: false, ownerAffiliations: [OWNER], orderBy: {field: PUSHED_AT, direction: DESC}) {
					pageInfo {
						hasNextPage
						endCursor
					}
					nodes {
						nameWithOwner
						pushedAt
					}
				}
			}
		}`

		variables := map[string]interface{}{}
		if cursor != "" {
			variables["cursor"] = cursor
		}

		var result json.RawMessage
		if err := client.Do(query, variables, &result); err != nil {
			return nil, fmt.Errorf("fetching user repos: %w", err)
		}

		var parsed struct {
			Viewer struct {
				Repositories struct {
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
					Nodes []struct {
						NameWithOwner string    `json:"nameWithOwner"`
						PushedAt      time.Time `json:"pushedAt"`
					} `json:"nodes"`
				} `json:"repositories"`
			} `json:"viewer"`
		}

		if err := json.Unmarshal(result, &parsed); err != nil {
			return nil, fmt.Errorf("parsing response: %w", err)
		}

		for _, node := range parsed.Viewer.Repositories.Nodes {
			allRepos = append(allRepos, DiscoveredRepo{
				FullName: node.NameWithOwner,
				PushedAt: node.PushedAt,
			})
		}

		hasNextPage = parsed.Viewer.Repositories.PageInfo.HasNextPage
		cursor = parsed.Viewer.Repositories.PageInfo.EndCursor
	}

	return allRepos, nil
}

// FilterByAge returns repos that were pushed within the given duration.
func FilterByAge(repos []DiscoveredRepo, maxAge time.Duration) []DiscoveredRepo {
	var filtered []DiscoveredRepo
	cutoff := time.Now().Add(-maxAge)
	for _, r := range repos {
		if r.PushedAt.After(cutoff) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}
