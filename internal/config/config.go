package config

import (
	"strings"

	"gopkg.in/yaml.v2"

	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/soft/config"
	"github.com/charmbracelet/soft/internal/git"
	gg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type Config struct {
	Name         string `yaml:"name"`
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	AnonAccess   string `yaml:"anon-access"`
	AllowKeyless bool   `yaml:"allow-keyless"`
	Users        []User `yaml:"users"`
	Repos        []Repo `yaml:"repos"`
	Source       *git.RepoSource
	Cfg          *config.Config
}

type User struct {
	Name        string   `yaml:"name"`
	Admin       bool     `yaml:"admin"`
	PublicKeys  []string `yaml:"public-keys"`
	CollabRepos []string `yaml:"collab-repos"`
}

type Repo struct {
	Name    string `yaml:"name"`
	Repo    string `yaml:"repo"`
	Note    string `yaml:"note"`
	Private bool   `yaml:"private"`
}

func NewConfig(cfg *config.Config) (*Config, error) {
	var anonAccess string
	var yamlUsers string
	var displayHost string
	host := cfg.Host
	port := cfg.Port
	pk := cfg.InitialAdminKey
	rs := git.NewRepoSource(cfg.RepoPath)
	c := &Config{
		Cfg: cfg,
	}
	c.Host = cfg.Host
	c.Port = port
	c.Source = rs
	if pk == "" {
		anonAccess = "read-write"
	} else {
		anonAccess = "no-access"
	}
	if host == "" {
		displayHost = "localhost"
	} else {
		displayHost = host
	}
	yamlConfig := fmt.Sprintf(defaultConfig, displayHost, port, anonAccess)
	if pk != "" {
		pks := ""
		for _, key := range strings.Split(strings.TrimSpace(pk), "\n") {
			pks += fmt.Sprintf("      - %s\n", key)
		}
		yamlUsers = fmt.Sprintf(hasKeyUserConfig, pks)
	} else {
		yamlUsers = defaultUserConfig
	}
	yaml := fmt.Sprintf("%s%s%s", yamlConfig, yamlUsers, exampleUserConfig)
	err := c.createDefaultConfigRepo(yaml)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (cfg *Config) Reload() error {
	err := cfg.Source.LoadRepos()
	if err != nil {
		return err
	}
	cr, err := cfg.Source.GetRepo("config")
	if err != nil {
		return err
	}
	cs, err := cr.LatestFile("config.yaml")
	if err != nil {
		return err
	}
	err = yaml.Unmarshal([]byte(cs), cfg)
	if err != nil {
		return fmt.Errorf("bad yaml in config.yaml: %s", err)
	}
	return nil
}

func createFile(path string, content string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	if err != nil {
		return err
	}
	return f.Sync()
}

func (cfg *Config) createDefaultConfigRepo(yaml string) error {
	cn := "config"
	rs := cfg.Source
	err := rs.LoadRepos()
	if err != nil {
		return err
	}
	_, err = rs.GetRepo(cn)
	if err == git.ErrMissingRepo {
		cr, err := rs.InitRepo(cn, false)
		if err != nil {
			return err
		}

		rp := filepath.Join(rs.Path, cn, "README.md")
		err = createFile(rp, defaultReadme)
		if err != nil {
			return err
		}
		cp := filepath.Join(rs.Path, cn, "config.yaml")
		err = createFile(cp, yaml)
		if err != nil {
			return err
		}
		wt, err := cr.Repository.Worktree()
		if err != nil {
			return err
		}
		_, err = wt.Add("README.md")
		if err != nil {
			return err
		}
		_, err = wt.Add("config.yaml")
		if err != nil {
			return err
		}
		_, err = wt.Commit("Default init", &gg.CommitOptions{
			All: true,
			Author: &object.Signature{
				Name:  "Soft Serve Server",
				Email: "vt100@charm.sh",
			},
		})
		if err != nil {
			return err
		}
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return cfg.Reload()
}

func (cfg *Config) isPrivate(repo string) bool {
	for _, r := range cfg.Repos {
		if r.Repo == repo {
			return r.Private
		}
	}
	return false
}
