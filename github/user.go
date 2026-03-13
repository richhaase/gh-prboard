package github

import (
	"encoding/json"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"
)

// FetchUsername returns the authenticated user's GitHub login.
func FetchUsername(client *api.GraphQLClient) (string, error) {
	var query struct {
		Viewer struct {
			Login string
		}
	}
	err := client.Query("CurrentUser", &query, nil)
	if err != nil {
		return "", fmt.Errorf("fetching username: %w", err)
	}
	return query.Viewer.Login, nil
}

// FetchOrgs returns the login names of organizations the authenticated user belongs to.
func FetchOrgs(client *api.GraphQLClient) ([]string, error) {
	q := `query UserOrgs {
		viewer {
			organizations(first: 100) {
				nodes {
					login
				}
			}
		}
	}`

	var result json.RawMessage
	if err := client.Do(q, nil, &result); err != nil {
		return nil, fmt.Errorf("fetching orgs: %w", err)
	}

	var parsed struct {
		Viewer struct {
			Organizations struct {
				Nodes []struct {
					Login string `json:"login"`
				} `json:"nodes"`
			} `json:"organizations"`
		} `json:"viewer"`
	}

	if err := json.Unmarshal(result, &parsed); err != nil {
		return nil, fmt.Errorf("parsing orgs response: %w", err)
	}

	orgs := make([]string, len(parsed.Viewer.Organizations.Nodes))
	for i, node := range parsed.Viewer.Organizations.Nodes {
		orgs[i] = node.Login
	}
	return orgs, nil
}
