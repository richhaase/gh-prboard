package github

import (
	"testing"
	"time"
)

func TestFilterByAge_AllRecent(t *testing.T) {
	now := time.Now()
	repos := []DiscoveredRepo{
		{FullName: "org/recent1", PushedAt: now.Add(-1 * 24 * time.Hour)},
		{FullName: "org/recent2", PushedAt: now.Add(-10 * 24 * time.Hour)},
	}
	filtered := FilterByAge(repos, 90*24*time.Hour)
	if len(filtered) != 2 {
		t.Errorf("expected 2 repos, got %d", len(filtered))
	}
}

func TestFilterByAge_SomeOld(t *testing.T) {
	now := time.Now()
	repos := []DiscoveredRepo{
		{FullName: "org/recent", PushedAt: now.Add(-30 * 24 * time.Hour)},
		{FullName: "org/old", PushedAt: now.Add(-120 * 24 * time.Hour)},
	}
	filtered := FilterByAge(repos, 90*24*time.Hour)
	if len(filtered) != 1 {
		t.Errorf("expected 1 repo, got %d", len(filtered))
	}
	if filtered[0].FullName != "org/recent" {
		t.Errorf("expected org/recent, got %s", filtered[0].FullName)
	}
}

func TestFilterByAge_AllOld(t *testing.T) {
	now := time.Now()
	repos := []DiscoveredRepo{
		{FullName: "org/old1", PushedAt: now.Add(-100 * 24 * time.Hour)},
		{FullName: "org/old2", PushedAt: now.Add(-200 * 24 * time.Hour)},
	}
	filtered := FilterByAge(repos, 90*24*time.Hour)
	if len(filtered) != 0 {
		t.Errorf("expected 0 repos, got %d", len(filtered))
	}
}

func TestFilterByAge_Empty(t *testing.T) {
	filtered := FilterByAge(nil, 90*24*time.Hour)
	if len(filtered) != 0 {
		t.Errorf("expected 0 repos, got %d", len(filtered))
	}
}
