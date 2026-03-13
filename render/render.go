package render

import (
	"fmt"
	"strings"
	"time"

	"github.com/richhaase/gh-prboard/github"
)

// RenderPRs formats a list of PRs into the terminal output format.
// PRs should already be sorted by attention priority.
func RenderPRs(prs []github.PR, repoCount int) string {
	if len(prs) == 0 {
		return fmt.Sprintf("No open PRs across %d watched repos.", repoCount)
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
