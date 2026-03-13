package github

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
)

type ReviewStatusType int

const (
	ReviewNone             ReviewStatusType = iota // no reviews yet
	ReviewReReviewNeeded                           // reviewed, but new commits since
	ReviewChangesRequested                         // reviewer requested changes
	ReviewApproved                                 // approved
)

func (r ReviewStatusType) Priority() int {
	return int(r)
}

func (r ReviewStatusType) String() string {
	switch r {
	case ReviewNone:
		return "needs review"
	case ReviewReReviewNeeded:
		return "re-review needed"
	case ReviewChangesRequested:
		return "changes requested"
	case ReviewApproved:
		return "approved"
	default:
		return "unknown"
	}
}

type CheckStatus int

const (
	CheckPending CheckStatus = iota
	CheckPassing
	CheckFailing
)

func (c CheckStatus) String() string {
	switch c {
	case CheckPassing:
		return "passing"
	case CheckFailing:
		return "failing"
	case CheckPending:
		return "pending"
	default:
		return "unknown"
	}
}

type Review struct {
	State       string
	Author      string
	SubmittedAt time.Time
}

type PR struct {
	Repo                 string
	RepoGroup            string
	Number               int
	Title                string
	Author               string
	CreatedAt            time.Time
	IsDraft              bool
	LatestCommitAt       time.Time
	Reviews              []Review
	Checks               CheckStatus
	MergedAt             *time.Time
	ClosedAt             *time.Time
	ReviewRequestedUsers []string
	State                string // "open", "merged", "closed"
}

// ReviewStatus classifies the review state of a PR.
func (pr *PR) ReviewStatus() ReviewStatusType {
	if len(pr.Reviews) == 0 {
		return ReviewNone
	}

	latest := pr.Reviews[0]
	for _, r := range pr.Reviews[1:] {
		if r.SubmittedAt.After(latest.SubmittedAt) {
			latest = r
		}
	}

	if pr.LatestCommitAt.After(latest.SubmittedAt) {
		return ReviewReReviewNeeded
	}

	switch latest.State {
	case "APPROVED":
		return ReviewApproved
	case "CHANGES_REQUESTED":
		return ReviewChangesRequested
	default:
		return ReviewNone
	}
}

func (pr *PR) Age() time.Duration {
	return time.Since(pr.CreatedAt)
}

func SortByAttention(prs []PR) []PR {
	sorted := make([]PR, len(prs))
	copy(sorted, prs)
	sort.Slice(sorted, func(i, j int) bool {
		pi := sorted[i].ReviewStatus().Priority()
		pj := sorted[j].ReviewStatus().Priority()
		if pi != pj {
			return pi < pj
		}
		return sorted[i].CreatedAt.Before(sorted[j].CreatedAt)
	})
	return sorted
}

// FetchPRs queries GitHub for PRs across the given repos filtered by states.
// repos should be in "owner/name" format. states are GitHub PullRequestState values (e.g. "OPEN", "MERGED", "CLOSED").
func FetchPRs(client *api.GraphQLClient, repos []string, states []string) ([]PR, error) {
	if len(repos) == 0 {
		return nil, nil
	}

	var prs []PR

	batchSize := 25
	for i := 0; i < len(repos); i += batchSize {
		end := i + batchSize
		if end > len(repos) {
			end = len(repos)
		}
		batch, err := fetchPRBatch(client, repos[i:end], states)
		if err != nil {
			return nil, err
		}
		prs = append(prs, batch...)
	}

	return prs, nil
}

func fetchPRBatch(client *api.GraphQLClient, repos []string, states []string) ([]PR, error) {
	var queryParts []string
	var varDecls []string
	variables := map[string]interface{}{}

	varDecls = append(varDecls, "$states: [PullRequestState!]!")
	variables["states"] = states

	for i, repo := range repos {
		parts := strings.SplitN(repo, "/", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid repo format %q, expected owner/name", repo)
		}
		alias := fmt.Sprintf("repo_%d", i)
		varDecls = append(varDecls,
			fmt.Sprintf("$owner_%d: String!, $name_%d: String!", i, i))
		queryParts = append(queryParts, fmt.Sprintf(`
			%s: repository(owner: $owner_%d, name: $name_%d) {
				pullRequests(states: $states, first: 10, orderBy: {field: CREATED_AT, direction: DESC}) {
					nodes {
						number
						title
						isDraft
						createdAt
						mergedAt
						closedAt
						author { login }
						commits(last: 1) {
							nodes {
								commit {
									committedDate
									statusCheckRollup { state }
								}
							}
						}
						reviewRequests(first: 10) {
							nodes {
								requestedReviewer {
									... on User { login }
								}
							}
						}
						latestReviews(last: 10) {
							nodes {
								state
								submittedAt
								author { login }
							}
						}
					}
				}
			}
		`, alias, i, i))
		variables[fmt.Sprintf("owner_%d", i)] = parts[0]
		variables[fmt.Sprintf("name_%d", i)] = parts[1]
	}

	query := fmt.Sprintf("query FetchPRs(%s) { %s }",
		strings.Join(varDecls, ", "),
		strings.Join(queryParts, "\n"))

	var result map[string]json.RawMessage
	err := client.Do(query, variables, &result)
	if err != nil {
		return nil, fmt.Errorf("GraphQL query failed: %w", err)
	}

	var prs []PR
	for i, repo := range repos {
		alias := fmt.Sprintf("repo_%d", i)
		raw, ok := result[alias]
		if !ok {
			continue
		}

		var repoData struct {
			PullRequests struct {
				Nodes []struct {
					Number    int        `json:"number"`
					Title     string     `json:"title"`
					IsDraft   bool       `json:"isDraft"`
					CreatedAt time.Time  `json:"createdAt"`
					MergedAt  *time.Time `json:"mergedAt"`
					ClosedAt  *time.Time `json:"closedAt"`
					Author    struct {
						Login string `json:"login"`
					} `json:"author"`
					Commits struct {
						Nodes []struct {
							Commit struct {
								CommittedDate     time.Time `json:"committedDate"`
								StatusCheckRollup struct {
									State string `json:"state"`
								} `json:"statusCheckRollup"`
							} `json:"commit"`
						} `json:"nodes"`
					} `json:"commits"`
					ReviewRequests struct {
						Nodes []struct {
							RequestedReviewer struct {
								Login string `json:"login"`
							} `json:"requestedReviewer"`
						} `json:"nodes"`
					} `json:"reviewRequests"`
					LatestReviews struct {
						Nodes []struct {
							State       string    `json:"state"`
							SubmittedAt time.Time `json:"submittedAt"`
							Author      struct {
								Login string `json:"login"`
							} `json:"author"`
						} `json:"nodes"`
					} `json:"latestReviews"`
				} `json:"nodes"`
			} `json:"pullRequests"`
		}

		if err := json.Unmarshal(raw, &repoData); err != nil {
			return nil, fmt.Errorf("failed to parse response for %s: %w", repo, err)
		}

		for _, node := range repoData.PullRequests.Nodes {
			pr := PR{
				Repo:      repo,
				Number:    node.Number,
				Title:     node.Title,
				Author:    node.Author.Login,
				CreatedAt: node.CreatedAt,
				IsDraft:   node.IsDraft,
				MergedAt:  node.MergedAt,
				ClosedAt:  node.ClosedAt,
			}

			// Determine state
			if node.MergedAt != nil {
				pr.State = "merged"
			} else if node.ClosedAt != nil {
				pr.State = "closed"
			} else {
				pr.State = "open"
			}

			if len(node.Commits.Nodes) > 0 {
				pr.LatestCommitAt = node.Commits.Nodes[0].Commit.CommittedDate

				switch node.Commits.Nodes[0].Commit.StatusCheckRollup.State {
				case "SUCCESS":
					pr.Checks = CheckPassing
				case "FAILURE", "ERROR":
					pr.Checks = CheckFailing
				default:
					pr.Checks = CheckPending
				}
			}

			for _, rr := range node.ReviewRequests.Nodes {
				if rr.RequestedReviewer.Login != "" {
					pr.ReviewRequestedUsers = append(pr.ReviewRequestedUsers, rr.RequestedReviewer.Login)
				}
			}

			for _, review := range node.LatestReviews.Nodes {
				if review.State == "APPROVED" || review.State == "CHANGES_REQUESTED" {
					pr.Reviews = append(pr.Reviews, Review{
						State:       review.State,
						Author:      review.Author.Login,
						SubmittedAt: review.SubmittedAt,
					})
				}
			}

			prs = append(prs, pr)
		}
	}

	return prs, nil
}
