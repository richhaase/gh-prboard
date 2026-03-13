# gh-prwatch Design Spec

A GitHub CLI extension that shows open PRs needing attention across watched repositories. A discovery tool, not a review tool — find what needs attention, then use `gh` or the browser to act on it.

## CLI Interface

Installed as a `gh` extension. Invoked as `gh prwatch`.

### Commands

```
gh prwatch                          # show PRs needing attention across watched repos
gh prwatch repos list               # list watched repos with group labels
gh prwatch repos add owner/repo [--group <name>]  # add a repo to watch
gh prwatch repos remove owner/repo                 # remove a repo from watch list
gh prwatch repos discover           # discover repos from configured orgs, pick which to watch
```

### Flags (main command)

```
gh prwatch --group <name>           # filter to a specific repo group
gh prwatch --author <username>      # filter by PR author
gh prwatch --mine                   # show only PRs you authored
gh prwatch --needs-review           # show only PRs with status: needs review, re-review needed, or changes requested
gh prwatch --include-drafts         # include draft PRs (hidden by default)
```

### Output Format

Structured markdown-like output, human-readable and easily parseable by LLMs.

```
## ts-platform (backend)

  #142  Add webhook retry logic          @maria   3d   ⚠ changes requested   ✓ CI
  #138  Fix rate limiting on /api/send   @jose    5d   ● needs review         ✓ CI

## teamsense (frontend)

  #891  Dashboard redesign v2            @rich    1d   ✓ approved             ✓ CI
  #887  Update i18n strings              @sarah   2d   ↻ re-review needed     ✗ CI failing

1 PR needs review · 1 needs re-review · 1 has failing CI
```

Review status indicators:
- `● needs review` — no reviews yet
- `⚠ changes requested` — reviewer requested changes
- `✓ approved` — approved and ready to merge
- `↻ re-review needed` — reviewed previously, but author pushed new commits since the last review

CI status indicators:
- `✓ CI` — all checks passing
- `✗ CI failing` — one or more checks failing
- `◌ CI pending` — checks still running

Summary line at the bottom aggregates counts across all repos.

## Configuration

Config file: `~/.config/gh-prwatch/config.yml` (XDG-respecting). One file per machine — different machines get different configs naturally.

```yaml
# Orgs to use for repo discovery
orgs:
  - teamsense

# Watched repos, optionally grouped
repos:
  - name: teamsense/ts-platform
    group: backend
  - name: teamsense/teamsense
    group: frontend
  - name: teamsense/ts-platform-url-shortener
    group: backend
  - name: rdh/personal-project
    # no group — renders under "Other" heading

# Optional defaults
defaults:
  hide_drafts: true
  username: rdh  # auto-detected from gh auth if omitted
```

### Config decisions

- **Groups are optional.** Ungrouped repos render under an "Other" heading.
- **`orgs` is only used by `discover`.** The main command only reads the `repos` list.
- **No per-repo config beyond name and group.** Flat and simple.
- **`username`** is needed for `--mine` and reviewer detection. Auto-detected from `gh auth status` if not set.

## Architecture

Go module, installed as a precompiled `gh` extension using the [go-gh](https://github.com/cli/go-gh) library for auth and API access.

### Packages

**`cmd/`** — CLI entry point using [cobra](https://github.com/spf13/cobra). Parses args and flags, calls into other packages, formats and renders output to stdout.

**`config/`** — Reads and writes the YAML config file. Handles XDG path resolution. Provides typed Go structs matching the config shape.

**`github/`** — All GitHub API interaction via `go-gh`. Two main operations:

1. **FetchPRs** — GraphQL query taking the watched repo list, returning open PRs with: title, number, author, created date, review decisions, check suite status, draft flag, latest commit timestamp, and latest review timestamp.

2. **DiscoverRepos** — GraphQL query listing all non-archived repos in configured orgs, sorted by `pushedAt`. Returns name and last push date.

### Data Flow (main command)

```
config.Load() → github.FetchPRs(repos) → filter by flags → sort by attention priority → render to stdout
```

### Sort Order

PRs are sorted by attention priority within each repo group:
1. `● needs review` — no one has looked at it yet (highest priority)
2. `↻ re-review needed` — has new commits since last review
3. `⚠ changes requested` — waiting on author, but visible for awareness
4. `✓ approved` — ready to merge, lowest priority

Secondary sort within each priority level: oldest first (longest-waiting PRs surface higher).

### Empty States and Errors

- **No config file:** Print a getting-started message pointing to `gh prwatch repos add` and `gh prwatch repos discover`.
- **No watched repos:** Same helpful message.
- **No open PRs:** Print "No open PRs across N watched repos." — not an error.
- **Auth failure:** Surface `gh`'s auth error and suggest `gh auth login`.
- **`repos add` for an already-watched repo:** Update its group if `--group` is provided, otherwise no-op.

### Re-review Detection

Compare the timestamp of the most recent review on a PR with the timestamp of the most recent commit on the PR's head branch. If the latest commit is newer than the latest review, the status is `↻ re-review needed`. This catches the case where Arc and similar tools would show a PR as "already reviewed" when it actually has new changes.

### Rate Limiting and Pagination

- GraphQL can fetch approximately 30 repos per query due to GitHub's node limits. For most users this is one API call.
- If someone watches 60+ repos, batch into multiple queries.
- Cap at 10 open PRs per repo (sane default, configurable if needed).

## Discover Flow

`gh prwatch repos discover` is the only interactive part of the tool.

1. Query all non-archived repos from configured `orgs` via GraphQL.
2. Sort by `pushedAt` descending (most recently active first).
3. Display the list with activity timestamps, marking currently watched repos.

```
Repos in teamsense (32 found):

  [watching]  teamsense/ts-platform              pushed 2d ago
  [watching]  teamsense/teamsense                pushed 3d ago
  [        ]  teamsense/ts-messaging-django      pushed 1w ago
  [        ]  teamsense/ts-analytics             pushed 3w ago
  [        ]  teamsense/old-migration-tool       pushed 8mo ago
  ...

Add/remove repos? (enter repo names, prefix with - to remove)
> teamsense/ts-messaging-django
> -teamsense/ts-platform
> done
```

- Text-based prompting. No TUI, no arrow keys. Type names, prefix with `-` to unwatch, `done` to finish.
- Config file is updated in place.
- Groups are not assigned during discover. Use `gh prwatch repos add <repo> --group <name>` or edit the YAML directly.
- Dead repos are visually obvious at the bottom of the recency-sorted list. No automatic pruning.

## Non-Goals

- **Not a review tool.** No diffs, no comments, no inline review. Use `gh` or the browser for that.
- **No TUI / interactive dashboard.** Output prints and exits.
- **No JSON output.** Structured text is better for both humans and LLMs.
- **No multi-provider support.** GitHub only.
- **No notification system.** This is pull-based — you run it when you want to see the state.

## Language & Distribution

- **Language:** Go
- **GitHub library:** [cli/go-gh](https://github.com/cli/go-gh) for auth token access, API helpers
- **CLI framework:** [spf13/cobra](https://github.com/spf13/cobra)
- **Config parsing:** Standard Go YAML library
- **Distribution:** Precompiled binaries attached to GitHub releases. Installed via `gh extension install`.
