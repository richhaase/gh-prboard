package config

import (
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

type Repo struct {
	Name  string `yaml:"name"`
	Group string `yaml:"group,omitempty"`
}

type Defaults struct {
	HideDrafts bool   `yaml:"hide_drafts,omitempty"`
	Username   string `yaml:"username,omitempty"`
}

type Config struct {
	Orgs     []string `yaml:"orgs,omitempty"`
	Repos    []Repo   `yaml:"repos,omitempty"`
	Defaults Defaults `yaml:"defaults,omitempty"`
}

// DefaultPath returns the XDG-respecting config path (~/.config/gh-prboard/config.yml).
func DefaultPath() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "gh-prboard", "config.yml")
}

// LoadFrom reads config from the given path. Returns an empty config if the file doesn't exist.
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Load reads config from the default path.
func Load() (*Config, error) {
	return LoadFrom(DefaultPath())
}

// SaveTo writes the config to the given path, creating directories as needed.
func (c *Config) SaveTo(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Save writes the config to the default path.
func (c *Config) Save() error {
	return c.SaveTo(DefaultPath())
}

// AddRepo adds a repo or updates its group. If the repo already exists and group is empty,
// this is a no-op.
func (c *Config) AddRepo(name, group string) {
	for i, r := range c.Repos {
		if r.Name == name {
			if group != "" {
				c.Repos[i].Group = group
			}
			return
		}
	}
	c.Repos = append(c.Repos, Repo{Name: name, Group: group})
}

// RemoveRepo removes a repo by name. Returns true if the repo was found and removed.
func (c *Config) RemoveRepo(name string) bool {
	for i, r := range c.Repos {
		if r.Name == name {
			c.Repos = append(c.Repos[:i], c.Repos[i+1:]...)
			return true
		}
	}
	return false
}

// ReposByGroup returns repos organized by group. Repos without a group use "" as the key.
func (c *Config) ReposByGroup() map[string][]Repo {
	groups := make(map[string][]Repo)
	for _, r := range c.Repos {
		groups[r.Group] = append(groups[r.Group], r)
	}
	return groups
}

// RepoNames returns a sorted list of all repo names.
func (c *Config) RepoNames() []string {
	names := make([]string, len(c.Repos))
	for i, r := range c.Repos {
		names[i] = r.Name
	}
	sort.Strings(names)
	return names
}
