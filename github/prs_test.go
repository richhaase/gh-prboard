package github

import (
	"testing"
	"time"
)

func TestClassifyReviewStatus_NoReviews(t *testing.T) {
	pr := &PR{
		Reviews: []Review{},
	}
	status := pr.ReviewStatus()
	if status != ReviewNone {
		t.Errorf("expected ReviewNone, got %v", status)
	}
}

func TestClassifyReviewStatus_Approved(t *testing.T) {
	pr := &PR{
		LatestCommitAt: time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC),
		Reviews: []Review{
			{State: "APPROVED", SubmittedAt: time.Date(2026, 3, 11, 12, 0, 0, 0, time.UTC)},
		},
	}
	status := pr.ReviewStatus()
	if status != ReviewApproved {
		t.Errorf("expected ReviewApproved, got %v", status)
	}
}

func TestClassifyReviewStatus_ChangesRequested(t *testing.T) {
	pr := &PR{
		LatestCommitAt: time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC),
		Reviews: []Review{
			{State: "CHANGES_REQUESTED", SubmittedAt: time.Date(2026, 3, 11, 12, 0, 0, 0, time.UTC)},
		},
	}
	status := pr.ReviewStatus()
	if status != ReviewChangesRequested {
		t.Errorf("expected ReviewChangesRequested, got %v", status)
	}
}

func TestClassifyReviewStatus_ReReviewNeeded(t *testing.T) {
	pr := &PR{
		LatestCommitAt: time.Date(2026, 3, 12, 12, 0, 0, 0, time.UTC),
		Reviews: []Review{
			{State: "APPROVED", SubmittedAt: time.Date(2026, 3, 11, 12, 0, 0, 0, time.UTC)},
		},
	}
	status := pr.ReviewStatus()
	if status != ReviewReReviewNeeded {
		t.Errorf("expected ReviewReReviewNeeded, got %v", status)
	}
}

func TestClassifyReviewStatus_ReReviewAfterChangesRequested(t *testing.T) {
	pr := &PR{
		LatestCommitAt: time.Date(2026, 3, 12, 12, 0, 0, 0, time.UTC),
		Reviews: []Review{
			{State: "CHANGES_REQUESTED", SubmittedAt: time.Date(2026, 3, 11, 12, 0, 0, 0, time.UTC)},
		},
	}
	status := pr.ReviewStatus()
	if status != ReviewReReviewNeeded {
		t.Errorf("expected ReviewReReviewNeeded, got %v", status)
	}
}

func TestClassifyReviewStatus_MultipleReviewsUsesLatest(t *testing.T) {
	pr := &PR{
		LatestCommitAt: time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC),
		Reviews: []Review{
			{State: "CHANGES_REQUESTED", SubmittedAt: time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)},
			{State: "APPROVED", SubmittedAt: time.Date(2026, 3, 11, 12, 0, 0, 0, time.UTC)},
		},
	}
	status := pr.ReviewStatus()
	if status != ReviewApproved {
		t.Errorf("expected ReviewApproved, got %v", status)
	}
}

func TestAttentionPriority(t *testing.T) {
	if ReviewNone.Priority() >= ReviewReReviewNeeded.Priority() {
		t.Error("needs review should have higher priority (lower number) than re-review")
	}
	if ReviewReReviewNeeded.Priority() >= ReviewChangesRequested.Priority() {
		t.Error("re-review should have higher priority than changes requested")
	}
	if ReviewChangesRequested.Priority() >= ReviewApproved.Priority() {
		t.Error("changes requested should have higher priority than approved")
	}
}

func TestSortPRsByAttention(t *testing.T) {
	now := time.Now()
	prs := []PR{
		{Number: 1, CreatedAt: now.Add(-1 * time.Hour), Reviews: []Review{{State: "APPROVED", SubmittedAt: now}}, LatestCommitAt: now.Add(-2 * time.Hour)},
		{Number: 2, CreatedAt: now.Add(-3 * time.Hour), Reviews: []Review{}},
		{Number: 3, CreatedAt: now.Add(-2 * time.Hour), Reviews: []Review{}},
	}

	sorted := SortByAttention(prs)

	// #2 should be first (needs review, oldest)
	if sorted[0].Number != 2 {
		t.Errorf("expected PR #2 first, got #%d", sorted[0].Number)
	}
	// #3 next (needs review, newer)
	if sorted[1].Number != 3 {
		t.Errorf("expected PR #3 second, got #%d", sorted[1].Number)
	}
	// #1 last (approved)
	if sorted[2].Number != 1 {
		t.Errorf("expected PR #1 last, got #%d", sorted[2].Number)
	}
}
