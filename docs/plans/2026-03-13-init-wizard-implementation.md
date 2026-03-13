# Init Wizard Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `gh prboard init` wizard that guides first-time users from zero to a working config with org discovery, repo selection, and grouping.

**Architecture:** New `github/user.go` for user/org GraphQL queries, new `cmd/input.go` for shared input parsing (number ranges, prompts, relative time), new `cmd/init.go` for the wizard command. Refactors `resolveUsername` out of `root.go` and `formatRelativeTime` out of `repos_discover.go` into shared locations.

**Tech Stack:** Go, github.com/cli/go-gh/v2, github.com/spf13/cobra, bufio.Scanner for interactive I/O

**Spec:** `docs/specs/2026-03-13-init-wizard-design.md`

---

## File Structure

```
gh-prwatch/
├── cmd/
│   ├── init.go              # NEW — init wizard command
│   ├── input.go             # NEW — shared input parsing (ParseNumberSelection, PromptLine, FormatRelativeTime)
│   ├── input_test.go        # NEW — tests for input parsing
│   ├── root.go              # MODIFY — update no-config message, replace resolveUsername with github.FetchUsername
│   ├── repos_discover.go    # MODIFY — accept optional org arg, use shared FormatRelativeTime
│   └── ... (unchanged)
├── github/
│   ├── user.go              # NEW — FetchUsername, FetchOrgs
│   ├── user_test.go         # NEW — tests for FetchOrgs (unit test with mock data)
│   ├── repos.go             # MODIFY — add FilterByAge
│   ├── repos_test.go        # NEW — tests for FilterByAge
│   └── ... (unchanged)
└── ... (unchanged)
```

---

## Chunk 1: Shared Utilities (github/user.go, github/repos.go, cmd/input.go)

### Task 1: Input parsing utilities

**Files:**
- Create: `cmd/input.go`
- Create: `cmd/input_test.go`

- [ ] **Step 1: Write failing tests for ParseNumberSelection**

```go
package cmd

import (
	"reflect"
	"testing"
)

func TestParseNumberSelection_Single(t *testing.T) {
	got, err := ParseNumberSelection("3", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []int{3}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseNumberSelection_CommaSeparated(t *testing.T) {
	got, err := ParseNumberSelection("1,3,5", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []int{1, 3, 5}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseNumberSelection_Range(t *testing.T) {
	got, err := ParseNumberSelection("2-4", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []int{2, 3, 4}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseNumberSelection_Mixed(t *testing.T) {
	got, err := ParseNumberSelection("1-3,7,9", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []int{1, 2, 3, 7, 9}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseNumberSelection_SpaceSeparated(t *testing.T) {
	got, err := ParseNumberSelection("1 3 5", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []int{1, 3, 5}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseNumberSelection_All(t *testing.T) {
	got, err := ParseNumberSelection("all", 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []int{1, 2, 3, 4}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseNumberSelection_OutOfRange(t *testing.T) {
	_, err := ParseNumberSelection("6", 5)
	if err == nil {
		t.Error("expected error for out-of-range number")
	}
}

func TestParseNumberSelection_Zero(t *testing.T) {
	_, err := ParseNumberSelection("0", 5)
	if err == nil {
		t.Error("expected error for zero")
	}
}

func TestParseNumberSelection_InvalidText(t *testing.T) {
	_, err := ParseNumberSelection("abc", 5)
	if err == nil {
		t.Error("expected error for invalid input")
	}
}

func TestParseNumberSelection_Deduplicate(t *testing.T) {
	got, err := ParseNumberSelection("1,1,2-3,2", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []int{1, 2, 3}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/rdh/src/gh-prwatch && go test ./cmd/... -run TestParseNumber -v
```

Expected: compilation errors (function doesn't exist yet)

- [ ] **Step 3: Implement input utilities**

```go
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

var scanner *bufio.Scanner

func getScanner() *bufio.Scanner {
	if scanner == nil {
		scanner = bufio.NewScanner(os.Stdin)
	}
	return scanner
}

// PromptLine prints a prompt and reads a single line from stdin.
func PromptLine(prompt string) (string, error) {
	fmt.Print(prompt)
	s := getScanner()
	if !s.Scan() {
		if err := s.Err(); err != nil {
			return "", err
		}
		return "", fmt.Errorf("no input")
	}
	return strings.TrimSpace(s.Text()), nil
}

// ParseNumberSelection parses user input like "1-3,5,7" into a sorted, deduplicated
// slice of integers. Supports single numbers, ranges, "all", and comma/space separators.
// Numbers must be between 1 and max (inclusive).
func ParseNumberSelection(input string, max int) ([]int, error) {
	input = strings.TrimSpace(input)
	if strings.EqualFold(input, "all") {
		result := make([]int, max)
		for i := range result {
			result[i] = i + 1
		}
		return result, nil
	}

	// Normalize separators: replace spaces with commas
	input = strings.ReplaceAll(input, " ", ",")

	seen := map[int]bool{}
	parts := strings.Split(input, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, "-") {
			bounds := strings.SplitN(part, "-", 2)
			lo, err := strconv.Atoi(strings.TrimSpace(bounds[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid number: %s", bounds[0])
			}
			hi, err := strconv.Atoi(strings.TrimSpace(bounds[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid number: %s", bounds[1])
			}
			if lo < 1 || hi > max || lo > hi {
				return nil, fmt.Errorf("range %d-%d is out of bounds (1-%d)", lo, hi, max)
			}
			for i := lo; i <= hi; i++ {
				seen[i] = true
			}
		} else {
			n, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid number: %s", part)
			}
			if n < 1 || n > max {
				return nil, fmt.Errorf("number %d is out of range (1-%d)", n, max)
			}
			seen[n] = true
		}
	}

	if len(seen) == 0 {
		return nil, fmt.Errorf("no numbers selected")
	}

	result := make([]int, 0, len(seen))
	for n := range seen {
		result = append(result, n)
	}
	sort.Ints(result)
	return result, nil
}

// FormatRelativeTime formats a time as a human-readable relative string.
func FormatRelativeTime(t time.Time) string {
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
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd /Users/rdh/src/gh-prwatch && go test ./cmd/... -run TestParseNumber -v
```

Expected: all 10 tests PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/input.go cmd/input_test.go
git commit -m "feat: add shared input parsing utilities (ParseNumberSelection, PromptLine, FormatRelativeTime)"
```

---

### Task 2: GitHub user and org discovery

**Files:**
- Create: `github/user.go`

- [ ] **Step 1: Implement FetchUsername and FetchOrgs**

```go
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
```

- [ ] **Step 2: Verify it compiles**

```bash
cd /Users/rdh/src/gh-prwatch && go build ./...
```

Expected: compiles cleanly

- [ ] **Step 3: Commit**

```bash
git add github/user.go
git commit -m "feat: add FetchUsername and FetchOrgs for user/org discovery"
```

---

### Task 3: FilterByAge for repo discovery

**Files:**
- Modify: `github/repos.go`
- Create: `github/repos_test.go`

- [ ] **Step 1: Write failing tests for FilterByAge**

```go
package github

import (
	"testing"
	"time"
)

func TestFilterByAge_AllRecent(t *testing.T) {
	now := time.Now()
	repos := []DiscoveredRepo{
		{FullName: "org/recent1", PushedAt: now.Add(-1 * 24 * time.Hour)},
		{FullName: "org/recent2", PushedAt: now.Add(-10 * 24 * time.Hour)},
	}
	filtered := FilterByAge(repos, 90*24*time.Hour)
	if len(filtered) != 2 {
		t.Errorf("expected 2 repos, got %d", len(filtered))
	}
}

func TestFilterByAge_SomeOld(t *testing.T) {
	now := time.Now()
	repos := []DiscoveredRepo{
		{FullName: "org/recent", PushedAt: now.Add(-30 * 24 * time.Hour)},
		{FullName: "org/old", PushedAt: now.Add(-120 * 24 * time.Hour)},
	}
	filtered := FilterByAge(repos, 90*24*time.Hour)
	if len(filtered) != 1 {
		t.Errorf("expected 1 repo, got %d", len(filtered))
	}
	if filtered[0].FullName != "org/recent" {
		t.Errorf("expected org/recent, got %s", filtered[0].FullName)
	}
}

func TestFilterByAge_AllOld(t *testing.T) {
	now := time.Now()
	repos := []DiscoveredRepo{
		{FullName: "org/old1", PushedAt: now.Add(-100 * 24 * time.Hour)},
		{FullName: "org/old2", PushedAt: now.Add(-200 * 24 * time.Hour)},
	}
	filtered := FilterByAge(repos, 90*24*time.Hour)
	if len(filtered) != 0 {
		t.Errorf("expected 0 repos, got %d", len(filtered))
	}
}

func TestFilterByAge_Empty(t *testing.T) {
	filtered := FilterByAge(nil, 90*24*time.Hour)
	if len(filtered) != 0 {
		t.Errorf("expected 0 repos, got %d", len(filtered))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/rdh/src/gh-prwatch && go test ./github/... -run TestFilterByAge -v
```

Expected: compilation error (FilterByAge doesn't exist)

- [ ] **Step 3: Implement FilterByAge**

Append to `github/repos.go`:

```go
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
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd /Users/rdh/src/gh-prwatch && go test ./github/... -run TestFilterByAge -v
```

Expected: all 4 tests PASS

- [ ] **Step 5: Commit**

```bash
git add github/repos.go github/repos_test.go
git commit -m "feat: add FilterByAge for time-based repo filtering"
```

---

## Chunk 2: Refactor Existing Code

### Task 4: Replace resolveUsername in root.go with github.FetchUsername

**Files:**
- Modify: `cmd/root.go`

- [ ] **Step 1: Update root.go imports and replace resolveUsername**

In `cmd/root.go`, change the `--mine` username resolution (around line 100-105) from:

```go
	if flagMine && username == "" {
		username, err = resolveUsername(client)
		if err != nil {
			return fmt.Errorf("could not detect username: %w", err)
		}
	}
```

to:

```go
	if flagMine && username == "" {
		username, err = ghapi.FetchUsername(client)
		if err != nil {
			return fmt.Errorf("could not detect username: %w", err)
		}
	}
```

Then **delete** the entire `resolveUsername` function (lines 140-151).

- [ ] **Step 2: Update the no-config message**

Change the no-config block (around lines 47-52) from:

```go
	if len(cfg.Repos) == 0 {
		fmt.Fprintln(os.Stderr, "No repos configured. Get started with:")
		fmt.Fprintln(os.Stderr, "  gh prboard repos add owner/repo")
		fmt.Fprintln(os.Stderr, "  gh prboard repos discover")
		return nil
	}
```

to:

```go
	if len(cfg.Repos) == 0 {
		fmt.Fprintln(os.Stderr, "No repos configured. Run `gh prboard init` to get started.")
		return nil
	}
```

- [ ] **Step 3: Remove unused api import**

The `"github.com/cli/go-gh/v2/pkg/api"` import is still needed for `api.DefaultGraphQLClient()`, so keep it. But remove it from the import if the compiler says it's unused.

- [ ] **Step 4: Verify build and tests pass**

```bash
cd /Users/rdh/src/gh-prwatch && go build ./... && go test ./... -v
```

Expected: compiles cleanly, all existing tests still pass

- [ ] **Step 5: Commit**

```bash
git add cmd/root.go
git commit -m "refactor: replace resolveUsername with github.FetchUsername, update no-config message"
```

---

### Task 5: Refactor repos_discover.go to use shared utilities and accept org arg

**Files:**
- Modify: `cmd/repos_discover.go`

- [ ] **Step 1: Update repos_discover.go**

Replace the entire file with:

```go
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
```

Key changes:
- Uses `cobra.MaximumNArgs(1)` to accept optional org arg
- Uses shared `FormatRelativeTime` and `PromptLine` from `cmd/input.go`
- Removes the local `formatRelativeTime` function
- Points users to `init` when no orgs configured

- [ ] **Step 2: Verify build and tests pass**

```bash
cd /Users/rdh/src/gh-prwatch && go build ./... && go test ./...
```

Expected: compiles cleanly, all tests pass

- [ ] **Step 3: Commit**

```bash
git add cmd/repos_discover.go
git commit -m "refactor: repos discover accepts org arg, uses shared input utilities"
```

---

## Chunk 3: Init Wizard Command

### Task 6: Implement the init wizard

**Files:**
- Create: `cmd/init.go`

- [ ] **Step 1: Write cmd/init.go**

```go
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
```

- [ ] **Step 2: Verify it compiles**

```bash
cd /Users/rdh/src/gh-prwatch && go build ./...
```

Expected: compiles cleanly

- [ ] **Step 3: Run all tests**

```bash
cd /Users/rdh/src/gh-prwatch && go test ./... -v
```

Expected: all tests pass (existing + new input tests)

- [ ] **Step 4: Commit**

```bash
git add cmd/init.go
git commit -m "feat: add init wizard for guided first-time setup"
```

---

## Chunk 4: Integration Testing

### Task 7: Manual integration test

- [ ] **Step 1: Build**

```bash
cd /Users/rdh/src/gh-prwatch && go build -o gh-prboard .
```

- [ ] **Step 2: Test init with no existing config**

Remove any existing config first:

```bash
rm -f ~/.config/gh-prboard/config.yml
./gh-prboard init
```

Expected: wizard walks through username detection, org discovery, repo selection, grouping, saves config, shows PRs.

- [ ] **Step 3: Test init re-run (merge)**

```bash
./gh-prboard init
```

Expected: prompts overwrite/merge/cancel. Test "m" for merge.

- [ ] **Step 4: Test root command with config**

```bash
./gh-prboard
```

Expected: shows PRs from configured repos.

- [ ] **Step 5: Test root command with no config**

```bash
rm -f ~/.config/gh-prboard/config.yml
./gh-prboard
```

Expected: prints `No repos configured. Run 'gh prboard init' to get started.`

- [ ] **Step 6: Test repos discover with org arg**

```bash
./gh-prboard repos discover teamsense
```

Expected: discovers repos for teamsense without needing config.

- [ ] **Step 7: Test repos discover without arg and no config**

```bash
rm -f ~/.config/gh-prboard/config.yml
./gh-prboard repos discover
```

Expected: prints message pointing to `init` or `discover <org>`.

- [ ] **Step 8: Fix any issues found during integration testing**

Iterate until everything works end-to-end.

- [ ] **Step 9: Final commit (only if fixes were needed)**

```bash
git add -A
git commit -m "fix: integration test fixes for init wizard"
```

- [ ] **Step 10: Push**

```bash
git push
```
