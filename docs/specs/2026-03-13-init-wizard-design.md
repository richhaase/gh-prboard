# gh-prboard init — Guided Setup Wizard

## Purpose

Provide a zero-to-working first-run experience so users never need to manually edit the config file. Running `gh prboard init` walks through org selection, repo discovery, grouping, and config generation interactively.

## Changes

### New: `cmd/init.go` — `gh prboard init` command

Interactive wizard with this flow:

1. **Auto-detect username** — call `viewer { login }` via GraphQL, set `defaults.username` silently. Print: `Detected GitHub user: <username>`

2. **Discover orgs** — call `viewer { organizations(first: 100) { nodes { login } } }` via GraphQL. Present numbered list, user enters numbers (comma-separated or space-separated) to select. Example:
   ```
   Found 3 organizations:
     1. teamsense
     2. acme-corp
     3. old-project

   Select orgs to watch (e.g. 1,3): 1,2
   ```

3. **Discover repos** — call existing `github.DiscoverRepos()` for selected orgs. Filter to repos with `pushedAt` within the last 90 days. Present numbered list with last-push time. User enters numbers to select. Example:
   ```
   Found 12 active repos in teamsense (pushed in last 90 days):
     1. teamsense/platform          pushed 2h ago
     2. teamsense/mobile-app        pushed 1d ago
     3. teamsense/docs              pushed 5d ago
     ...

   Select repos to watch (e.g. 1-3,5): 1,2
   ```

4. **Group assignment** — for each selected repo, ask what group to assign. Offer a "skip" option for ungrouped. Allow reusing previously entered group names via numbered shortcut. Example:
   ```
   Assign groups to repos (enter to skip):
     teamsense/platform group: backend
     teamsense/mobile-app group: frontend
   ```

5. **Write config** — save to `config.DefaultPath()` (`~/.config/gh-prboard/config.yml`). Include selected orgs, repos with groups, and `defaults.username` + `defaults.hide_drafts: true`.

6. **Confirm and run** — print summary, then execute the main PR fetch-and-display flow so the user immediately sees results. Example:
   ```
   Config saved to ~/.config/gh-prboard/config.yml
   Watching 2 repos across 1 org.

   Fetching PRs...

   ## platform (backend)
     #142  Add webhook retry logic  @maria  3d  ● needs review  ✓ CI
   ...
   ```

**Re-run behavior:** If config already exists, prompt:
```
Config already exists at ~/.config/gh-prboard/config.yml
Overwrite, merge, or cancel? [o/m/c]:
```
- **Overwrite:** replace entirely with new wizard output
- **Merge:** add newly selected orgs/repos to existing config (skip duplicates)
- **Cancel:** exit

### Modified: `cmd/root.go` — no-config message

Change the no-config message from listing manual commands to:
```
No repos configured. Run `gh prboard init` to get started.
```

### Modified: `cmd/repos_discover.go` — accept org argument

Allow `gh prboard repos discover [org]` so it works without pre-existing config:
- With arg: discover repos for that org directly
- Without arg: use orgs from config (existing behavior)

### New: `github/user.go` — user and org discovery

Two new functions:
- `FetchUsername(client) (string, error)` — returns `viewer.login`
- `FetchOrgs(client) ([]string, error)` — returns org logins the user belongs to

### Modified: `github/repos.go` — time filtering

Add a `FilterByAge(repos []DiscoveredRepo, maxAge time.Duration) []DiscoveredRepo` function that filters repos by `pushedAt` recency.

## Input Parsing

Number selection supports:
- Single numbers: `1,3,5`
- Ranges: `1-5`
- Mixed: `1-3,7,9`
- `all` to select everything
- Spaces or commas as separators

## Error Handling

- If `gh auth status` fails: print "Not authenticated. Run `gh auth login` first." and exit
- If user belongs to no orgs: skip org step, ask to add repos manually via `repos add`
- If no repos found in selected orgs: inform user, suggest `repos add` for personal repos
- If all discovered repos are older than 90 days: mention the filter and suggest `repos add` for specific repos

## Config File Output

```yaml
orgs:
  - teamsense
repos:
  - name: teamsense/platform
    group: backend
  - name: teamsense/mobile-app
    group: frontend
defaults:
  username: rdh
  hide_drafts: true
```

## Testing

- Unit test `FilterByAge` with repos at various ages
- Unit test number-range parsing (`1-3,5` → `[1,2,3,5]`)
- Integration: manual `gh prboard init` walkthrough
