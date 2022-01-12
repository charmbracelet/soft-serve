package config

import (
	"github.com/charmbracelet/soft-serve/pkg/webhooks"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"

	"fmt"
	"os"

	"github.com/charmbracelet/soft-serve/config"
	"github.com/charmbracelet/soft-serve/internal/git"
	gg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// Config is the Soft Serve configuration.
type Config struct {
	Name         string   `yaml:"name"`
	Host         string   `yaml:"host"`
	Port         int      `yaml:"port"`
	AnonAccess   string   `yaml:"anon-access"`
	AllowKeyless bool     `yaml:"allow-keyless"`
	Users        []User   `yaml:"users"`
	Repos        []Repo   `yaml:"repos"`
	Webhooks     Webhooks `yaml:"webhooks"`
	Source       *git.RepoSource
	Cfg          *config.Config
}

// User contains user-level configuration for a repository.
type User struct {
	Name        string   `yaml:"name"`
	Admin       bool     `yaml:"admin"`
	PublicKeys  []string `yaml:"public-keys"`
	CollabRepos []string `yaml:"collab-repos"`
}

// Repo contains repository configuration information.
type Repo struct {
	Name    string `yaml:"name"`
	Repo    string `yaml:"repo"`
	Note    string `yaml:"note"`
	Private bool   `yaml:"private"`
}

// Webhooks contains infos about the incoming webhooks that are served
type Webhooks struct {
	// TLSCertificatePath is an optional setting that points to the path with the tls
	// certificate used to serve the webhooks. Make sure soft-serve can read that file.
	TLSCertificatePath string `yaml:"tls-certificate-path"`

	// TLSKeyPath is an optional setting that points to the path with the tls
	// key used to serve the webhooks. Make sure soft-serve can read that file.
	TLSKeyPath string `yaml:"tls-key-path"`

	PluginPath string               `yaml:"plugin-path"`
	Routes     webhooks.RouteSpec[] `yaml:"routes"`

	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// NewConfig creates a new internal Config struct.
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

// Reload reloads the configuration.
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
		cr, err := rs.InitRepo(cn, true)
		if err != nil {
			return err
		}
		wt, err := cr.Repository.Worktree()
		if err != nil {
			return err
		}
		rm, err := wt.Filesystem.Create("README.md")
		if err != nil {
			return err
		}
		_, err = rm.Write([]byte(defaultReadme))
		if err != nil {
			return err
		}
		cf, err := wt.Filesystem.Create("config.yaml")
		if err != nil {
			return err
		}
		_, err = cf.Write([]byte(yaml))
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
		err = cr.Repository.Push(&gg.PushOptions{})
		if err != nil {
			return err
		}
		cmd := exec.Command("git", "update-server-info")
		cmd.Dir = filepath.Join(rs.Path, cn)
		err = cmd.Run()
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
