# Composable Filters Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `--state`, `--since`, and `--reviewer` flags to gh-prboard so users can compose filters to answer standup questions (what merged overnight, what's blocking me, what am I blocking).

**Architecture:** Extend existing GraphQL query to accept variable PR states and return merged/closed timestamps + review request data. Add duration/date parsing utility. Wire new flags into existing filter pipeline. Adjust renderer for non-open PR states.

**Tech Stack:** Go, github.com/cli/go-gh/v2 (GraphQL), github.com/spf13/cobra

**Spec:** `docs/specs/2026-03-13-composable-filters-design.md`

---

## File Structure

```
Modify: cmd/input.go              # Add ParseSince (duration/date → time.Time)
Modify: cmd/input_test.go         # Tests for ParseSince
Modify: cmd/root.go               # New flags, updated filter logic, state→GraphQL mapping
Modify: github/prs.go             # PR struct fields, FetchPRs states param, GraphQL query
Modify: github/prs_test.go        # Tests for new filtering helpers
Modify: render/render.go          # Merged/closed PR rendering
Modify: render/render_test.go     # Tests for merged/closed rendering
```

---

## Chunk 1: Duration/Date Parsing

### Task 1: ParseSince utility

**Files:**
- Modify: `cmd/input.go`
- Modify: `cmd/input_test.go`

- [ ] **Step 1: Write failing tests for ParseSince**

Add to `cmd/input_test.go`:

```go
func TestParseSince_Duration_Days(t *testing.T) {
	cutoff, err := ParseSince("1d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := time.Now().Add(-24 * time.Hour)
	if cutoff.Sub(expected).Abs() > time.Second {
		t.Errorf("expected ~%v, got %v", expected, cutoff)
	}
}

func TestParseSince_Duration_Weeks(t *testing.T) {
	cutoff, err := ParseSince("2w")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := time.Now().Add(-14 * 24 * time.Hour)
	if cutoff.Sub(expected).Abs() > time.Second {
		t.Errorf("expected ~%v, got %v", expected, cutoff)
	}
}

func TestParseSince_Duration_Hours(t *testing.T) {
	cutoff, err := ParseSince("12h")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := time.Now().Add(-12 * time.Hour)
	if cutoff.Sub(expected).Abs() > time.Second {
		t.Errorf("expected ~%v, got %v", expected, cutoff)
	}
}

func TestParseSince_ISODate(t *testing.T) {
	cutoff, err := ParseSince("2026-03-10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := time.Date(2026, 3, 10, 0, 0, 0, 0, time.Local)
	if !cutoff.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, cutoff)
	}
}

func TestParseSince_Invalid(t *testing.T) {
	_, err := ParseSince("banana")
	if err == nil {
		t.Fatal("expected error for invalid input")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/rdh/src/gh-prboard && go test ./cmd/... -run TestParseSince -v`
Expected: FAIL — `ParseSince` not defined

- [ ] **Step 3: Implement ParseSince**

Add to `cmd/input.go`:

```go
// ParseSince parses a duration shorthand (1d, 2w, 12h) or ISO date (2026-03-10)
// into a time.Time cutoff point.
func ParseSince(s string) (time.Time, error) {
	s = strings.TrimSpace(s)

	// Try duration shorthands: Nd, Nw (not supported by time.ParseDuration)
	if len(s) > 1 {
		suffix := s[len(s)-1]
		numStr := s[:len(s)-1]
		if n, err := strconv.Atoi(numStr); err == nil {
			switch suffix {
			case 'd':
				return time.Now().Add(-time.Duration(n) * 24 * time.Hour), nil
			case 'w':
				return time.Now().Add(-time.Duration(n) * 7 * 24 * time.Hour), nil
			}
		}
	}

	// Try Go duration (e.g., 12h, 30m)
	if d, err := time.ParseDuration(s); err == nil {
		return time.Now().Add(-d), nil
	}

	// Try ISO date
	if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("invalid --since value %q: use a duration (1d, 2w, 12h) or date (2026-03-10)", s)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/rdh/src/gh-prboard && go test ./cmd/... -run TestParseSince -v`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
cd /Users/rdh/src/gh-prboard
git add cmd/input.go cmd/input_test.go
git commit -m "feat: add ParseSince utility for duration/date parsing"
```

---

## Chunk 2: PR Struct and GraphQL Query

### Task 2: Extend PR struct with new fields

**Files:**
- Modify: `github/prs.go`

- [ ] **Step 1: Add new fields to PR struct**

In `github/prs.go`, update the `PR` struct to add:

```go
MergedAt             *time.Time
ClosedAt             *time.Time
ReviewRequestedUsers []string
State                string // "open", "merged", "closed"
```

- [ ] **Step 2: Run existing tests to verify nothing breaks**

Run: `cd /Users/rdh/src/gh-prboard && go test ./... -v`
Expected: All existing tests PASS (new fields are zero-valued)

- [ ] **Step 3: Commit**

```bash
cd /Users/rdh/src/gh-prboard
git add github/prs.go
git commit -m "feat: add MergedAt, ClosedAt, ReviewRequestedUsers, State fields to PR struct"
```

### Task 3: Update FetchPRs to accept states parameter

**Files:**
- Modify: `github/prs.go`
- Modify: `cmd/root.go`

- [ ] **Step 1: Update FetchPRs signature and GraphQL query**

Change `FetchPRs` signature to:

```go
func FetchPRs(client *api.GraphQLClient, repos []string, states []string) ([]PR, error)
```

Update `fetchPRBatch` to also accept `states []string`. In the GraphQL query:
- Change `pullRequests(states: OPEN, ...)` to use a `$states` variable: `pullRequests(states: $states, ...)`
- Add `$states: [PullRequestState!]!` to the variable declarations
- Add `mergedAt`, `closedAt` fields to the query
- Add `reviewRequests` block:
  ```graphql
  reviewRequests(first: 10) {
    nodes {
      requestedReviewer {
        ... on User { login }
      }
    }
  }
  ```
- Set `variables["states"] = states`

In the response parsing, populate the new PR fields:
- `pr.State` based on which timestamps are set (if `mergedAt` non-nil → "merged", if `closedAt` non-nil → "closed", else → "open")
- `pr.MergedAt` and `pr.ClosedAt` from the response
- `pr.ReviewRequestedUsers` from `reviewRequests.nodes`

- [ ] **Step 2: Update call site in cmd/root.go**

In `runRoot`, change:
```go
prs, err := ghapi.FetchPRs(client, repoNames)
```
to:
```go
prs, err := ghapi.FetchPRs(client, repoNames, []string{"OPEN"})
```

This preserves current default behavior. (The flag wiring comes in Task 4.)

- [ ] **Step 3: Run all tests**

Run: `cd /Users/rdh/src/gh-prboard && go test ./... -v`
Expected: All PASS

- [ ] **Step 4: Commit**

```bash
cd /Users/rdh/src/gh-prboard
git add github/prs.go cmd/root.go
git commit -m "feat: FetchPRs accepts states parameter, query returns merged/closed/reviewer data"
```

---

## Chunk 3: Flags and Filter Logic

### Task 4: Add new flags and wire filter logic

**Files:**
- Modify: `cmd/root.go`

- [ ] **Step 1: Add flag variables and register flags**

Add to the `var` block at top:
```go
flagState    string
flagSince    string
flagReviewer string
```

In `init()`, add:
```go
rootCmd.Flags().StringVar(&flagState, "state", "open", "PR state: open, merged, closed, all")
rootCmd.Flags().StringVar(&flagSince, "since", "", "filter by recency: duration (1d, 2w) or date (2026-03-10)")
rootCmd.Flags().StringVar(&flagReviewer, "reviewer", "", "filter by reviewer username (use @me for yourself)")
```

- [ ] **Step 2: Map --state to GraphQL states in runRoot**

Before the `FetchPRs` call, add state mapping:

```go
var prStates []string
switch strings.ToLower(flagState) {
case "open":
	prStates = []string{"OPEN"}
case "merged":
	prStates = []string{"MERGED"}
case "closed":
	prStates = []string{"CLOSED"}
case "all":
	prStates = []string{"OPEN", "MERGED", "CLOSED"}
default:
	return fmt.Errorf("invalid --state value %q: use open, merged, closed, or all", flagState)
}
```

Update the FetchPRs call to use `prStates`.

- [ ] **Step 3: Resolve --reviewer @me**

After the existing username resolution block, add:

```go
if flagReviewer == "@me" {
	if username == "" {
		username, err = ghapi.FetchUsername(client)
		if err != nil {
			return fmt.Errorf("could not detect username: %w", err)
		}
	}
	flagReviewer = username
}
```

- [ ] **Step 4: Add --since and --reviewer to filterPRs**

Update `filterPRs` signature to accept `sinceCutoff *time.Time`:

```go
func filterPRs(prs []ghapi.PR, username string, sinceCutoff *time.Time) []ghapi.PR
```

Add filter logic inside the loop:

```go
// --since filter
if sinceCutoff != nil {
	var ts time.Time
	switch {
	case pr.MergedAt != nil:
		ts = *pr.MergedAt
	case pr.ClosedAt != nil:
		ts = *pr.ClosedAt
	default:
		ts = pr.CreatedAt
	}
	if ts.Before(*sinceCutoff) {
		continue
	}
}

// --reviewer filter
if flagReviewer != "" {
	found := false
	for _, u := range pr.ReviewRequestedUsers {
		if strings.EqualFold(u, flagReviewer) {
			found = true
			break
		}
	}
	if !found {
		for _, r := range pr.Reviews {
			if strings.EqualFold(r.Author, flagReviewer) {
				found = true
				break
			}
		}
	}
	if !found {
		continue
	}
}
```

- [ ] **Step 5: Parse --since in runRoot and pass to filterPRs**

Before calling `filterPRs`:

```go
var sinceCutoff *time.Time
if flagSince != "" {
	t, err := ParseSince(flagSince)
	if err != nil {
		return err
	}
	sinceCutoff = &t
}
```

Update the call: `prs = filterPRs(prs, username, sinceCutoff)`

- [ ] **Step 6: Add Author field to Review struct**

In `github/prs.go`, update Review struct:
```go
type Review struct {
	State       string
	Author      string
	SubmittedAt time.Time
}
```

Update the GraphQL query's `latestReviews` block to include `author { login }`.

Update the response parsing to populate `Review.Author`.

- [ ] **Step 7: Run all tests**

Run: `cd /Users/rdh/src/gh-prboard && go test ./... -v`
Expected: All PASS

- [ ] **Step 8: Commit**

```bash
cd /Users/rdh/src/gh-prboard
git add cmd/root.go github/prs.go
git commit -m "feat: add --state, --since, --reviewer flags with composable filtering"
```

---

## Chunk 4: Render Updates

### Task 5: Update renderer for merged/closed PRs

**Files:**
- Modify: `render/render.go`
- Modify: `render/render_test.go`

- [ ] **Step 1: Write failing tests for merged/closed rendering**

Add to `render/render_test.go`:

```go
func TestRenderPRs_MergedState(t *testing.T) {
	now := time.Now()
	mergedAt := now.Add(-2 * time.Hour)
	prs := []github.PR{
		{
			Repo:      "a/one",
			Number:    10,
			Title:     "Add feature",
			Author:    "alice",
			CreatedAt: now.Add(-48 * time.Hour),
			MergedAt:  &mergedAt,
			State:     "merged",
			Reviews:   []github.Review{},
		},
	}

	output := RenderPRs(prs, 1)

	if !strings.Contains(output, "merged") {
		t.Errorf("expected 'merged' in output for merged PR, got:\n%s", output)
	}
	if strings.Contains(output, "needs review") {
		t.Errorf("merged PR should not show review status, got:\n%s", output)
	}
}

func TestRenderPRs_ClosedState(t *testing.T) {
	now := time.Now()
	closedAt := now.Add(-1 * time.Hour)
	prs := []github.PR{
		{
			Repo:      "a/one",
			Number:    11,
			Title:     "Old PR",
			Author:    "bob",
			CreatedAt: now.Add(-72 * time.Hour),
			ClosedAt:  &closedAt,
			State:     "closed",
			Reviews:   []github.Review{},
		},
	}

	output := RenderPRs(prs, 1)

	if !strings.Contains(output, "closed") {
		t.Errorf("expected 'closed' in output for closed PR, got:\n%s", output)
	}
}

func TestRenderPRs_EmptyNonOpen(t *testing.T) {
	output := RenderPRs(nil, 5)
	if !strings.Contains(output, "No") {
		t.Errorf("expected empty message, got:\n%s", output)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/rdh/src/gh-prboard && go test ./render/... -run "TestRenderPRs_Merged|TestRenderPRs_Closed" -v`
Expected: FAIL — merged PR still shows "needs review"

- [ ] **Step 3: Update RenderPRs to handle merged/closed states**

In `render/render.go`, update the PR line rendering inside `RenderPRs`. Replace the current format line with logic that checks `pr.State`:

```go
for _, pr := range g.prs {
	var statusStr string
	switch pr.State {
	case "merged":
		if pr.MergedAt != nil {
			statusStr = green + "✓ merged " + formatAge(time.Since(*pr.MergedAt)) + " ago" + reset
		} else {
			statusStr = green + "✓ merged" + reset
		}
	case "closed":
		if pr.ClosedAt != nil {
			statusStr = red + "✗ closed " + formatAge(time.Since(*pr.ClosedAt)) + " ago" + reset
		} else {
			statusStr = red + "✗ closed" + reset
		}
	default:
		statusStr = fmt.Sprintf("%s  %s", formatReviewStatus(pr.ReviewStatus()), formatCheckStatus(pr.Checks))
	}

	fmt.Fprintf(&b, "  %s#%-5d%s %-40s %s@%-10s%s %s%-4s%s %s\n",
		cyan, pr.Number, reset,
		truncate(pr.Title, 40),
		magenta, pr.Author, reset,
		dim, formatAge(pr.Age()), reset,
		statusStr,
	)
}
```

Also update the empty message in `RenderPRs` to be state-agnostic:
```go
if len(prs) == 0 {
	return fmt.Sprintf("%sNo PRs found across %d watched repos.%s", dim, repoCount, reset)
}
```

- [ ] **Step 4: Run all tests**

Run: `cd /Users/rdh/src/gh-prboard && go test ./... -v`
Expected: All PASS (update `TestRenderPRs_Empty` to match "No PRs found" instead of "No open PRs")

- [ ] **Step 5: Commit**

```bash
cd /Users/rdh/src/gh-prboard
git add render/render.go render/render_test.go
git commit -m "feat: render merged/closed PR states with timestamps"
```

---

## Chunk 5: Manual Verification

### Task 6: End-to-end verification

- [ ] **Step 1: Build and run all tests**

```bash
cd /Users/rdh/src/gh-prboard && make check
```

Expected: All checks pass (fmt, vet, lint, test)

- [ ] **Step 2: Build and test composable commands**

```bash
cd /Users/rdh/src/gh-prboard && make build

# Default behavior unchanged
./bin/gh-prboard

# What merged recently
./bin/gh-prboard --state merged --since 7d

# What am I blocking
./bin/gh-prboard --reviewer @me

# My open PRs
./bin/gh-prboard --mine

# Full week of activity
./bin/gh-prboard --state all --since 7d --mine
```

- [ ] **Step 3: Install**

```bash
cd /Users/rdh/src/gh-prboard && go install .
```

- [ ] **Step 4: Final commit if any fixups needed**

- [ ] **Step 5: Update CLAUDE.md**

Add the new flags to the project description in `CLAUDE.md` under the root.go entry.
