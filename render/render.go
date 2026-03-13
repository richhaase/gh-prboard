package render

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/richhaase/gh-prboard/github"
	"golang.org/x/term"
)

// ANSI color codes
var (
	bold    = "\033[1m"
	dim     = "\033[2m"
	reset   = "\033[0m"
	red     = "\033[31m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	cyan    = "\033[36m"
	magenta = "\033[35m"
)

func init() {
	// Disable colors if not a terminal (piped output, CI, etc.)
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		bold, dim, reset = "", "", ""
		red, green, yellow, cyan, magenta = "", "", "", "", ""
	}
}

// RenderPRs formats a list of PRs into the terminal output format.
// PRs should already be sorted by attention priority.
func RenderPRs(prs []github.PR, repoCount int) string {
	if len(prs) == 0 {
		return fmt.Sprintf("%sNo PRs found across %d watched repos.%s", dim, repoCount, reset)
	}

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
		fmt.Fprintf(&b, "%s%s%s\n", bold, g.header, reset)
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
	}

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
		return yellow + "● needs review" + reset
	case github.ReviewReReviewNeeded:
		return yellow + "↻ re-review needed" + reset
	case github.ReviewChangesRequested:
		return red + "⚠ changes requested" + reset
	case github.ReviewApproved:
		return green + "✓ approved" + reset
	default:
		return "? unknown"
	}
}

func formatCheckStatus(status github.CheckStatus) string {
	switch status {
	case github.CheckPassing:
		return green + "✓ CI" + reset
	case github.CheckFailing:
		return red + "✗ CI failing" + reset
	case github.CheckPending:
		return dim + "◌ CI pending" + reset
	default:
		return ""
	}
}

func renderSummary(prs []github.PR) string {
	needsReview := 0
	needsReReview := 0
	failingCI := 0

	for _, pr := range prs {
		if pr.State == "merged" || pr.State == "closed" {
			continue
		}
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
		parts = append(parts, fmt.Sprintf("%s%d %s review%s", yellow, needsReview, noun, reset))
	}
	if needsReReview > 0 {
		noun := "need"
		if needsReReview == 1 {
			noun = "needs"
		}
		parts = append(parts, fmt.Sprintf("%s%d %s re-review%s", yellow, needsReReview, noun, reset))
	}
	if failingCI > 0 {
		verb := "have"
		if failingCI == 1 {
			verb = "has"
		}
		parts = append(parts, fmt.Sprintf("%s%d %s failing CI%s", red, failingCI, verb, reset))
	}

	if len(parts) == 0 {
		return green + "All PRs look good." + reset
	}

	return strings.Join(parts, " · ")
}
