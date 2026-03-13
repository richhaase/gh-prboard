# gh-prboard init — Guided Setup Wizard

## Purpose

Provide a zero-to-working first-run experience so users never need to manually edit the config file. Running `gh prboard init` walks through org selection, repo discovery, grouping, and config generation interactively.

## Changes

### New: `cmd/init.go` — `gh prboard init` command

Interactive wizard with this flow:

1. **Auto-detect username** — call `github.FetchUsername()` via GraphQL, set `defaults.username` silently. Print: `Detected GitHub user: <username>`

2. **Discover orgs** — call `github.FetchOrgs()` via GraphQL. Present numbered list, user enters numbers (comma-separated or space-separated) to select. Example:
   ```
   Found 3 organizations:
     1. teamsense
     2. acme-corp
     3. old-project

   Select orgs to watch (e.g. 1,3): 1,2
   ```

3. **Discover repos** — call existing `github.DiscoverRepos()` for selected orgs, then `github.FilterByAge()` to keep only repos pushed within the last 90 days. Present numbered list per org with last-push time. Cap display at 50 repos per org (sorted by most recently pushed). User enters numbers to select. Example:
   ```
   Found 12 active repos in teamsense (pushed in last 90 days):
     1. teamsense/platform          pushed 2h ago
     2. teamsense/mobile-app        pushed 1d ago
     3. teamsense/docs              pushed 5d ago
     ...

   Select repos to watch (e.g. 1-3,5 or all): 1,2
   ```

4. **Group assignment** — for each selected repo, ask what group to assign. Enter to skip (ungrouped). Show previously used group names as numbered shortcuts. Example:
   ```
   Assign groups to repos (enter to skip):
     teamsense/platform group: backend
     teamsense/mobile-app group [1=backend]: frontend
     teamsense/docs group [1=backend, 2=frontend]:
   ```

5. **Write config** — save to `config.DefaultPath()` (`~/.config/gh-prboard/config.yml`). Include selected orgs, repos with groups, and `defaults.username` + `defaults.hide_drafts: true`.

6. **Confirm and run** — print summary, then execute the main PR fetch-and-display as a best-effort step. Config is already saved at this point, so a fetch failure doesn't lose setup work. Example:
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
- **Merge:** add newly selected orgs and repos to existing config. For repos already in config, preserve existing group (matches `config.AddRepo` behavior — only updates group if new group is non-empty). Preserves existing `defaults` values.
- **Cancel:** exit

### Modified: `cmd/root.go` — no-config message and refactor

Change the no-config message from listing manual commands to:
```
No repos configured. Run `gh prboard init` to get started.
```

Remove `resolveUsername()` from `root.go` — replaced by `github.FetchUsername()`.

### Modified: `cmd/repos_discover.go` — accept org argument

Change `Args` to `cobra.MaximumNArgs(1)`. Behavior:
- **With arg** (`gh prboard repos discover teamsense`): discover repos for that org directly, show the interactive add/remove loop. Does not require pre-existing config — creates/updates config file.
- **Without arg**: use orgs from config (existing behavior). Errors if no orgs configured with a message pointing to `init`.

### New: `github/user.go` — user and org discovery

Two new functions:
- `FetchUsername(client *api.GraphQLClient) (string, error)` — returns `viewer.login`. Replaces the duplicate `resolveUsername` in `cmd/root.go`.
- `FetchOrgs(client *api.GraphQLClient) ([]string, error)` — returns org logins the user belongs to via `viewer { organizations(first: 100) { nodes { login } } }`.

### Modified: `github/repos.go` — time filtering

Add `FilterByAge(repos []DiscoveredRepo, maxAge time.Duration) []DiscoveredRepo` — returns only repos where `time.Since(r.PushedAt) <= maxAge`.

### New: `cmd/input.go` — shared input parsing utilities

Extract input helpers used by both `init.go` and `repos_discover.go`:
- `ParseNumberSelection(input string, max int) ([]int, error)` — parse `1-3,5,7` into `[1,2,3,5,7]`
- `PromptLine(prompt string) (string, error)` — read a single line from stdin
- `FormatRelativeTime(t time.Time) string` — consolidate duplicate relative time formatting from `repos_discover.go`

## Input Parsing

Number selection supports:
- Single numbers: `1,3,5`
- Ranges: `1-5`
- Mixed: `1-3,7,9`
- `all` to select everything
- Spaces or commas as separators
- Out-of-range numbers produce an error message and re-prompt

## Interactive I/O

Uses `bufio.Scanner` for line-based input — no TUI library dependency. This matches the existing `repos_discover.go` pattern and keeps the dependency footprint small. Acceptable trade-off: no cursor control or inline validation, but the wizard flow is simple enough that line-by-line input works well.

## Error Handling

- If GitHub auth fails: print "Not authenticated. Run `gh auth login` first." and exit
- If user belongs to no orgs: skip org step, suggest adding repos manually via `gh prboard repos add owner/repo`
- If no repos found in selected orgs within 90 days: mention the age filter and suggest `gh prboard repos add` for specific repos
- If final PR fetch fails after config save: print warning but don't fail — config is already saved

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

- Unit test `FilterByAge` with repos at various ages (within/outside 90 days, edge cases)
- Unit test `ParseNumberSelection` — single numbers, ranges, mixed, `all`, out-of-range, invalid input
- Unit test merge behavior — adding repos that already exist with/without group changes
- Integration: manual `gh prboard init` walkthrough
