# gh-prwatch Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a GitHub CLI extension that shows open PRs needing attention across watched repositories.

**Architecture:** Go CLI using cobra for commands, go-gh v2 for GitHub GraphQL API access, YAML config for repo watching. Three packages: `cmd/` (CLI), `config/` (YAML read/write), `github/` (API queries).

**Tech Stack:** Go, github.com/cli/go-gh/v2, github.com/spf13/cobra, gopkg.in/yaml.v3

**Spec:** `docs/specs/2026-03-13-gh-prwatch-design.md`

---

## File Structure

```
gh-prwatch/
├── main.go                      # Entry point, calls cmd.Execute()
├── go.mod
├── go.sum
├── cmd/
│   ├── root.go                  # Root command (gh prwatch) — fetches and displays PRs
│   ├── repos.go                 # `repos` parent command
│   ├── repos_list.go            # `repos list` — prints watched repos
│   ├── repos_add.go             # `repos add` — adds a repo to config
│   ├── repos_remove.go          # `repos remove` — removes a repo from config
│   └── repos_discover.go        # `repos discover` — interactive org repo picker
├── config/
│   ├── config.go                # Config struct, Load(), Save(), path resolution
│   └── config_test.go           # Tests for config loading, saving, edge cases
├── github/
│   ├── prs.go                   # FetchPRs GraphQL query, PR struct, review status logic
│   ├── prs_test.go              # Tests for review status classification, sorting, filtering
│   ├── repos.go                 # DiscoverRepos GraphQL query
│   └── repos_test.go            # Tests for repo discovery
├── render/
│   ├── render.go                # Format PRs into terminal output with colors and summary
│   └── render_test.go           # Tests for output formatting
├── .github/
│   └── workflows/
│       └── release.yml          # gh-extension-precompile release workflow
└── docs/
    ├── specs/                   # Design spec (already exists)
    └── plans/                   # This plan
```

Note: `render/` is split from `cmd/` so output formatting is testable without cobra.

---

## Chunk 1: Project Scaffold and Config Package

### Task 1: Initialize Go module and project scaffold

**Files:**
- Create: `main.go`
- Create: `go.mod`
- Create: `cmd/root.go`

- [ ] **Step 1: Initialize Go module**

```bash
cd /Users/rdh/src/gh-prwatch
go mod init github.com/rdh/gh-prwatch
```

- [ ] **Step 2: Add dependencies**

```bash
go get github.com/spf13/cobra
go get github.com/cli/go-gh/v2
go get gopkg.in/yaml.v3
```

- [ ] **Step 3: Write main.go**

```go
package main

import (
	"fmt"
	"os"

	"github.com/rdh/gh-prwatch/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 4: Write cmd/root.go with placeholder**

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "prwatch",
	Short: "Show open PRs needing attention across watched repos",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("gh-prwatch: not yet implemented")
		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}
```

- [ ] **Step 5: Verify it compiles and runs**

```bash
go build -o gh-prwatch . && ./gh-prwatch
```

Expected: prints "gh-prwatch: not yet implemented"

- [ ] **Step 6: Commit**

```bash
git add main.go go.mod go.sum cmd/root.go
git commit -m "feat: initialize go module with cobra scaffold"
```

---

### Task 2: Config package — Load and Save

**Files:**
- Create: `config/config.go`
- Create: `config/config_test.go`

- [ ] **Step 1: Write failing tests for config**

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadNonexistentReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadFrom(filepath.Join(dir, "config.yml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Repos) != 0 {
		t.Errorf("expected empty repos, got %d", len(cfg.Repos))
	}
}

func TestLoadAndSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")

	cfg := &Config{
		Orgs: []string{"teamsense"},
		Repos: []Repo{
			{Name: "teamsense/platform", Group: "backend"},
			{Name: "rdh/personal"},
		},
		Defaults: Defaults{
			HideDrafts: true,
			Username:   "rdh",
		},
	}

	if err := cfg.SaveTo(path); err != nil {
		t.Fatalf("save error: %v", err)
	}

	loaded, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}

	if len(loaded.Repos) != 2 {
		t.Errorf("expected 2 repos, got %d", len(loaded.Repos))
	}
	if loaded.Repos[0].Name != "teamsense/platform" {
		t.Errorf("expected teamsense/platform, got %s", loaded.Repos[0].Name)
	}
	if loaded.Repos[0].Group != "backend" {
		t.Errorf("expected group backend, got %s", loaded.Repos[0].Group)
	}
	if loaded.Defaults.Username != "rdh" {
		t.Errorf("expected username rdh, got %s", loaded.Defaults.Username)
	}
}

func TestAddRepo(t *testing.T) {
	cfg := &Config{}
	cfg.AddRepo("owner/repo", "mygroup")

	if len(cfg.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(cfg.Repos))
	}
	if cfg.Repos[0].Name != "owner/repo" || cfg.Repos[0].Group != "mygroup" {
		t.Errorf("unexpected repo: %+v", cfg.Repos[0])
	}
}

func TestAddRepoDuplicateUpdatesGroup(t *testing.T) {
	cfg := &Config{
		Repos: []Repo{{Name: "owner/repo", Group: "old"}},
	}
	cfg.AddRepo("owner/repo", "new")

	if len(cfg.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(cfg.Repos))
	}
	if cfg.Repos[0].Group != "new" {
		t.Errorf("expected group new, got %s", cfg.Repos[0].Group)
	}
}

func TestAddRepoDuplicateNoGroupIsNoop(t *testing.T) {
	cfg := &Config{
		Repos: []Repo{{Name: "owner/repo", Group: "existing"}},
	}
	cfg.AddRepo("owner/repo", "")

	if cfg.Repos[0].Group != "existing" {
		t.Errorf("expected group to remain existing, got %s", cfg.Repos[0].Group)
	}
}

func TestRemoveRepo(t *testing.T) {
	cfg := &Config{
		Repos: []Repo{
			{Name: "owner/repo1"},
			{Name: "owner/repo2"},
		},
	}
	removed := cfg.RemoveRepo("owner/repo1")

	if !removed {
		t.Error("expected RemoveRepo to return true")
	}
	if len(cfg.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(cfg.Repos))
	}
	if cfg.Repos[0].Name != "owner/repo2" {
		t.Errorf("expected owner/repo2, got %s", cfg.Repos[0].Name)
	}
}

func TestRemoveRepoNotFound(t *testing.T) {
	cfg := &Config{
		Repos: []Repo{{Name: "owner/repo1"}},
	}
	removed := cfg.RemoveRepo("owner/nope")

	if removed {
		t.Error("expected RemoveRepo to return false for missing repo")
	}
}

func TestReposByGroup(t *testing.T) {
	cfg := &Config{
		Repos: []Repo{
			{Name: "a/one", Group: "backend"},
			{Name: "a/two", Group: "frontend"},
			{Name: "a/three", Group: "backend"},
			{Name: "a/four"},
		},
	}

	groups := cfg.ReposByGroup()
	if len(groups["backend"]) != 2 {
		t.Errorf("expected 2 backend repos, got %d", len(groups["backend"]))
	}
	if len(groups["frontend"]) != 1 {
		t.Errorf("expected 1 frontend repo, got %d", len(groups["frontend"]))
	}
	if len(groups[""]) != 1 {
		t.Errorf("expected 1 ungrouped repo, got %d", len(groups[""]))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/rdh/src/gh-prwatch && go test ./config/...
```

Expected: compilation errors (types don't exist yet)

- [ ] **Step 3: Implement config package**

```go
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Repo struct {
	Name  string `yaml:"name"`
	Group string `yaml:"group,omitempty"`
}

type Defaults struct {
	HideDrafts bool   `yaml:"hide_drafts,omitempty"`
	Username   string `yaml:"username,omitempty"`
}

type Config struct {
	Orgs     []string `yaml:"orgs,omitempty"`
	Repos    []Repo   `yaml:"repos,omitempty"`
	Defaults Defaults `yaml:"defaults,omitempty"`
}

// DefaultPath returns the XDG-respecting config file path.
func DefaultPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "gh-prwatch", "config.yml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "gh-prwatch", "config.yml")
}

// LoadFrom reads config from the given path. Returns empty config if file doesn't exist.
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Load reads config from the default path.
func Load() (*Config, error) {
	return LoadFrom(DefaultPath())
}

// SaveTo writes config to the given path, creating directories as needed.
func (c *Config) SaveTo(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Save writes config to the default path.
func (c *Config) Save() error {
	return c.SaveTo(DefaultPath())
}

// AddRepo adds a repo or updates its group if it already exists.
// If the repo exists and group is empty, it's a no-op.
func (c *Config) AddRepo(name, group string) {
	for i, r := range c.Repos {
		if r.Name == name {
			if group != "" {
				c.Repos[i].Group = group
			}
			return
		}
	}
	c.Repos = append(c.Repos, Repo{Name: name, Group: group})
}

// RemoveRepo removes a repo by name. Returns true if it was found and removed.
func (c *Config) RemoveRepo(name string) bool {
	for i, r := range c.Repos {
		if r.Name == name {
			c.Repos = append(c.Repos[:i], c.Repos[i+1:]...)
			return true
		}
	}
	return false
}

// ReposByGroup returns repos organized by group name. Ungrouped repos use "" as key.
func (c *Config) ReposByGroup() map[string][]Repo {
	groups := make(map[string][]Repo)
	for _, r := range c.Repos {
		groups[r.Group] = append(groups[r.Group], r)
	}
	return groups
}

// RepoNames returns a flat list of repo name strings.
func (c *Config) RepoNames() []string {
	names := make([]string, len(c.Repos))
	for i, r := range c.Repos {
		names[i] = r.Name
	}
	return names
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd /Users/rdh/src/gh-prwatch && go test ./config/... -v
```

Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add config/
git commit -m "feat: add config package with load, save, add, remove, and group support"
```

---

## Chunk 2: GitHub Package — PR Fetching and Review Status

### Task 3: PR data types and review status logic

**Files:**
- Create: `github/prs.go`
- Create: `github/prs_test.go`

- [ ] **Step 1: Write failing tests for review status classification**

The review status logic is the core of this tool. Test it thoroughly without needing real API calls.

```go
package github

import (
	"testing"
	"time"
)

func TestClassifyReviewStatus_NoReviews(t *testing.T) {
	pr := &PR{
		Reviews: []Review{},
	}
	status := pr.ReviewStatus()
	if status != ReviewNone {
		t.Errorf("expected ReviewNone, got %v", status)
	}
}

func TestClassifyReviewStatus_Approved(t *testing.T) {
	pr := &PR{
		LatestCommitAt: time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC),
		Reviews: []Review{
			{State: "APPROVED", SubmittedAt: time.Date(2026, 3, 11, 12, 0, 0, 0, time.UTC)},
		},
	}
	status := pr.ReviewStatus()
	if status != ReviewApproved {
		t.Errorf("expected ReviewApproved, got %v", status)
	}
}

func TestClassifyReviewStatus_ChangesRequested(t *testing.T) {
	pr := &PR{
		LatestCommitAt: time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC),
		Reviews: []Review{
			{State: "CHANGES_REQUESTED", SubmittedAt: time.Date(2026, 3, 11, 12, 0, 0, 0, time.UTC)},
		},
	}
	status := pr.ReviewStatus()
	if status != ReviewChangesRequested {
		t.Errorf("expected ReviewChangesRequested, got %v", status)
	}
}

func TestClassifyReviewStatus_ReReviewNeeded(t *testing.T) {
	pr := &PR{
		LatestCommitAt: time.Date(2026, 3, 12, 12, 0, 0, 0, time.UTC),
		Reviews: []Review{
			{State: "APPROVED", SubmittedAt: time.Date(2026, 3, 11, 12, 0, 0, 0, time.UTC)},
		},
	}
	status := pr.ReviewStatus()
	if status != ReviewReReviewNeeded {
		t.Errorf("expected ReviewReReviewNeeded, got %v", status)
	}
}

func TestClassifyReviewStatus_ReReviewAfterChangesRequested(t *testing.T) {
	pr := &PR{
		LatestCommitAt: time.Date(2026, 3, 12, 12, 0, 0, 0, time.UTC),
		Reviews: []Review{
			{State: "CHANGES_REQUESTED", SubmittedAt: time.Date(2026, 3, 11, 12, 0, 0, 0, time.UTC)},
		},
	}
	status := pr.ReviewStatus()
	if status != ReviewReReviewNeeded {
		t.Errorf("expected ReviewReReviewNeeded, got %v", status)
	}
}

func TestClassifyReviewStatus_MultipleReviewsUsesLatest(t *testing.T) {
	pr := &PR{
		LatestCommitAt: time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC),
		Reviews: []Review{
			{State: "CHANGES_REQUESTED", SubmittedAt: time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)},
			{State: "APPROVED", SubmittedAt: time.Date(2026, 3, 11, 12, 0, 0, 0, time.UTC)},
		},
	}
	status := pr.ReviewStatus()
	if status != ReviewApproved {
		t.Errorf("expected ReviewApproved, got %v", status)
	}
}

func TestAttentionPriority(t *testing.T) {
	if ReviewNone.Priority() >= ReviewReReviewNeeded.Priority() {
		t.Error("needs review should have higher priority (lower number) than re-review")
	}
	if ReviewReReviewNeeded.Priority() >= ReviewChangesRequested.Priority() {
		t.Error("re-review should have higher priority than changes requested")
	}
	if ReviewChangesRequested.Priority() >= ReviewApproved.Priority() {
		t.Error("changes requested should have higher priority than approved")
	}
}

func TestSortPRsByAttention(t *testing.T) {
	now := time.Now()
	prs := []PR{
		{Number: 1, CreatedAt: now.Add(-1 * time.Hour), Reviews: []Review{{State: "APPROVED", SubmittedAt: now}}, LatestCommitAt: now.Add(-2 * time.Hour)},
		{Number: 2, CreatedAt: now.Add(-3 * time.Hour), Reviews: []Review{}},
		{Number: 3, CreatedAt: now.Add(-2 * time.Hour), Reviews: []Review{}},
	}

	sorted := SortByAttention(prs)

	// #2 should be first (needs review, oldest)
	if sorted[0].Number != 2 {
		t.Errorf("expected PR #2 first, got #%d", sorted[0].Number)
	}
	// #3 next (needs review, newer)
	if sorted[1].Number != 3 {
		t.Errorf("expected PR #3 second, got #%d", sorted[1].Number)
	}
	// #1 last (approved)
	if sorted[2].Number != 1 {
		t.Errorf("expected PR #1 last, got #%d", sorted[2].Number)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/rdh/src/gh-prwatch && go test ./github/...
```

Expected: compilation errors

- [ ] **Step 3: Implement PR types and review status logic**

```go
package github

import (
	"sort"
	"time"
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
	CheckPending CheckStatus = iota // zero value — safe default for PRs with no CI
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
	SubmittedAt time.Time
}

type PR struct {
	Repo           string
	RepoGroup      string
	Number         int
	Title          string
	Author         string
	CreatedAt      time.Time
	IsDraft        bool
	LatestCommitAt time.Time
	Reviews        []Review
	Checks         CheckStatus
}

// ReviewStatus classifies the review state of a PR.
func (pr *PR) ReviewStatus() ReviewStatusType {
	if len(pr.Reviews) == 0 {
		return ReviewNone
	}

	// Find the most recent review
	latest := pr.Reviews[0]
	for _, r := range pr.Reviews[1:] {
		if r.SubmittedAt.After(latest.SubmittedAt) {
			latest = r
		}
	}

	// If commits are newer than the latest review, it needs re-review
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

// Age returns the duration since the PR was created.
func (pr *PR) Age() time.Duration {
	return time.Since(pr.CreatedAt)
}

// SortByAttention sorts PRs by attention priority (highest first), then by age (oldest first).
// Returns a new slice; does not modify the input.
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
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd /Users/rdh/src/gh-prwatch && go test ./github/... -v
```

Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add github/prs.go github/prs_test.go
git commit -m "feat: add PR types, review status classification, and attention sorting"
```

---

### Task 4: GraphQL query for fetching PRs

**Files:**
- Modify: `github/prs.go` (add FetchPRs function)

This task adds the actual GitHub API call. It uses go-gh's GraphQL client with struct-based queries.

- [ ] **Step 1: Add FetchPRs to github/prs.go**

Update the import block in `github/prs.go` to include:

```go
import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
)
```

Then append the following functions:

```go
// FetchPRs queries GitHub for open PRs across the given repos.
// repos should be in "owner/name" format.
func FetchPRs(client api.GraphQLClient, repos []string) ([]PR, error) {
	if len(repos) == 0 {
		return nil, nil
	}

	var prs []PR

	// Batch repos into groups to stay within GraphQL node limits
	batchSize := 25
	for i := 0; i < len(repos); i += batchSize {
		end := i + batchSize
		if end > len(repos) {
			end = len(repos)
		}
		batch, err := fetchPRBatch(client, repos[i:end])
		if err != nil {
			return nil, err
		}
		prs = append(prs, batch...)
	}

	return prs, nil
}

func fetchPRBatch(client api.GraphQLClient, repos []string) ([]PR, error) {
	// Build a dynamic GraphQL query with aliases for each repo.
	// We use raw query strings because go-gh's struct-based Query() doesn't
	// support dynamic aliases for multiple repos in a single request.
	var queryParts []string
	var varDecls []string
	variables := map[string]interface{}{}

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
				pullRequests(states: OPEN, first: 10, orderBy: {field: CREATED_AT, direction: DESC}) {
					nodes {
						number
						title
						isDraft
						createdAt
						author { login }
						commits(last: 1) {
							nodes {
								commit {
									committedDate
									statusCheckRollup { state }
								}
							}
						}
						latestReviews(last: 10) {
							nodes {
								state
								submittedAt
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
					Number    int       `json:"number"`
					Title     string    `json:"title"`
					IsDraft   bool      `json:"isDraft"`
					CreatedAt time.Time `json:"createdAt"`
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
					LatestReviews struct {
						Nodes []struct {
							State       string    `json:"state"`
							SubmittedAt time.Time `json:"submittedAt"`
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
			}

			// Latest commit timestamp + check status
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

			// Reviews — only consider APPROVED and CHANGES_REQUESTED states.
			// COMMENTED and DISMISSED reviews don't represent a review decision.
			for _, review := range node.LatestReviews.Nodes {
				if review.State == "APPROVED" || review.State == "CHANGES_REQUESTED" {
					pr.Reviews = append(pr.Reviews, Review{
						State:       review.State,
						SubmittedAt: review.SubmittedAt,
					})
				}
			}

			prs = append(prs, pr)
		}
	}

	return prs, nil
}
```

**Implementation note:** The `client.Do(query, variables, &result)` method on go-gh's `GraphQLClient` accepts raw query strings. If `Do` is unavailable in the installed version, fall back to individual `Query()` calls per repo — slower but simpler. Validate the go-gh API at implementation time.

- [ ] **Step 2: Verify it compiles**

```bash
cd /Users/rdh/src/gh-prwatch && go build ./...
```

Expected: compiles cleanly. We can't unit test this without mocking the GraphQL client, so integration testing happens in Task 8.

- [ ] **Step 3: Commit**

```bash
git add github/prs.go
git commit -m "feat: add GraphQL-based FetchPRs for cross-repo PR fetching"
```

---

### Task 5: GraphQL query for repo discovery

**Files:**
- Create: `github/repos.go`

- [ ] **Step 1: Implement DiscoverRepos**

```go
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
func DiscoverRepos(client api.GraphQLClient, orgs []string) ([]DiscoveredRepo, error) {
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

func discoverOrgRepos(client api.GraphQLClient, org string) ([]DiscoveredRepo, error) {
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
```

- [ ] **Step 2: Verify it compiles**

```bash
cd /Users/rdh/src/gh-prwatch && go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add github/repos.go
git commit -m "feat: add DiscoverRepos for org-wide repo discovery"
```

---

## Chunk 3: Rendering and CLI Commands

### Task 6: Output rendering

**Files:**
- Create: `render/render.go`
- Create: `render/render_test.go`

- [ ] **Step 1: Write failing tests for rendering**

```go
package render

import (
	"strings"
	"testing"
	"time"

	"github.com/rdh/gh-prwatch/github"
)

func TestRenderPRs_GroupedOutput(t *testing.T) {
	now := time.Now()
	prs := []github.PR{
		{
			Repo:           "teamsense/platform",
			RepoGroup:      "backend",
			Number:         142,
			Title:          "Add webhook retry logic",
			Author:         "maria",
			CreatedAt:      now.Add(-3 * 24 * time.Hour),
			LatestCommitAt: now.Add(-4 * 24 * time.Hour),
			Reviews:        []github.Review{},
			Checks:         github.CheckPassing,
		},
	}

	output := RenderPRs(prs, 2)

	if !strings.Contains(output, "platform (backend)") {
		t.Errorf("expected group header, got:\n%s", output)
	}
	if !strings.Contains(output, "#142") {
		t.Errorf("expected PR number, got:\n%s", output)
	}
	if !strings.Contains(output, "@maria") {
		t.Errorf("expected author, got:\n%s", output)
	}
	if !strings.Contains(output, "needs review") {
		t.Errorf("expected review status, got:\n%s", output)
	}
}

func TestRenderPRs_SummaryLine(t *testing.T) {
	now := time.Now()
	prs := []github.PR{
		{Repo: "a/one", Number: 1, CreatedAt: now, Reviews: []github.Review{}, Checks: github.CheckPassing},
		{Repo: "a/two", Number: 2, CreatedAt: now, Reviews: []github.Review{}, Checks: github.CheckFailing},
	}

	output := RenderPRs(prs, 2)

	if !strings.Contains(output, "2 PRs need review") {
		t.Errorf("expected summary with count, got:\n%s", output)
	}
	if !strings.Contains(output, "1 has failing CI") {
		t.Errorf("expected CI failure count, got:\n%s", output)
	}
}

func TestRenderPRs_UngroupedReposShowAsOther(t *testing.T) {
	now := time.Now()
	prs := []github.PR{
		{Repo: "rdh/personal", Number: 1, CreatedAt: now, Reviews: []github.Review{}},
	}

	output := RenderPRs(prs, 2)

	if !strings.Contains(output, "personal") {
		t.Errorf("expected repo name in output, got:\n%s", output)
	}
}

func TestRenderPRs_Empty(t *testing.T) {
	output := RenderPRs(nil, 5)
	if !strings.Contains(output, "No open PRs") {
		t.Errorf("expected empty message, got:\n%s", output)
	}
}

func TestRenderPRs_AgeFormatting(t *testing.T) {
	now := time.Now()
	prs := []github.PR{
		{Repo: "a/one", Number: 1, CreatedAt: now.Add(-2 * time.Hour), Reviews: []github.Review{}},
		{Repo: "a/two", Number: 2, CreatedAt: now.Add(-3 * 24 * time.Hour), Reviews: []github.Review{}},
	}

	output := RenderPRs(prs, 2)

	if !strings.Contains(output, "2h") {
		t.Errorf("expected '2h' for 2-hour-old PR, got:\n%s", output)
	}
	if !strings.Contains(output, "3d") {
		t.Errorf("expected '3d' for 3-day-old PR, got:\n%s", output)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/rdh/src/gh-prwatch && go test ./render/...
```

- [ ] **Step 3: Implement render package**

```go
package render

import (
	"fmt"
	"strings"
	"time"

	"github.com/rdh/gh-prwatch/github"
)

// RenderPRs formats a list of PRs into the terminal output format.
// PRs should already be sorted by attention priority.
func RenderPRs(prs []github.PR, repoCount int) string {
	if len(prs) == 0 {
		return fmt.Sprintf("No open PRs across %d watched repos.", repoCount)
	}

	// Group PRs by repo, preserving sort order within each group
	type repoGroup struct {
		header string
		prs    []github.PR
	}
	seen := map[string]int{}
	var groups []repoGroup

	for _, pr := range prs {
		key := pr.Repo + "|" + pr.RepoGroup
		if idx, ok := seen[key]; ok {
			groups[idx].prs = append(groups[idx].prs, pr)
		} else {
			header := repoShortName(pr.Repo)
			if pr.RepoGroup != "" {
				header += fmt.Sprintf(" (%s)", pr.RepoGroup)
			}
			seen[key] = len(groups)
			groups = append(groups, repoGroup{header: header, prs: []github.PR{pr}})
		}
	}

	var b strings.Builder

	for i, g := range groups {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "## %s\n\n", g.header)
		for _, pr := range g.prs {
			fmt.Fprintf(&b, "  #%-5d %-40s @%-10s %-4s %s   %s\n",
				pr.Number,
				truncate(pr.Title, 40),
				pr.Author,
				formatAge(pr.Age()),
				formatReviewStatus(pr.ReviewStatus()),
				formatCheckStatus(pr.Checks),
			)
		}
	}

	// Summary line
	b.WriteString("\n")
	b.WriteString(renderSummary(prs))

	return b.String()
}

func repoShortName(fullName string) string {
	parts := strings.SplitN(fullName, "/", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return fullName
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

func formatAge(d time.Duration) string {
	hours := int(d.Hours())
	if hours < 24 {
		return fmt.Sprintf("%dh", hours)
	}
	days := hours / 24
	if days < 7 {
		return fmt.Sprintf("%dd", days)
	}
	weeks := days / 7
	return fmt.Sprintf("%dw", weeks)
}

func formatReviewStatus(status github.ReviewStatusType) string {
	switch status {
	case github.ReviewNone:
		return "● needs review"
	case github.ReviewReReviewNeeded:
		return "↻ re-review needed"
	case github.ReviewChangesRequested:
		return "⚠ changes requested"
	case github.ReviewApproved:
		return "✓ approved"
	default:
		return "? unknown"
	}
}

func formatCheckStatus(status github.CheckStatus) string {
	switch status {
	case github.CheckPassing:
		return "✓ CI"
	case github.CheckFailing:
		return "✗ CI failing"
	case github.CheckPending:
		return "◌ CI pending"
	default:
		return ""
	}
}

func renderSummary(prs []github.PR) string {
	needsReview := 0
	needsReReview := 0
	failingCI := 0

	for _, pr := range prs {
		switch pr.ReviewStatus() {
		case github.ReviewNone:
			needsReview++
		case github.ReviewReReviewNeeded:
			needsReReview++
		}
		if pr.Checks == github.CheckFailing {
			failingCI++
		}
	}

	var parts []string
	if needsReview > 0 {
		noun := "PRs need"
		if needsReview == 1 {
			noun = "PR needs"
		}
		parts = append(parts, fmt.Sprintf("%d %s review", needsReview, noun))
	}
	if needsReReview > 0 {
		noun := "need"
		if needsReReview == 1 {
			noun = "needs"
		}
		parts = append(parts, fmt.Sprintf("%d %s re-review", needsReReview, noun))
	}
	if failingCI > 0 {
		verb := "have"
		if failingCI == 1 {
			verb = "has"
		}
		parts = append(parts, fmt.Sprintf("%d %s failing CI", failingCI, verb))
	}

	if len(parts) == 0 {
		return "All PRs look good."
	}

	return strings.Join(parts, " · ")
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd /Users/rdh/src/gh-prwatch && go test ./render/... -v
```

Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add render/
git commit -m "feat: add render package for terminal PR output with grouping and summary"
```

---

### Task 7: Wire up CLI commands

**Files:**
- Modify: `cmd/root.go` (implement main PR display)
- Create: `cmd/repos.go`
- Create: `cmd/repos_list.go`
- Create: `cmd/repos_add.go`
- Create: `cmd/repos_remove.go`
- Create: `cmd/repos_discover.go`

- [ ] **Step 1: Implement the root command (main PR display)**

Update `cmd/root.go`:

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/spf13/cobra"

	"github.com/rdh/gh-prwatch/config"
	ghapi "github.com/rdh/gh-prwatch/github"
	"github.com/rdh/gh-prwatch/render"
)

var (
	flagGroup        string
	flagAuthor       string
	flagMine         bool
	flagNeedsReview  bool
	flagIncludeDrafts bool
)

var rootCmd = &cobra.Command{
	Use:   "prwatch",
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
		fmt.Fprintln(os.Stderr, "  gh prwatch repos add owner/repo")
		fmt.Fprintln(os.Stderr, "  gh prwatch repos discover")
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

func resolveUsername(client api.GraphQLClient) (string, error) {
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
```

- [ ] **Step 2: Implement repos parent command**

`cmd/repos.go`:
```go
package cmd

import "github.com/spf13/cobra"

var reposCmd = &cobra.Command{
	Use:   "repos",
	Short: "Manage watched repositories",
}

func init() {
	rootCmd.AddCommand(reposCmd)
}
```

- [ ] **Step 3: Implement repos list**

`cmd/repos_list.go`:
```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rdh/gh-prwatch/config"
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
```

- [ ] **Step 4: Implement repos add**

`cmd/repos_add.go`:
```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rdh/gh-prwatch/config"
)

var addGroupFlag string

var reposAddCmd = &cobra.Command{
	Use:   "add <owner/repo>",
	Short: "Add a repo to watch",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		cfg.AddRepo(args[0], addGroupFlag)

		if err := cfg.Save(); err != nil {
			return err
		}

		fmt.Printf("Watching %s", args[0])
		if addGroupFlag != "" {
			fmt.Printf(" (group: %s)", addGroupFlag)
		}
		fmt.Println()
		return nil
	},
}

func init() {
	reposAddCmd.Flags().StringVar(&addGroupFlag, "group", "", "assign repo to a group")
	reposCmd.AddCommand(reposAddCmd)
}
```

- [ ] **Step 5: Implement repos remove**

`cmd/repos_remove.go`:
```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rdh/gh-prwatch/config"
)

var reposRemoveCmd = &cobra.Command{
	Use:   "remove <owner/repo>",
	Short: "Remove a repo from watch list",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if !cfg.RemoveRepo(args[0]) {
			fmt.Printf("Repo %s was not being watched.\n", args[0])
			return nil
		}

		if err := cfg.Save(); err != nil {
			return err
		}

		fmt.Printf("Stopped watching %s\n", args[0])
		return nil
	},
}

func init() {
	reposCmd.AddCommand(reposRemoveCmd)
}
```

- [ ] **Step 6: Implement repos discover**

`cmd/repos_discover.go`:
```go
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/spf13/cobra"

	"github.com/rdh/gh-prwatch/config"
	ghapi "github.com/rdh/gh-prwatch/github"
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
```

- [ ] **Step 7: Verify everything compiles**

```bash
cd /Users/rdh/src/gh-prwatch && go build ./...
```

- [ ] **Step 8: Commit**

```bash
git add cmd/
git commit -m "feat: wire up all CLI commands — root, repos list/add/remove/discover"
```

---

## Chunk 4: Release Workflow and Integration Testing

### Task 8: Release workflow

**Files:**
- Create: `.github/workflows/release.yml`

- [ ] **Step 1: Create release workflow**

```yaml
name: release

on:
  push:
    tags:
      - "v*"

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: cli/gh-extension-precompile@v2
        with:
          go_version_file: go.mod
```

- [ ] **Step 2: Commit**

```bash
git add .github/
git commit -m "ci: add gh-extension-precompile release workflow"
```

---

### Task 9: Manual integration test

This validates the full flow end-to-end using your real GitHub auth.

- [ ] **Step 1: Build and install locally**

```bash
cd /Users/rdh/src/gh-prwatch
go build -o gh-prwatch .
```

- [ ] **Step 2: Test repos add**

```bash
./gh-prwatch repos add teamsense/ts-platform --group backend
./gh-prwatch repos add teamsense/teamsense --group frontend
./gh-prwatch repos list
```

Expected: lists both repos under their groups.

- [ ] **Step 3: Test main command**

```bash
./gh-prwatch
```

Expected: shows open PRs across both repos with review status, CI status, and summary line.

- [ ] **Step 4: Test filters**

```bash
./gh-prwatch --needs-review
./gh-prwatch --mine
./gh-prwatch --group backend
```

Expected: each flag filters output as expected.

- [ ] **Step 5: Test repos remove**

```bash
./gh-prwatch repos remove teamsense/teamsense
./gh-prwatch repos list
```

Expected: only ts-platform remains.

- [ ] **Step 6: Fix any issues found during integration testing**

Iterate until the tool works end-to-end.

- [ ] **Step 7: Final commit**

```bash
git add -A
git commit -m "fix: integration test fixes"
```

(Only if changes were needed.)

---

### Task 10: Create GitHub repo and initial release

- [ ] **Step 1: Create the GitHub repo**

```bash
cd /Users/rdh/src/gh-prwatch
gh repo create rdh/gh-prwatch --public --source=. --push
```

- [ ] **Step 2: Tag and push for first release**

```bash
git tag v0.1.0
git push --tags
```

- [ ] **Step 3: Verify the extension installs**

Wait for the release workflow to complete, then:

```bash
gh extension install rdh/gh-prwatch
gh prwatch --help
```

Expected: help output shows all commands and flags.
