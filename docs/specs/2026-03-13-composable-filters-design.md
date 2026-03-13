# Composable Filters: --state, --since, --reviewer

## Problem

The board only shows open PRs. To answer "what happened overnight?" or "what am I blocking?" requires different views of the same data — merged PRs, PRs where you're a reviewer, etc. Rather than a separate subcommand, extend the existing board with composable filters.

## New Flags

### `--state <open|merged|closed|all>`
- Default: `open` (current behavior unchanged)
- Passed through to GraphQL query as PR states

### `--since <duration|date>`
- No default (omitting shows all)
- Accepts durations: `1d`, `7d`, `24h`, `2w` (parsed as Go durations with d/w shorthand)
- Accepts ISO dates: `2026-03-12`
- Filters on the appropriate timestamp per state:
  - open → `createdAt`
  - merged → `mergedAt`
  - closed → `closedAt`
  - all → whichever is most recent of the above

### `--reviewer <username|@me>`
- Filters to PRs where the user has a pending review request OR has submitted a review
- `@me` resolves to authenticated user (same as `--mine`)

## Changes

### `github/prs.go`
- `FetchPRs` takes a `states []string` parameter instead of hardcoding `OPEN`
- GraphQL query adds: `mergedAt`, `closedAt`, `reviewRequests(first: 10) { nodes { requestedReviewer { ... on User { login } } } }`
- `PR` struct gets: `MergedAt *time.Time`, `ClosedAt *time.Time`, `ReviewRequestedUsers []string`

### `cmd/root.go`
- New flags: `flagState`, `flagSince`, `flagReviewer`
- Map `--state` value to GraphQL states (open→OPEN, merged→MERGED, closed→CLOSED, all→all three)
- `filterPRs` extended with `--since` and `--reviewer` logic
- `--reviewer @me` resolves username same way `--mine` does

### `render/render.go`
- Merged PRs render "merged Xd ago" instead of review status
- Closed PRs render "closed Xd ago"

## Composable Usage

```
gh prboard --mine                         # blocking me (open PRs I authored)
gh prboard --reviewer @me                 # I'm blocking (need my review)
gh prboard --state merged --since 1d      # what happened overnight
gh prboard --state all --since 7d --mine  # my full week of activity
```
