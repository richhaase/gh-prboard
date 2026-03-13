package render

import (
	"strings"
	"testing"
	"time"

	"github.com/richhaase/gh-prboard/github"
)

func TestRenderPRs_GroupedOutput(t *testing.T) {
	now := time.Now()
	prs := []github.PR{
		{
			Repo:           "teamsense/platform",
			RepoGroup:      "backend",
			Number:         142,
			Title:          "Add webhook retry logic",
			Author:         "maria",
			CreatedAt:      now.Add(-3 * 24 * time.Hour),
			LatestCommitAt: now.Add(-4 * 24 * time.Hour),
			Reviews:        []github.Review{},
			Checks:         github.CheckPassing,
		},
	}

	output := RenderPRs(prs, 2)

	if !strings.Contains(output, "platform (backend)") {
		t.Errorf("expected group header, got:\n%s", output)
	}
	if !strings.Contains(output, "#142") {
		t.Errorf("expected PR number, got:\n%s", output)
	}
	if !strings.Contains(output, "@maria") {
		t.Errorf("expected author, got:\n%s", output)
	}
	if !strings.Contains(output, "needs review") {
		t.Errorf("expected review status, got:\n%s", output)
	}
}

func TestRenderPRs_SummaryLine(t *testing.T) {
	now := time.Now()
	prs := []github.PR{
		{Repo: "a/one", Number: 1, CreatedAt: now, Reviews: []github.Review{}, Checks: github.CheckPassing},
		{Repo: "a/two", Number: 2, CreatedAt: now, Reviews: []github.Review{}, Checks: github.CheckFailing},
	}

	output := RenderPRs(prs, 2)

	if !strings.Contains(output, "2 PRs need review") {
		t.Errorf("expected summary with count, got:\n%s", output)
	}
	if !strings.Contains(output, "1 has failing CI") {
		t.Errorf("expected CI failure count, got:\n%s", output)
	}
}

func TestRenderPRs_UngroupedReposShowAsOther(t *testing.T) {
	now := time.Now()
	prs := []github.PR{
		{Repo: "rdh/personal", Number: 1, CreatedAt: now, Reviews: []github.Review{}},
	}

	output := RenderPRs(prs, 2)

	if !strings.Contains(output, "personal") {
		t.Errorf("expected repo name in output, got:\n%s", output)
	}
}

func TestRenderPRs_Empty(t *testing.T) {
	output := RenderPRs(nil, 5)
	if !strings.Contains(output, "No PRs found") {
		t.Errorf("expected empty message, got:\n%s", output)
	}
}

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

func TestRenderPRs_AgeFormatting(t *testing.T) {
	now := time.Now()
	prs := []github.PR{
		{Repo: "a/one", Number: 1, CreatedAt: now.Add(-2 * time.Hour), Reviews: []github.Review{}},
		{Repo: "a/two", Number: 2, CreatedAt: now.Add(-3 * 24 * time.Hour), Reviews: []github.Review{}},
	}

	output := RenderPRs(prs, 2)

	if !strings.Contains(output, "2h") {
		t.Errorf("expected '2h' for 2-hour-old PR, got:\n%s", output)
	}
	if !strings.Contains(output, "3d") {
		t.Errorf("expected '3d' for 3-day-old PR, got:\n%s", output)
	}
}
