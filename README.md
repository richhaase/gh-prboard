# gh-prboard

A GitHub CLI extension that shows open PRs needing attention across your watched repositories.

## Install

```
gh extension install richhaase/gh-prboard
```

Requires [GitHub CLI](https://cli.github.com/) with `gh auth login` completed.

## Quick Start

```
gh prboard init
```

The setup wizard discovers your repos, lets you pick which to watch, and saves the config. Then run:

```
gh prboard
```

## Commands

### `gh prboard`

Show open PRs across watched repos, sorted by attention priority.

```
gh prboard                    # show all PRs
gh prboard --mine             # show only your PRs
gh prboard --needs-review     # show PRs needing review
gh prboard --author octocat   # filter by author
gh prboard --group backend    # filter by repo group
gh prboard --include-drafts   # include draft PRs (hidden by default)
```

### `gh prboard init`

Interactive setup wizard. Discovers your personal repos and orgs, filters by recent activity (90 days), and generates the config file.

If a config already exists, offers to overwrite, merge, or cancel.

### `gh prboard repos`

Manage watched repos individually.

```
gh prboard repos list                    # list watched repos
gh prboard repos add owner/repo         # add a repo
gh prboard repos add owner/repo --group backend  # add with group
gh prboard repos remove owner/repo      # remove a repo
gh prboard repos discover               # discover repos from configured orgs
gh prboard repos discover some-org      # discover repos from a specific org
```

## Config

Stored at `~/.config/gh-prboard/config.yml` (respects `$XDG_CONFIG_HOME`).

```yaml
orgs:
  - my-org
repos:
  - name: me/my-project
  - name: my-org/api
    group: backend
  - name: my-org/web
    group: frontend
defaults:
  username: me
  hide_drafts: true
```

## PR Status

PRs are sorted by attention priority and display:

- **Review status**: needs review, re-review needed, changes requested, approved
- **CI status**: passing, failing, pending
- **Age**: time since PR was opened

Colors are used in terminal output and automatically disabled when piped.
