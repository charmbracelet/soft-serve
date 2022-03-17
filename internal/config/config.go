package config

import (
	"bytes"
	"log"
	"strings"
	"sync"
	"text/template"

	"golang.org/x/crypto/ssh"
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
	Name         string `yaml:"name"`
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	AnonAccess   string `yaml:"anon-access"`
	AllowKeyless bool   `yaml:"allow-keyless"`
	Users        []User `yaml:"users"`
	Repos        []Repo `yaml:"repos"`
	Source       *git.RepoSource
	Cfg          *config.Config
	mtx          sync.Mutex
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
	Readme  string `yaml:"readme"`
}

// NewConfig creates a new internal Config struct.
func NewConfig(cfg *config.Config) (*Config, error) {
	var anonAccess string
	var yamlUsers string
	var displayHost string
	host := cfg.Host
	port := cfg.Port

	pks := make([]string, 0)
	for _, k := range cfg.InitialAdminKeys {
		var pk = strings.TrimSpace(k)
		if pk != "" {
			if bts, err := os.ReadFile(k); err == nil {
				// pk is a file, set its contents as pk
				pk = string(bts)
			}
			// it is a valid ssh key, nothing to do
			if _, _, _, _, err := ssh.ParseAuthorizedKey([]byte(pk)); err != nil {
				return nil, fmt.Errorf("invalid initial admin key %q: %w", k, err)
			}
			pks = append(pks, pk)
		}
	}

	rs := git.NewRepoSource(cfg.RepoPath)
	c := &Config{
		Cfg: cfg,
	}
	c.Host = cfg.Host
	c.Port = port
	c.Source = rs
	if len(pks) == 0 {
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
	if len(pks) == 0 {
		yamlUsers = defaultUserConfig
	} else {
		var result string
		for _, pk := range pks {
			result += fmt.Sprintf("      - %s\n", pk)
		}
		yamlUsers = fmt.Sprintf(hasKeyUserConfig, result)
	}
	yaml := fmt.Sprintf("%s%s%s", yamlConfig, yamlUsers, exampleUserConfig)
	err := c.createDefaultConfigRepo(yaml)
	if err != nil {
		return nil, err
	}
	err = c.Reload()
	if err != nil {
		return nil, err
	}
	return c, nil
}

// Reload reloads the configuration.
func (cfg *Config) Reload() error {
	cfg.mtx.Lock()
	defer cfg.mtx.Unlock()
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
		log.Print("here")
		return err
	}
	err = yaml.Unmarshal([]byte(cs), cfg)
	if err != nil {
		return fmt.Errorf("bad yaml in config.yaml: %s", err)
	}
	for _, r := range cfg.Source.AllRepos() {
		name := r.Name()
		err = r.UpdateServerInfo()
		if err != nil {
			log.Printf("error updating server info for %s: %s", name, err)
		}
		pat := "README*"
		rp := ""
		for _, rr := range cfg.Repos {
			if name == rr.Repo {
				rp = rr.Readme
				break
			}
		}
		if rp != "" {
			pat = rp
		}
		rm := ""
		fs, err := r.LsFiles(pat)
		if err != nil {
			return err
		}
		if len(fs) > 0 {
			fc, err := r.LatestFile(fs[0])
			if err != nil {
				return err
			}
			rm = fc
			r.ReadmePath = fs[0]
		}
		if name == "config" {
			md, err := templatize(rm, cfg)
			if err != nil {
				return err
			}
			rm = md
		}
		r.Readme = rm
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
	err := rs.LoadRepo(cn)
	if err == gg.ErrRepositoryNotExists {
		cr, err := rs.InitRepo(cn, true)
		if err != nil {
			return err
		}
		wt, err := cr.Repository().Worktree()
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
		err = cr.Repository().Push(&gg.PushOptions{})
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}

func (cfg *Config) isPrivate(repo string) bool {
	for _, r := range cfg.Repos {
		if r.Repo == repo {
			return r.Private
		}
	}
	return false
}

func templatize(mdt string, tmpl interface{}) (string, error) {
	t, err := template.New("readme").Parse(mdt)
	if err != nil {
		return "", err
	}
	buf := &bytes.Buffer{}
	err = t.Execute(buf, tmpl)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
