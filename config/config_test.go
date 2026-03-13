package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadNonexistentReturnsEmpty(t *testing.T) {
	cfg, err := LoadFrom("/tmp/gh-prboard-test-nonexistent/config.yml")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(cfg.Orgs) != 0 || len(cfg.Repos) != 0 {
		t.Fatalf("expected empty config, got orgs=%v repos=%v", cfg.Orgs, cfg.Repos)
	}
}

func TestLoadAndSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "config.yml")

	cfg := &Config{
		Orgs: []string{"org1", "org2"},
		Repos: []Repo{
			{Name: "owner/repo1", Group: "frontend"},
			{Name: "owner/repo2", Group: "backend"},
			{Name: "owner/repo3"},
		},
		Defaults: Defaults{
			HideDrafts: true,
			Username:   "testuser",
		},
	}

	if err := cfg.SaveTo(path); err != nil {
		t.Fatalf("SaveTo failed: %v", err)
	}

	loaded, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}

	if len(loaded.Orgs) != 2 || loaded.Orgs[0] != "org1" || loaded.Orgs[1] != "org2" {
		t.Errorf("orgs mismatch: %v", loaded.Orgs)
	}
	if len(loaded.Repos) != 3 {
		t.Fatalf("expected 3 repos, got %d", len(loaded.Repos))
	}
	if loaded.Repos[0].Name != "owner/repo1" || loaded.Repos[0].Group != "frontend" {
		t.Errorf("repo[0] mismatch: %+v", loaded.Repos[0])
	}
	if loaded.Repos[2].Group != "" {
		t.Errorf("repo[2] should have empty group, got %q", loaded.Repos[2].Group)
	}
	if !loaded.Defaults.HideDrafts {
		t.Error("expected HideDrafts to be true")
	}
	if loaded.Defaults.Username != "testuser" {
		t.Errorf("expected username 'testuser', got %q", loaded.Defaults.Username)
	}
}

func TestAddRepo(t *testing.T) {
	cfg := &Config{}
	cfg.AddRepo("owner/repo1", "frontend")

	if len(cfg.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(cfg.Repos))
	}
	if cfg.Repos[0].Name != "owner/repo1" || cfg.Repos[0].Group != "frontend" {
		t.Errorf("unexpected repo: %+v", cfg.Repos[0])
	}
}

func TestAddRepoDuplicateUpdatesGroup(t *testing.T) {
	cfg := &Config{}
	cfg.AddRepo("owner/repo1", "frontend")
	cfg.AddRepo("owner/repo1", "backend")

	if len(cfg.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(cfg.Repos))
	}
	if cfg.Repos[0].Group != "backend" {
		t.Errorf("expected group 'backend', got %q", cfg.Repos[0].Group)
	}
}

func TestAddRepoDuplicateNoGroupIsNoop(t *testing.T) {
	cfg := &Config{}
	cfg.AddRepo("owner/repo1", "frontend")
	cfg.AddRepo("owner/repo1", "")

	if len(cfg.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(cfg.Repos))
	}
	if cfg.Repos[0].Group != "frontend" {
		t.Errorf("expected group 'frontend' preserved, got %q", cfg.Repos[0].Group)
	}
}

func TestRemoveRepo(t *testing.T) {
	cfg := &Config{
		Repos: []Repo{
			{Name: "owner/repo1", Group: "frontend"},
			{Name: "owner/repo2", Group: "backend"},
		},
	}

	removed := cfg.RemoveRepo("owner/repo1")
	if !removed {
		t.Error("expected RemoveRepo to return true")
	}
	if len(cfg.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(cfg.Repos))
	}
	if cfg.Repos[0].Name != "owner/repo2" {
		t.Errorf("expected remaining repo 'owner/repo2', got %q", cfg.Repos[0].Name)
	}
}

func TestRemoveRepoNotFound(t *testing.T) {
	cfg := &Config{
		Repos: []Repo{{Name: "owner/repo1"}},
	}

	removed := cfg.RemoveRepo("owner/nonexistent")
	if removed {
		t.Error("expected RemoveRepo to return false for nonexistent repo")
	}
	if len(cfg.Repos) != 1 {
		t.Fatalf("expected 1 repo unchanged, got %d", len(cfg.Repos))
	}
}

func TestReposByGroup(t *testing.T) {
	cfg := &Config{
		Repos: []Repo{
			{Name: "owner/repo1", Group: "frontend"},
			{Name: "owner/repo2", Group: "backend"},
			{Name: "owner/repo3", Group: "frontend"},
			{Name: "owner/repo4"},
		},
	}

	groups := cfg.ReposByGroup()

	if len(groups["frontend"]) != 2 {
		t.Errorf("expected 2 frontend repos, got %d", len(groups["frontend"]))
	}
	if len(groups["backend"]) != 1 {
		t.Errorf("expected 1 backend repo, got %d", len(groups["backend"]))
	}
	if len(groups[""]) != 1 {
		t.Errorf("expected 1 ungrouped repo, got %d", len(groups[""]))
	}

	// Suppress unused variable warning
	_ = os.Stderr
}
