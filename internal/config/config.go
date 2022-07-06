package config

import (
	"bytes"
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

	"github.com/charmbracelet/soft-serve/config"
	"github.com/charmbracelet/soft-serve/internal/git"
	"github.com/go-git/go-billy/v5/memfs"
	ggit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/storage/memory"
)

// Config is the Soft Serve configuration.
type Config struct {
	Name         string          `yaml:"name"`
	Host         string          `yaml:"host"`
	Port         int             `yaml:"port"`
	AnonAccess   string          `yaml:"anon-access"`
	AllowKeyless bool            `yaml:"allow-keyless"`
	Users        []User          `yaml:"users"`
	Repos        []Repo          `yaml:"repos"`
	Source       *git.RepoSource `yaml:"-"`
	Cfg          *config.Config  `yaml:"-"`
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
	cs, _, err := cr.LatestFile("config.yaml")
	if err != nil {
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
		fc, fp, _ := r.LatestFile(pat)
		rm = fc
		if name == "config" {
			md, err := templatize(rm, cfg)
			if err != nil {
				return err
			}
			rm = md
		}
		r.SetReadme(rm, fp)
		err := cfg.createHooks(r)
		if err != nil {
			return err
		}
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
	rp := filepath.Join(cfg.Cfg.RepoPath, cn)
	rs := cfg.Source
	err := rs.LoadRepo(cn)
	if errors.Is(err, fs.ErrNotExist) {
		log.Printf("creating default config repo %s", cn)
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
		author := &object.Signature{
			Name:  "Soft Serve Server",
			Email: "vt100@charm.sh",
			When:  time.Now(),
		}
		_, err = wt.Commit("Default init", &ggit.CommitOptions{
			All:    true,
			Author: author,
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

type hookScript struct {
	Executable string
	Hook       string
	Args       string
	Envs       []string
}

var hookTmpl *template.Template

func (cfg *Config) createHooks(repo *git.Repo) error {
	if hookTmpl == nil {
		var err error
		hookTmpl, err = template.New("hook").Parse(`#!/usr/bin/env bash
# AUTO GENERATED BY SOFT SERVE, DO NOT MODIFY
{{ range $_, $env := .Envs }}
{{ $env }} \{{ end }}
{{ .Executable }} internal hook {{ .Hook }} {{ .Args }}
`)
		if err != nil {
			return err
		}
	}

	err := ensureDir(filepath.Join(repo.Path(), "hooks"))
	if err != nil {
		return err
	}
	ex, err := os.Executable()
	if err != nil {
		return err
	}
	rp, err := filepath.Abs(cfg.Cfg.RepoPath)
	if err != nil {
		return err
	}
	kp, err := filepath.Abs(cfg.Cfg.KeyPath)
	if err != nil {
		return err
	}
	ikp, err := filepath.Abs(cfg.Cfg.InternalKeyPath)
	if err != nil {
		return err
	}
	envs := []string{
		fmt.Sprintf("SOFT_SERVE_BIND_ADDRESS=%s", cfg.Cfg.BindAddr),
		fmt.Sprintf("SOFT_SERVE_PORT=%d", cfg.Cfg.Port),
		fmt.Sprintf("SOFT_SERVE_HOST=%s", cfg.Cfg.Host),
		fmt.Sprintf("SOFT_SERVE_REPO_PATH=%s", rp),
		fmt.Sprintf("SOFT_SERVE_KEY_PATH=%s", kp),
		fmt.Sprintf("SOFT_SERVE_INTERNAL_KEY_PATH=%s", ikp),
	}
	for _, hook := range []string{"pre-receive", "update", "post-receive"} {
		var data bytes.Buffer
		var args string
		hp := filepath.Join(repo.Path(), "hooks", hook)
		if hook == "update" {
			args = "$1 $2 $3"
		}
		err = hookTmpl.Execute(&data, hookScript{
			Executable: ex,
			Hook:       hook,
			Args:       args,
			Envs:       envs,
		})
		if err != nil {
			return err
		}
		err = os.WriteFile(hp, data.Bytes(), 0755) //nolint:gosec
		if err != nil {
			return err
		}
	}

	return nil
}

func ensureDir(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return os.MkdirAll(path, 0755)
	}
	return err
}
