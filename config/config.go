package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v3"

	"fmt"
	"os"

	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/server/config"
	gm "github.com/charmbracelet/soft-serve/server/git"
	"github.com/go-git/go-billy/v5/memfs"
	ggit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/storage/memory"
)

var (
	// ErrNoConfig is returned when a repo has no config file.
	ErrNoConfig = errors.New("no config file found")
)

const (
	defaultConfigRepo = "config"
)

// Config is the Soft Serve configuration.
type Config struct {
	Name         string         `yaml:"name" json:"name"`
	Host         string         `yaml:"host" json:"host"`
	Port         int            `yaml:"port" json:"port"`
	AnonAccess   string         `yaml:"anon-access" json:"anon-access"`
	AllowKeyless bool           `yaml:"allow-keyless" json:"allow-keyless"`
	Users        []User         `yaml:"users" json:"users"`
	Repos        []RepoConfig   `yaml:"repos" json:"repos"`
	Source       *RepoSource    `yaml:"-" json:"-"`
	Cfg          *config.Config `yaml:"-" json:"-"`
	mtx          sync.RWMutex
}

// User contains user-level configuration for a repository.
type User struct {
	Name        string   `yaml:"name" json:"name"`
	Admin       bool     `yaml:"admin" json:"admin"`
	PublicKeys  []string `yaml:"public-keys" json:"public-keys"`
	CollabRepos []string `yaml:"collab-repos" json:"collab-repos"`
}

// RepoConfig is a repository configuration.
type RepoConfig struct {
	Name    string   `yaml:"name" json:"name"`
	Repo    string   `yaml:"repo" json:"repo"`
	Note    string   `yaml:"note" json:"note"`
	Private bool     `yaml:"private" json:"private"`
	Readme  string   `yaml:"readme" json:"readme"`
	Collabs []string `yaml:"collabs" json:"collabs"`
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
		if bts, err := os.ReadFile(k); err == nil {
			// pk is a file, set its contents as pk
			k = string(bts)
		}
		var pk = strings.TrimSpace(k)
		if pk == "" {
			continue
		}
		// it is a valid ssh key, nothing to do
		if _, _, _, _, err := ssh.ParseAuthorizedKey([]byte(pk)); err != nil {
			return nil, fmt.Errorf("invalid initial admin key %q: %w", k, err)
		}
		pks = append(pks, pk)
	}

	rs := NewRepoSource(cfg.RepoPath)
	c := &Config{
		Cfg: cfg,
	}
	c.Host = host
	c.Port = port
	c.Source = rs
	// Grant read-write access when no keys are provided.
	if len(pks) == 0 {
		anonAccess = gm.ReadWriteAccess.String()
	} else {
		anonAccess = gm.ReadOnlyAccess.String()
	}
	if host == "" {
		displayHost = "localhost"
	} else {
		displayHost = host
	}
	yamlConfig := fmt.Sprintf(defaultConfig,
		displayHost,
		port,
		anonAccess,
		len(pks) == 0,
	)
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
	return c, nil
}

// readConfig reads the config file for the repo. All config files are stored in
// the config repo.
func (cfg *Config) readConfig(repo string, v interface{}) error {
	cr, err := cfg.Source.GetRepo(defaultConfigRepo)
	if err != nil {
		return err
	}
	// Parse YAML files
	var cy string
	for _, ext := range []string{".yaml", ".yml"} {
		cy, _, err = cr.LatestFile(repo + ext)
		if err != nil && !errors.Is(err, git.ErrFileNotFound) {
			return err
		} else if err == nil {
			break
		}
	}
	// Parse JSON files
	cj, _, err := cr.LatestFile(repo + ".json")
	if err != nil && !errors.Is(err, git.ErrFileNotFound) {
		return err
	}
	if cy != "" {
		err = yaml.Unmarshal([]byte(cy), v)
		if err != nil {
			return err
		}
	} else if cj != "" {
		err = json.Unmarshal([]byte(cj), v)
		if err != nil {
			return err
		}
	} else {
		return ErrNoConfig
	}
	return nil
}

// Reload reloads the configuration.
func (cfg *Config) Reload() error {
	cfg.mtx.Lock()
	defer cfg.mtx.Unlock()
	err := cfg.Source.LoadRepos()
	if err != nil {
		return err
	}
	if err := cfg.readConfig(defaultConfigRepo, cfg); err != nil {
		return fmt.Errorf("error reading config: %w", err)
	}
	// sanitize repo configs
	repos := make(map[string]RepoConfig, 0)
	for _, r := range cfg.Repos {
		repos[r.Repo] = r
	}
	for _, r := range cfg.Source.AllRepos() {
		var rc RepoConfig
		repo := r.Repo()
		if repo == defaultConfigRepo {
			continue
		}
		if err := cfg.readConfig(repo, &rc); err != nil {
			if !errors.Is(err, ErrNoConfig) {
				log.Printf("error reading config: %v", err)
			}
			continue
		}
		repos[r.Repo()] = rc
	}
	cfg.Repos = make([]RepoConfig, 0, len(repos))
	for n, r := range repos {
		r.Repo = n
		cfg.Repos = append(cfg.Repos, r)
	}
	// Populate readmes and descriptions
	for _, r := range cfg.Source.AllRepos() {
		repo := r.Repo()
		err = r.UpdateServerInfo()
		if err != nil {
			log.Printf("error updating server info for %s: %s", repo, err)
		}
		pat := "README*"
		rp := ""
		for _, rr := range cfg.Repos {
			if repo == rr.Repo {
				rp = rr.Readme
				r.name = rr.Name
				r.description = rr.Note
				r.private = rr.Private
				break
			}
		}
		if rp != "" {
			pat = rp
		}
		rm := ""
		fc, fp, _ := r.LatestFile(pat)
		rm = fc
		if repo == "config" {
			md, err := templatize(rm, cfg)
			if err != nil {
				return err
			}
			rm = md
		}
		r.SetReadme(rm, fp)
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
	cn := defaultConfigRepo
	rp := filepath.Join(cfg.Cfg.RepoPath, cn) + ".git"
	rs := cfg.Source
	err := rs.LoadRepo(cn)
	if errors.Is(err, fs.ErrNotExist) {
		repo, err := ggit.PlainInit(rp, true)
		if err != nil {
			return err
		}
		repo, err = ggit.Clone(memory.NewStorage(), memfs.New(), &ggit.CloneOptions{
			URL: rp,
		})
		if err != nil && err != transport.ErrEmptyRemoteRepository {
			return err
		}
		wt, err := repo.Worktree()
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
		_, err = wt.Add("README.md")
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
		_, err = wt.Add("config.yaml")
		if err != nil {
			return err
		}
		author := object.Signature{
			Name:  "Soft Serve Server",
			Email: "vt100@charm.sh",
			When:  time.Now(),
		}
		_, err = wt.Commit("Default init", &ggit.CommitOptions{
			All:       true,
			Author:    &author,
			Committer: &author,
		})
		if err != nil {
			return err
		}
		err = repo.Push(&ggit.PushOptions{})
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return cfg.Reload()
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
