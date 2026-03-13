# gh-prboard

GitHub CLI extension that shows open PRs needing attention across watched repos.

## Build & Test

```bash
make build          # build binary to bin/gh-prboard
make test           # run tests
make test-race      # run tests with race detector
make lint           # run golangci-lint (v2 config)
make check          # run all checks (fmt, vet, lint, test)
```

Single test: `go test ./cmd/... -run TestParseNumber -v`

## Project Structure

```
main.go              # entry point, version ldflags injection
cmd/                  # cobra commands
  root.go             # main PR display with filters (--mine, --group, etc.)
  init.go             # interactive setup wizard
  input.go            # shared input parsing (ParseNumberSelection, PromptLine, FormatRelativeTime)
  repos*.go           # repo management subcommands (add, remove, list, discover)
  version.go          # version command
config/               # YAML config at ~/.config/gh-prboard/config.yml
github/               # GraphQL API layer
  prs.go              # PR fetching, review status classification, attention sorting
  repos.go            # org/user repo discovery, FilterByAge
  user.go             # FetchUsername, FetchOrgs
render/               # terminal output formatting with ANSI colors
```

## Key Patterns

- GraphQL client: always `*api.GraphQLClient` (pointer), obtained from `api.DefaultGraphQLClient()`
- Config: `config.Load()` returns `*Config`, returns empty config (not error) if file missing
- Interactive I/O: `bufio.Scanner` via `cmd/input.go`, no TUI library
- Colors: ANSI escape codes in `render/`, auto-disabled when stdout is not a terminal
- Module: `github.com/richhaase/gh-prboard`
- Release: push a `v*` tag; `cli/gh-extension-precompile@v2` builds binaries with ldflags version injection

## Lint

golangci-lint v2 format. gosec exclusions: G104, G115, G204, G301, G302, G304, G306, G404.
